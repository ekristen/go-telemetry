package telemetry

import (
	"context"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// ITelemetry is the interface for the telemetry system.
type ITelemetry interface {
	// Shutdown shuts down all telemetry providers.
	Shutdown(ctx context.Context)

	// Tracer returns the tracer instance.
	Tracer() trace.Tracer

	// LoggerProvider returns the OTel logger provider (nil if OTel is disabled).
	LoggerProvider() *sdklog.LoggerProvider
	// MeterProvider returns the OTel meter provider (nil if OTel is disabled).
	MeterProvider() *sdkmetric.MeterProvider
	// TracerProvider returns the OTel tracer provider (nil if OTel is disabled).
	TracerProvider() *sdktrace.TracerProvider

	// StartSpan starts a new span with the given name.
	// The returned context contains the span information which will be automatically extracted
	// by the logger's OTel integration.
	StartSpan(ctx context.Context, name string) (context.Context, trace.Span)
}
