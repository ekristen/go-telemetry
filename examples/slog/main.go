package main

import (
	"context"
	"log/slog"
	"os"

	sloghook "github.com/ekristen/go-telemetry/hooks/slog/v2"
	"github.com/ekristen/go-telemetry/v2"
)

// This example demonstrates the OTel-first, external handler pattern for slog.
// By creating your slog handler externally and wrapping it with the OTel handler,
// you maintain accurate caller reporting when using slog's native API.
//
// KEY: You must use slog's native API (not a wrapper interface) for accurate caller info.

func main() {
	ctx := context.Background()

	// Step 1: Create YOUR slog handler with full control
	// AddSource: true enables caller information (file:line)
	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	// You could also use TextHandler or a custom handler
	// baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	//     AddSource: true,
	// })

	// Step 2: Initialize OpenTelemetry providers
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "slog-external-handler-example",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Step 3: Wrap YOUR handler with OTel handler (external pattern)
	// This is analogous to adding a logrus hook
	var handler slog.Handler = baseHandler
	if t.LoggerProvider() != nil {
		otelHandler := sloghook.New(
			baseHandler,
			t.ServiceName(),
			t.ServiceVersion(),
			t.LoggerProvider(),
		)
		if otelHandler != nil {
			handler = otelHandler
			slog.Info("OpenTelemetry handler attached successfully")
		}
	}

	// Create slog logger with wrapped handler
	log := slog.New(handler)

	// Step 4: Use slog's NATIVE API directly
	// This is the key to accurate caller reporting!

	log.Info("Application started",
		slog.String("status", "running"),
		slog.String("environment", "production"),
	)

	log.Debug("Debug message with accurate caller info",
		slog.Int("count", 42),
		slog.Bool("enabled", true),
	)

	// Create a span for distributed tracing
	ctx, span := t.StartSpan(ctx, "process-request")
	defer span.End()

	// Log within span context
	log.InfoContext(ctx, "Processing request within span",
		slog.String("request_id", "req-12345"),
		slog.String("user_id", "user-67890"),
	)

	// Use slog groups for structured data
	log.Info("Request completed",
		slog.Group("request",
			slog.Int("duration_ms", 150),
			slog.Int("status_code", 200),
		),
	)

	// Warning message
	log.Warn("Cache miss - falling back to database",
		slog.String("component", "cache"),
		slog.String("key", "user:123"),
	)

	// Error message
	log.Error("Database connection failed",
		slog.String("error", "connection timeout"),
		slog.String("component", "database"),
		slog.Int("retry", 3),
	)

	// Using different log levels
	log.Log(ctx, slog.LevelInfo, "Custom log level",
		slog.String("custom", "attribute"),
	)

	log.Info("Application finished - check your OTel collector for exported logs!")
}
