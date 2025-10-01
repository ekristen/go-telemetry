package telemetry

import (
	"context"

	"github.com/ekristen/go-telemetry/logger"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ITelemetry is the interface for the telemetry system.
type ITelemetry interface {
	// Shutdown shuts down all telemetry providers.
	Shutdown(ctx context.Context)

	// Logger returns the logger instance.
	Logger() logger.Logger
	// Tracer returns the tracer instance.
	Tracer() trace.Tracer

	// LoggerProvider returns the OTel logger provider (nil if OTel is disabled).
	LoggerProvider() *sdklog.LoggerProvider
	// MeterProvider returns the OTel meter provider (nil if OTel is disabled).
	MeterProvider() *sdkmetric.MeterProvider
	// TracerProvider returns the OTel tracer provider (nil if OTel is disabled).
	TracerProvider() *sdktrace.TracerProvider

	// StartSpan starts a new span with the given name.
	StartSpan(ctx context.Context, name string) (context.Context, trace.Span)
	// StartSpanWithLogger starts a new span and returns a logger with the span context.
	StartSpanWithLogger(ctx context.Context, name string) (context.Context, trace.Span, logger.Logger)
}
