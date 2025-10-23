package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

type Telemetry struct {
	cfg *Options

	lp *sdklog.LoggerProvider
	mp *sdkmetric.MeterProvider
	tp *sdktrace.TracerProvider

	logger otellog.Logger
	tracer trace.Tracer

	// Prometheus-specific fields
	promServer  *http.Server
	promHandler http.Handler
}

// Shutdown shuts down the logger, meter, and tracer.
// It forces a flush of all pending telemetry data before shutting down.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var err error

	// Shutdown Prometheus HTTP server first
	if t.promServer != nil {
		if shutdownErr := t.promServer.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown Prometheus server: %w", shutdownErr)
		}
	}

	// Force flush and shutdown logger provider
	if t.lp != nil {
		if flushErr := t.lp.ForceFlush(ctx); flushErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to flush logs: %w", err, flushErr)
			} else {
				err = fmt.Errorf("failed to flush logs: %w", flushErr)
			}
		}
		if shutdownErr := t.lp.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown logs: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown logs: %w", shutdownErr)
			}
		}
	}

	// Force flush and shutdown meter provider
	if t.mp != nil {
		if flushErr := t.mp.ForceFlush(ctx); flushErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to flush metrics: %w", err, flushErr)
			} else {
				err = fmt.Errorf("failed to flush metrics: %w", flushErr)
			}
		}
		if shutdownErr := t.mp.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown metrics: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown metrics: %w", shutdownErr)
			}
		}
	}

	// Force flush and shutdown tracer provider
	if t.tp != nil {
		if flushErr := t.tp.ForceFlush(ctx); flushErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to flush traces: %w", err, flushErr)
			} else {
				err = fmt.Errorf("failed to flush traces: %w", flushErr)
			}
		}
		if shutdownErr := t.tp.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown traces: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown traces: %w", shutdownErr)
			}
		}
	}

	return err
}

// Logger returns the OTel logger.
func (t *Telemetry) Logger() otellog.Logger {
	return t.logger
}

// Tracer returns the tracer.
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// LoggerProvider returns the logger otel logger provider.
// Returns nil if OTel logs are disabled.
func (t *Telemetry) LoggerProvider() *sdklog.LoggerProvider {
	return t.lp
}

// MeterProvider returns the meter otel meter provider.
// Returns nil if OTel metrics are disabled.
func (t *Telemetry) MeterProvider() *sdkmetric.MeterProvider {
	return t.mp
}

// TracerProvider returns the tracer otel tracer provider.
// Returns nil if OTel traces are disabled.
func (t *Telemetry) TracerProvider() *sdktrace.TracerProvider {
	return t.tp
}

// StartSpan starts a new span with the given name. The span must be ended by calling End.
// The returned context contains the span information which will be automatically extracted
// by the logger's OTel integration (for supported loggers like Zap, Zerolog, Logrus, Slog).
func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

// PrometheusHandler returns the Prometheus HTTP handler for metrics.
// Returns nil if Prometheus metrics are not enabled.
// Use this to integrate Prometheus metrics into your own HTTP server.
func (t *Telemetry) PrometheusHandler() http.Handler {
	return t.promHandler
}

// ServiceName returns the configured service name.
func (t *Telemetry) ServiceName() string {
	if t.cfg == nil {
		return ""
	}
	return t.cfg.ServiceName
}

// ServiceVersion returns the configured service version.
func (t *Telemetry) ServiceVersion() string {
	if t.cfg == nil {
		return ""
	}
	return t.cfg.ServiceVersion
}

// New creates a new Telemetry instance with the given options.
// If opts is nil, default options with environment variable overrides are used.
func New(ctx context.Context, opts *Options) (*Telemetry, error) {
	// Use defaults if no options provided
	if opts == nil {
		opts = DefaultOptions()
	}

	// Apply environment variable overrides
	opts.applyEnvVars()

	return newWithOptions(ctx, opts)
}

