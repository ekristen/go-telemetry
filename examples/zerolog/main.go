package main

import (
	"context"
	"os"

	zerologhook "github.com/ekristen/go-telemetry/hooks/zerolog/v2"
	"github.com/ekristen/go-telemetry/v2"

	"github.com/rs/zerolog"
)

// This example demonstrates the OTel-first, external hook pattern for Zerolog.
// By creating your zerolog logger externally and attaching the OTel hook,
// you maintain full control, accurate caller reporting, and zero allocation performance.
//
// This is the RECOMMENDED pattern for production use with Zerolog.

func main() {
	ctx := context.Background()

	// Step 1: Create YOUR zerolog logger with full control
	// Zerolog uses a builder pattern for configuration
	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller(). // Enable caller reporting - will be accurate!
		Logger()

	// You could also configure:
	// - Console writer for colored output
	// - Different output destinations
	// - Custom timestamp format
	// - Log level
	// - Any zerolog feature

	// Example: Pretty console output (optional)
	// log = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// Step 2: Initialize OpenTelemetry providers
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "zerolog-external-hook-example",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Step 3: Attach OTel hook to YOUR logger (non-invasive!)
	// This is like adding a middleware - logs go to BOTH output AND OTel
	if t.LoggerProvider() != nil {
		otelHook := zerologhook.New(
			t.ServiceName(),
			t.ServiceVersion(),
			t.LoggerProvider(),
		)
		if otelHook != nil {
			log = log.Hook(otelHook)
		}
	}

	// Step 4: Use zerolog's NATIVE API directly
	log.Info().
		Str("status", "running").
		Str("environment", "production").
		Msg("Application started")

	log.Debug().
		Int("count", 42).
		Bool("enabled", true).
		Msg("Debug message with accurate caller info")

	// Create a span for distributed tracing
	ctx, span := t.StartSpan(ctx, "process-request")
	defer span.End()

	// Log within span context
	// Zerolog's Ctx() extracts trace info from context
	log.Info().
		Ctx(ctx).
		Str("request_id", "req-12345").
		Str("user_id", "user-67890").
		Msg("Processing request within span")

	log.Info().
		Int("duration_ms", 150).
		Int("status_code", 200).
		Msg("Request completed successfully")

	// Warning message
	log.Warn().
		Str("component", "cache").
		Str("key", "user:123").
		Msg("Cache miss - falling back to database")

	// Error message
	log.Error().
		Str("error", "connection timeout").
		Str("component", "database").
		Int("retry", 3).
		Msg("Database connection failed")

	// Using zerolog's Dict for nested fields
	log.Info().
		Dict("request", zerolog.Dict().
			Str("method", "GET").
			Str("path", "/api/users").
			Int("status", 200),
		).
		Dict("response", zerolog.Dict().
			Int("bytes", 1024).
			Str("encoding", "gzip"),
		).
		Msg("HTTP request completed")

	// Array logging
	log.Info().
		Strs("tags", []string{"important", "user-action", "audit"}).
		Msg("Tagged event")

	// Timestamp override
	log.Info().
		Time("custom_time", zerolog.TimestampFunc()).
		Msg("Custom timestamp")

	// Using sublogger with context
	sublog := log.With().
		Str("module", "auth").
		Str("version", "2.0").
		Logger()

	sublog.Info().
		Str("user", "john").
		Msg("Authentication successful")

	log.Info().Msg("Application finished - check your OTel collector for exported logs!")
}
