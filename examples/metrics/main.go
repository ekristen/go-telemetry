package main

import (
	"context"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ekristen/go-telemetry/v2"
)

func main() {
	ctx := context.Background()

	// Create telemetry with OTel metrics enabled
	// Set OTEL_EXPORTER_OTLP_ENDPOINT to enable OTel:
	//   export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "metrics-example",
		ServiceVersion: "1.0.0",
		BatchExport:    true, // Use batching for metrics (recommended)
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	logger := t.Logger()
	logger.Info().Msg("Starting metrics example")

	// Get the meter provider
	mp := t.MeterProvider()
	if mp == nil {
		logger.Warn().Msg("Metrics are disabled - set OTEL_EXPORTER_OTLP_ENDPOINT to enable")
		return
	}

	// Create a meter for this component
	meter := mp.Meter("example-meter")

	// Counter: Monotonically increasing value (e.g., total requests)
	requestCounter, err := meter.Int64Counter(
		"http.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}

	// UpDownCounter: Can increase or decrease (e.g., active connections)
	activeConnections, err := meter.Int64UpDownCounter(
		"http.connections.active",
		metric.WithDescription("Number of active HTTP connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		panic(err)
	}

	// Histogram: Distribution of values (e.g., request duration)
	requestDuration, err := meter.Float64Histogram(
		"http.request.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}

	// Gauge (via Observable): Current value at observation time (e.g., memory usage)
	_, err = meter.Int64ObservableGauge(
		"system.memory.usage",
		metric.WithDescription("Current memory usage"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			// In real code, you'd read actual memory stats
			memoryUsage := int64(rand.Intn(1000000000))
			observer.Observe(memoryUsage, metric.WithAttributes(
				attribute.String("type", "heap"),
			))
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}

	// Simulate application workload
	logger.Info().Msg("Simulating application workload...")

	endpoints := []string{"/api/users", "/api/orders", "/api/products"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []int{200, 201, 400, 404, 500}

	for i := 0; i < 50; i++ {
		// Simulate an HTTP request
		endpoint := endpoints[rand.Intn(len(endpoints))]
		method := methods[rand.Intn(len(methods))]
		statusCode := statusCodes[rand.Intn(len(statusCodes))]
		duration := float64(rand.Intn(500)) + 10.0 // 10-510ms

		// Record metrics with attributes
		attrs := metric.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.route", endpoint),
			attribute.Int("http.status_code", statusCode),
		)

		// Increment request counter
		requestCounter.Add(ctx, 1, attrs)

		// Record request duration histogram
		requestDuration.Record(ctx, duration, attrs)

		// Simulate connection lifecycle
		if i%5 == 0 {
			activeConnections.Add(ctx, 1, metric.WithAttributes(
				attribute.String("protocol", "http"),
			))
		}
		if i%7 == 0 {
			activeConnections.Add(ctx, -1, metric.WithAttributes(
				attribute.String("protocol", "http"),
			))
		}

		logger.Info().
			Str("method", method).
			Str("endpoint", endpoint).
			Int("status", statusCode).
			Float64("duration_ms", duration).
			Msg("Request processed")

		time.Sleep(100 * time.Millisecond)
	}

	logger.Info().Msg("Workload complete - metrics will be flushed on shutdown")

	// Force flush to ensure all metrics are sent
	if err := mp.ForceFlush(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to flush metrics")
	}

	logger.Info().Msg("Metrics example complete")
}