// newWithOptions creates a new Telemetry instance with the given options.
func newWithOptions(ctx context.Context, opts *Options) (*Telemetry, error) {
	var lp *sdklog.LoggerProvider
	var mp *sdkmetric.MeterProvider
	var tp *sdktrace.TracerProvider
	var logger otellog.Logger
	var tracer trace.Tracer
	var promServer *http.Server
	var promHandler http.Handler
	var err error

	// Create resource if OTel is enabled (auto-detected from environment)
	// or if metrics exporter is explicitly configured
	var res *resource.Resource
	metricsExporterSet := opts.MetricsExporter != "" || os.Getenv("OTEL_METRICS_EXPORTER") != ""
	if shouldEnableOTel() || metricsExporterSet {
		res = newResource(opts.ServiceName, opts.ServiceVersion)
	}

	// Initialize providers conditionally based on environment variables
	lp, err = newLoggerProvider(ctx, res, opts.BatchExport)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger provider: %w", err)
	}

	if lp != nil {
		logger = lp.Logger(opts.ServiceName)
	} else {
		// Use noop logger if logs are disabled (default OTel behavior)
		logger = lognoop.NewLoggerProvider().Logger(opts.ServiceName)
	}

	tp, err = newTracerProvider(ctx, res, opts.BatchExport)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	if tp != nil {
		tracer = tp.Tracer(opts.ServiceName)
	} else {
		// Use noop tracer if traces are disabled (default OTel behavior)
		tracer = tracenoop.NewTracerProvider().Tracer(opts.ServiceName)
	}

	// Initialize meter provider based on exporter type
	// Check if metrics exporter is explicitly set in options or environment
	exporter := opts.MetricsExporter
	if exporter == "" {
		exporter = os.Getenv("OTEL_METRICS_EXPORTER")
	}

	// Determine if we should enable metrics
	enableMetrics := false
	if exporter != "" && exporter != "none" {
		// Explicitly configured via options or env var
		enableMetrics = true
	} else if shouldEnableMetrics() {
		// Auto-enabled via OTel environment variables
		enableMetrics = true
		exporter = "otlp" // Default to OTLP
	}

	if enableMetrics {
		// Support multiple exporters via comma-separated list (e.g., "prometheus,otlp")
		exportersList := strings.Split(exporter, ",")
		var readers []sdkmetric.Reader

		for _, exp := range exportersList {
			exp = strings.TrimSpace(exp)
			if exp == "" || exp == "none" {
				continue
			}

			switch exp {
			case "prometheus":
				var handler http.Handler
				var promReader sdkmetric.Reader
				promReader, handler, err = newPrometheusReader(res)
				if err != nil {
					return nil, fmt.Errorf("failed to create Prometheus reader: %w", err)
				}
				readers = append(readers, promReader)

				// Store handler for external use (only first Prometheus exporter)
				if promHandler == nil {
					promHandler = handler
				}

				// Only start built-in server if explicitly enabled and not already started
				if opts.PrometheusServer && promServer == nil {
					// Start Prometheus HTTP server
					mux := http.NewServeMux()
					mux.Handle(opts.PrometheusPath, handler)

					promServer = &http.Server{
						Addr:    ":" + strconv.Itoa(opts.PrometheusPort),
						Handler: mux,
					}

					// Start server in background
					go func() {
						if err := promServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
							fmt.Fprintf(os.Stderr, "Prometheus server error: %v\n", err)
						}
					}()
				}

			case "otlp":
				otlpReader, err := newOTLPReader(ctx, opts.BatchExport)
				if err != nil {
					return nil, fmt.Errorf("failed to create OTLP reader: %w", err)
				}
				readers = append(readers, otlpReader)

			default:
				return nil, fmt.Errorf("unsupported metrics exporter: %s (supported: otlp, prometheus, none)", exp)
			}
		}

		// Create meter provider with all readers
		if len(readers) > 0 {
			meterProviderOptions := []sdkmetric.Option{sdkmetric.WithResource(res)}
			for _, reader := range readers {
				meterProviderOptions = append(meterProviderOptions, sdkmetric.WithReader(reader))
			}
			mp = sdkmetric.NewMeterProvider(meterProviderOptions...)
			otel.SetMeterProvider(mp)
		}
	}

	return &Telemetry{
		cfg:         opts,
		lp:          lp,
		mp:          mp,
		tp:          tp,
		logger:      logger,
		tracer:      tracer,
		promServer:  promServer,
		promHandler: promHandler,
	}, nil
}
