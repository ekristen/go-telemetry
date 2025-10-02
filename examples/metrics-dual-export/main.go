package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ekristen/go-telemetry"
)

func main() {
	ctx := context.Background()

	// Create telemetry with BOTH Prometheus and OTLP exporters
	// Metrics will be available via:
	// 1. Prometheus HTTP endpoint at http://localhost:9090/metrics
	// 2. Pushed to OTLP collector (if OTEL_EXPORTER_OTLP_ENDPOINT is set)
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:      "metrics-dual-export-example",
		ServiceVersion:   "1.0.0",
		MetricsExporter:  "prometheus,otlp", // Comma-separated list!
		PrometheusPort:   9090,
		PrometheusPath:   "/metrics",
		PrometheusServer: true, // Enable the built-in HTTP server
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	logger := t.Logger()
	logger.Info().Msg("Starting dual export metrics example")
	logger.Info().Msg("Metrics are being exported to:")
	logger.Info().Msg("  1. Prometheus HTTP: http://localhost:9090/metrics")

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		logger.Info().Str("endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")).Msg("  2. OTLP Collector")
	} else {
		logger.Warn().Msg("  2. OTLP Collector: DISABLED (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		logger.Warn().Msg("     Example: export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317")
	}

	// Get the meter provider
	mp := t.MeterProvider()
	if mp == nil {
		logger.Error().Msg("Metrics provider is nil - this should not happen")
		return
	}

	// Create a meter for this component
	meter := mp.Meter("example-meter")

	// Counter: Monotonically increasing value
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}

	// Histogram: Distribution of values
	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_milliseconds",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}

	// UpDownCounter: Can increase or decrease
	activeConnections, err := meter.Int64UpDownCounter(
		"http_connections_active",
		metric.WithDescription("Number of active HTTP connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		panic(err)
	}

	// Gauge (via Observable): Current value at observation time
	_, err = meter.Int64ObservableGauge(
		"system_memory_usage_bytes",
		metric.WithDescription("Current memory usage"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
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

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	logger.Info().Msg("Simulating application workload... (Press Ctrl+C to stop)")
	logger.Info().Msg("")
	logger.Info().Msg("To view metrics:")
	logger.Info().Msg("  Prometheus: curl http://localhost:9090/metrics")
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		logger.Info().Msg("  OTLP: Check your collector/backend (Jaeger, Grafana, etc.)")
	}
	logger.Info().Msg("")

	endpoints := []string{"/api/users", "/api/orders", "/api/products"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []int{200, 201, 400, 404, 500}

	// Run workload until interrupted
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	requestCount := 0

	for {
		select {
		case <-sigCh:
			logger.Info().Msg("Received shutdown signal")
			logger.Info().Int("total_requests", requestCount).Msg("Processed requests")
			return

		case <-ticker.C:
			// Simulate an HTTP request
			endpoint := endpoints[rand.Intn(len(endpoints))]
			method := methods[rand.Intn(len(methods))]
			statusCode := statusCodes[rand.Intn(len(statusCodes))]
			duration := float64(rand.Intn(500)) + 10.0 // 10-510ms

			// Record metrics with attributes
			attrs := metric.WithAttributes(
				attribute.String("http_method", method),
				attribute.String("http_route", endpoint),
				attribute.Int("http_status_code", statusCode),
			)

			// Increment request counter
			requestCounter.Add(ctx, 1, attrs)

			// Record request duration histogram
			requestDuration.Record(ctx, duration, attrs)

			// Simulate connection lifecycle
			if rand.Intn(5) == 0 {
				activeConnections.Add(ctx, 1, metric.WithAttributes(
					attribute.String("protocol", "http"),
				))
			}
			if rand.Intn(7) == 0 {
				activeConnections.Add(ctx, -1, metric.WithAttributes(
					attribute.String("protocol", "http"),
				))
			}

			requestCount++

			logger.Debug().
				Str("method", method).
				Str("endpoint", endpoint).
				Int("status", statusCode).
				Float64("duration_ms", duration).
				Msg("Request processed")

			// Occasional reminder
			if requestCount%25 == 0 {
				fmt.Printf("\n📊 Processed %d requests - metrics available at both endpoints\n\n", requestCount)
			}
		}
	}
}
