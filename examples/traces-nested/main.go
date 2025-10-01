package main

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/ekristen/go-telemetry"
)

func main() {
	ctx := context.Background()

	// Create telemetry with OTel traces enabled
	// Set OTEL_EXPORTER_OTLP_ENDPOINT to enable OTel:
	//   export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "traces-nested-example",
		ServiceVersion: "1.0.0",
		BatchExport:    false,
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	logger := t.Logger()
	logger.Info().Msg("Starting traces nested example")

	// Start root span
	ctx, rootSpan := t.StartSpan(ctx, "process-order")
	rootSpan.SetAttributes(
		attribute.String("order.id", "12345"),
		attribute.Int("order.items", 3),
	)

	logger.Info().Str("span", "root").Msg("Processing order")

	// Nest child spans under the root span
	validateOrder(ctx, t)
	chargePayment(ctx, t)
	fulfillOrder(ctx, t)

	rootSpan.SetStatus(codes.Ok, "Order processed successfully")
	rootSpan.End()

	logger.Info().Msg("Order processing complete")
}

// validateOrder demonstrates a nested span with attributes
func validateOrder(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "validate-order")
	defer span.End()

	span.SetAttributes(
		attribute.String("validation.type", "inventory"),
		attribute.Bool("validation.passed", true),
	)

	logger := t.Logger()
	logger.Info().Str("span", "validate").Msg("Validating order")

	time.Sleep(50 * time.Millisecond)

	// Add span event
	span.AddEvent("Inventory checked")
}

// chargePayment demonstrates nested spans with multiple levels
func chargePayment(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "charge-payment")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment.method", "credit_card"),
		attribute.Float64("payment.amount", 99.99),
	)

	logger := t.Logger()
	logger.Info().Str("span", "payment").Msg("Charging payment")

	// Nested span for authorization
	authorizePayment(ctx, t)

	// Nested span for capture
	capturePayment(ctx, t)

	span.AddEvent("Payment processed")
}

// authorizePayment is a deeply nested span (third level)
func authorizePayment(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "authorize-payment")
	defer span.End()

	span.SetAttributes(
		attribute.String("auth.provider", "stripe"),
		attribute.String("auth.status", "approved"),
	)

	logger := t.Logger()
	logger.Info().Str("span", "authorize").Msg("Authorizing payment")

	time.Sleep(30 * time.Millisecond)
}

// capturePayment is another deeply nested span (third level)
func capturePayment(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "capture-payment")
	defer span.End()

	span.SetAttributes(
		attribute.String("capture.id", "ch_abc123"),
		attribute.Bool("capture.success", true),
	)

	logger := t.Logger()
	logger.Info().Str("span", "capture").Msg("Capturing payment")

	time.Sleep(20 * time.Millisecond)
}

// fulfillOrder demonstrates error handling in spans
func fulfillOrder(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "fulfill-order")
	defer span.End()

	span.SetAttributes(
		attribute.String("warehouse", "west-1"),
		attribute.String("shipping.method", "express"),
	)

	logger := t.Logger()
	logger.Info().Str("span", "fulfill").Msg("Fulfilling order")

	// Nested span for packing
	packOrder(ctx, t)

	// Nested span for shipping
	shipOrder(ctx, t)

	span.SetStatus(codes.Ok, "Order fulfilled")
}

// packOrder demonstrates span with custom events
func packOrder(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span := t.StartSpan(ctx, "pack-order")
	defer span.End()

	logger := t.Logger()
	logger.Info().Str("span", "pack").Msg("Packing order")

	span.AddEvent("Started packing")
	time.Sleep(40 * time.Millisecond)

	span.AddEvent("Items packed")
	span.SetAttributes(
		attribute.String("box.size", "medium"),
		attribute.Int("items.count", 3),
	)
}

// shipOrder demonstrates using StartSpanWithLogger
func shipOrder(ctx context.Context, t *telemetry.Telemetry) {
	ctx, span, logger := t.StartSpanWithLogger(ctx, "ship-order")
	defer span.End()

	span.SetAttributes(
		attribute.String("carrier", "USPS"),
		attribute.String("tracking", "1Z999AA10123456784"),
	)

	// Logger has span context attached automatically
	logger.Info().Str("carrier", "USPS").Msg("Shipping order")

	time.Sleep(30 * time.Millisecond)

	span.AddEvent("Label printed")
	logger.Info().Msg("Tracking number generated")
}
