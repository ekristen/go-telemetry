package main

import (
	"context"

	"github.com/ekristen/go-telemetry"
)

func main() {
	ctx := context.Background()

	// Create telemetry (OTel disabled by default - no-op providers)
	// To enable OTel, set: export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger
	logger := t.Logger()

	// Use the logger with full zerolog capabilities
	logger.Info().Str("status", "started").Msg("Application started")
	logger.Debug().Int("workers", 5).Msg("Configured workers")
	logger.Warn().Msg("This is a warning")

	// Logger also supports the standard interface methods
	logger.Info().Msg("Processing data")

	// Example with error
	logger.Error().Err(nil).Msg("Example error log")
}
