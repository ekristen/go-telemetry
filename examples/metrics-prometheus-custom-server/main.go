package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
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

	// Create telemetry with Prometheus metrics exporter
	// PrometheusServer defaults to false, so we use our own HTTP server
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:     "metrics-prometheus-custom-server",
		ServiceVersion:  "1.0.0",
		MetricsExporter: "prometheus",
		// PrometheusServer: false is the default - built-in server is OFF
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	logger := t.Logger()
	logger.Info().Msg("Starting Prometheus metrics example with custom HTTP server")

	// Get the Prometheus handler
	promHandler := t.PrometheusHandler()
	if promHandler == nil {
		logger.Error().Msg("Prometheus handler is nil - this should not happen")
		return
	}

	// Create your own HTTP server with custom routes
	mux := http.NewServeMux()

	// Add your application routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from custom HTTP server!\n")
		fmt.Fprintf(w, "Visit /metrics for Prometheus metrics\n")
		fmt.Fprintf(w, "Visit /health for health check\n")
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK\n")
	})

	// Register the Prometheus metrics handler at /metrics
	mux.Handle("/metrics", promHandler)

	// Start your custom HTTP server
	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	go func() {
		logger.Info().Str("addr", server.Addr).Msg("Starting custom HTTP server")
		logger.Info().Msg("Routes available:")
		logger.Info().Msg("  GET  http://localhost:8081/")
		logger.Info().Msg("  GET  http://localhost:8081/health")
		logger.Info().Msg("  GET  http://localhost:8081/metrics")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Get the meter provider
	mp := t.MeterProvider()
	if mp == nil {
		logger.Error().Msg("Metrics provider is nil - this should not happen")
		return
	}

	// Create a meter for this component
	meter := mp.Meter("example-meter")

	// Counter: Monotonically increasing value (e.g., total requests)
	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		panic(err)
	}

	// Histogram: Distribution of values (e.g., request duration)
	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_milliseconds",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}

	// Gauge (via Observable): Current value at observation time (e.g., memory usage)
	_, err = meter.Int64ObservableGauge(
		"system_memory_usage_bytes",
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

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	logger.Info().Msg("Simulating application workload... (Press Ctrl+C to stop)")

	endpoints := []string{"/api/users", "/api/orders", "/api/products"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []int{200, 201, 400, 404, 500}

	// Run workload until interrupted
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			logger.Info().Msg("Received shutdown signal")
			// Shutdown the custom HTTP server
			if err := server.Shutdown(ctx); err != nil {
				logger.Error().Err(err).Msg("Failed to shutdown HTTP server")
			}
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

			logger.Debug().
				Str("method", method).
				Str("endpoint", endpoint).
				Int("status", statusCode).
				Float64("duration_ms", duration).
				Msg("Request processed")

			// Occasional reminder
			if rand.Intn(20) == 0 {
				fmt.Printf("\n💡 Reminder: Your custom HTTP server is running at http://localhost:8081\n")
				fmt.Printf("   Try: curl http://localhost:8081/metrics\n\n")
			}
		}
	}
}
