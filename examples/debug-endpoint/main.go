package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ekristen/go-telemetry"
)

func main() {
	ctx := context.Background()

	// Print environment variables for debugging
	fmt.Println("=== Environment Variables ===")
	fmt.Printf("OTEL_EXPORTER_OTLP_ENDPOINT: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	fmt.Printf("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT"))
	fmt.Printf("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"))
	fmt.Printf("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT"))
	fmt.Printf("OTEL_LOGS_EXPORTER: %s\n", os.Getenv("OTEL_LOGS_EXPORTER"))
	fmt.Printf("OTEL_SDK_DISABLED: %s\n", os.Getenv("OTEL_SDK_DISABLED"))
	fmt.Println()

	// Create telemetry instance
	fmt.Println("=== Creating Telemetry ===")
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "debug-endpoint",
		ServiceVersion: "1.0.0",
		BatchExport:    false, // Use simple export for immediate feedback
	})
	if err != nil {
		fmt.Printf("ERROR creating telemetry: %v\n", err)
		panic(err)
	}
	defer func() {
		fmt.Println("\n=== Shutting Down ===")
		if err := t.Shutdown(ctx); err != nil {
			fmt.Printf("ERROR shutting down: %v\n", err)
		}
		fmt.Println("Shutdown complete")
	}()

	// Check what providers were created
	fmt.Printf("Logger Provider: %v\n", t.LoggerProvider())
	fmt.Printf("Tracer Provider: %v\n", t.TracerProvider())
	fmt.Printf("Meter Provider: %v\n", t.MeterProvider())
	fmt.Println()

	// Send some logs
	fmt.Println("=== Sending Logs ===")
	logger := t.Logger()

	logger.Info().Str("test", "value1").Msg("Test log 1")
	logger.Info().Str("test", "value2").Msg("Test log 2")
	logger.Warn().Str("test", "value3").Msg("Test log 3")
	logger.Error().Str("test", "value4").Msg("Test log 4")

	fmt.Println("Logs sent - check your OTel collector")
	fmt.Println("\nNOTE: If using OTEL_EXPORTER_OTLP_ENDPOINT, the SDK appends /v1/logs to the path")
	fmt.Println("      e.g., http://localhost:4317 becomes http://localhost:4317/v1/logs")
}
