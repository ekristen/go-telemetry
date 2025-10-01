package telemetry

import (
	"context"
	"fmt"
	"io"
	"os"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ekristen/go-telemetry/logger"
	zerologger "github.com/ekristen/go-telemetry/logger/zerolog"
)

type Telemetry struct {
	cfg *Options

	lp *sdklog.LoggerProvider
	mp *sdkmetric.MeterProvider
	tp *sdktrace.TracerProvider

	tracer trace.Tracer
	logger logger.Logger
}

// Shutdown shuts down the logger, meter, and tracer.
// It forces a flush of all pending telemetry data before shutting down.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var err error

	// Force flush and shutdown logger provider
	if t.lp != nil {
		if flushErr := t.lp.ForceFlush(ctx); flushErr != nil {
			err = fmt.Errorf("failed to flush logs: %w", flushErr)
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

// Logger returns the logger.
func (t *Telemetry) Logger() logger.Logger {
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
func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name)
}

// StartSpanWithLogger starts a new span with the given name and returns the context, span, and logger with the span context.
func (t *Telemetry) StartSpanWithLogger(ctx context.Context, name string) (context.Context, trace.Span, logger.Logger) {
	ctx, span := t.tracer.Start(ctx, name)
	logger := t.logger.WithContext(ctx)
	return ctx, span, logger
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
	var tracer trace.Tracer
	var err error

	// Create resource if OTel is enabled (auto-detected from environment)
	var res *resource.Resource
	if shouldEnableOTel() {
		res = newResource(opts.ServiceName, opts.ServiceVersion)
	}

	// Initialize providers conditionally based on environment variables
	lp, err = newLoggerProvider(ctx, res, opts.BatchExport)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger provider: %w", err)
	}

	tp, err = newTracerProvider(ctx, res, opts.BatchExport)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	if tp != nil {
		tracer = tp.Tracer(opts.ServiceName)
	} else {
		// Use noop tracer if traces are disabled (default OTel behavior)
		tracer = noop.NewTracerProvider().Tracer(opts.ServiceName)
	}

	mp, err = newMeterProvider(ctx, res, opts.BatchExport)
	if err != nil {
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}

	// Use provided logger or create default zerolog logger
	var log logger.Logger
	if opts.Logger != nil {
		log = opts.Logger

		// Update logger with service name and version if it supports it
		if optUpdater, ok := log.(logger.LoggerOptionsUpdater); ok {
			optUpdater.SetOptions(opts.ServiceName, opts.ServiceVersion)
		}

		// If logger was provided, update it with the OTel logger provider
		if lp != nil {
			if providerUpdater, ok := log.(logger.LoggerProviderUpdater); ok {
				providerUpdater.UpdateLoggerProvider(lp)
			}
		}
	} else {
		// Create default zerolog logger
		var output io.Writer = os.Stdout
		if opts.LogConsoleOutput {
			cw := zerologger.NewConsoleWriter(opts.LogConsoleColor)
			output = cw
		}

		log = zerologger.New(zerologger.Options{
			ServiceName:    opts.ServiceName,
			ServiceVersion: opts.ServiceVersion,
			LoggerProvider: lp,
			Output:         output,
			EnableCaller:   true,
			EnableColor:    opts.LogConsoleColor,
		})
	}

	return &Telemetry{
		cfg:    opts,
		lp:     lp,
		mp:     mp,
		tp:     tp,
		tracer: tracer,
		logger: log,
	}, nil
}
