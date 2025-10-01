package main

import (
	"context"
	"errors"
	"os"

	"github.com/ekristen/go-telemetry"
	zaplogger "github.com/ekristen/go-telemetry/logger/zap"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	// Super simplified: Create zap logger with just the logger config
	// No need to specify ServiceName/ServiceVersion here - telemetry will set them!
	zapLog := zaplogger.New(zaplogger.Options{
		Output:       os.Stdout,
		EnableCaller: true,
		Development:  true, // Pretty console output
		JSONFormat:   false,
	})

	// Create telemetry instance with the zap logger
	// The telemetry system will:
	// 1. Automatically set ServiceName/ServiceVersion in the logger
	// 2. Add OTel integration if OTEL_EXPORTER_OTLP_ENDPOINT is set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "zap-example",
		ServiceVersion: "1.0.0",
		Logger:         zapLog,
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	// Use the logger with standard interface methods
	log.Info().Str("status", "started").Msg("Application started with zap")
	log.Debug().Int("workers", 5).Msg("Configured workers")
	log.Warn().Msg("This is a warning")

	// Example with error
	log.Error().Str("component", "processor").Msg("Example error log")

	// Access full zap capabilities by type asserting
	if zapLog, ok := log.(*zaplogger.Logger); ok {
		// Access full zap capabilities directly
		zapLog.Logger.Info("Processing request with full zap API",
			zap.String("user_id", "user-123"),
			zap.String("request_id", "req-456"),
			zap.String("method", "GET"),
		)

		// Use zap-specific features
		zapLog.Logger.Info("Request completed",
			zap.Int("duration_ms", 150),
			zap.Bool("success", true),
		)

		// Structured logging with multiple types
		zapLog.Logger.Info("Complex event",
			zap.String("event", "user_action"),
			zap.Int64("timestamp", 1234567890),
			zap.Float64("score", 98.5),
			zap.Strings("tags", []string{"important", "verified"}),
		)

		// Error with stack trace
		err := errors.New("something went wrong")
		zapLog.Logger.Error("Failed to process",
			zap.Error(err),
			zap.String("component", "processor"),
			zap.Stack("stacktrace"),
		)

		// Use SugaredLogger for printf-style
		zapLog.Logger.Sugar().Infow("Sugared logging",
			"user", "john",
			"age", 30,
			"active", true,
		)
	}

	log.Info().Msg("Application finished")
}
