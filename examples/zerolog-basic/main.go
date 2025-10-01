package main

import (
	"context"
	"os"

	"github.com/ekristen/go-telemetry"
	zerologger "github.com/ekristen/go-telemetry/logger/zerolog"
)

func main() {
	ctx := context.Background()

	// Create telemetry instance with zerolog logger
	// OTel will be auto-enabled if OTEL_EXPORTER_OTLP_ENDPOINT or other OTel env vars are set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "zerolog-example",
		ServiceVersion: "1.0.0",
		Logger: zerologger.New(zerologger.Options{
			ServiceName:    "zerolog-example",
			ServiceVersion: "1.0.0",
			LoggerProvider: nil, // Will be set automatically if OTel is enabled
			Output:         os.Stdout,
			EnableCaller:   true,
			EnableColor:    true,
		}),
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	// Use the logger with standard interface methods
	log.Info().Str("status", "started").Msg("Application started with zerolog")
	log.Debug().Int("workers", 5).Msg("Configured workers")
	log.Warn().Msg("This is a warning")

	// Example with error
	log.Error().Str("component", "processor").Msg("Example error log")

	// Access full zerolog capabilities by type asserting
	if zerologLog, ok := log.(*zerologger.Logger); ok {
		// Now you have access to the full zerolog API
		zerologLog.Logger.Info().
			Str("user_id", "user-123").
			Str("request_id", "req-456").
			Str("method", "GET").
			Msg("Processing request with full zerolog API")

		// Use zerolog-specific features like sub-loggers
		subLogger := zerologLog.Logger.With().
			Str("component", "database").
			Logger()
		subLogger.Info().Int("connections", 10).Msg("Database pool initialized")

		// Structured logging with multiple field types
		zerologLog.Logger.Info().
			Int("duration_ms", 150).
			Bool("success", true).
			Float64("score", 98.5).
			Msg("Request completed")
	}

	log.Info().Msg("Application finished")
}
