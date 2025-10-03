package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// newLoggerProvider creates a new logger provider with the OTLP gRPC exporter.
// Returns nil if logs are disabled via environment variables.
func newLoggerProvider(ctx context.Context, res *resource.Resource, batchExport bool) (*log.LoggerProvider, error) {
	if !shouldEnableLogs() {
		return nil, nil
	}

	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Choose processor based on batchExport option
	var processor log.Processor
	if batchExport {
		// BatchProcessor for higher throughput, lower resource usage (with latency)
		processor = log.NewBatchProcessor(exporter)
	} else {
		// SimpleProcessor for immediate export without delays
		processor = log.NewSimpleProcessor(exporter)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(res),
	)

	return lp, nil
}

// newMeterProvider creates a new meter provider with the OTLP gRPC exporter.
// Returns nil if metrics are disabled via environment variables.
// Deprecated: Use newOTLPReader instead for better composability.
func newMeterProvider(ctx context.Context, res *resource.Resource, batchExport bool) (*metric.MeterProvider, error) {
	if !shouldEnableMetrics() {
		return nil, nil
	}

	reader, err := newOTLPReader(ctx, batchExport)
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return mp, nil
}

// newOTLPReader creates an OTLP metric reader with the gRPC exporter.
// Returns a Reader that can be used with a MeterProvider.
func newOTLPReader(ctx context.Context, batchExport bool) (metric.Reader, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Note: Metrics use PeriodicReader by default which is always batched.
	// The batchExport flag doesn't significantly affect metrics since they're
	// inherently periodic/batched by design. We keep the parameter for consistency.
	reader := metric.NewPeriodicReader(exporter)
	return reader, nil
}

// newPrometheusReader creates a Prometheus metric reader and HTTP handler.
// Returns the Reader and an HTTP handler for the /metrics endpoint.
func newPrometheusReader(res *resource.Resource) (metric.Reader, http.Handler, error) {
	// Create a Prometheus registry
	registry := prometheus.NewRegistry()

	// Create Prometheus exporter with the registry
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create HTTP handler from the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	return exporter, handler, nil
}

// newTracerProvider creates a new tracer provider with the OTLP gRPC exporter.
// Returns nil if traces are disabled via environment variables.
func newTracerProvider(ctx context.Context, res *resource.Resource, batchExport bool) (*trace.TracerProvider, error) {
	if !shouldEnableTraces() {
		return nil, nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	var tp *trace.TracerProvider
	if batchExport {
		// Use batcher for batched export (default OTel behavior)
		tp = trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(res),
		)
	} else {
		// Use syncer for immediate export
		tp = trace.NewTracerProvider(
			trace.WithSyncer(exporter),
			trace.WithResource(res),
		)
	}

	otel.SetTracerProvider(tp)

	return tp, nil
}

// newResource creates a new OTEL resource with the service name and version.
func newResource(serviceName string, serviceVersion string) *resource.Resource {
	hostName, _ := os.Hostname()

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.HostName(hostName),
	)
}
