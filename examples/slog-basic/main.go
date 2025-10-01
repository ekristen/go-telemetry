package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ekristen/go-telemetry"
	sloglogger "github.com/ekristen/go-telemetry/logger/slog"
)

func main() {
	ctx := context.Background()

	// Example 1: Using New() to create a new slog logger
	exampleWithNew(ctx)

	// Example 2: Using Wrap() to wrap an existing slog logger
	exampleWithWrap(ctx)
}

func exampleWithNew(ctx context.Context) {
	// Super simplified: Create slog logger with just the logger config
	// No need to specify ServiceName/ServiceVersion here - telemetry will set them!
	slogLog := sloglogger.New(sloglogger.Options{
		Output:     os.Stdout,
		Level:      slog.LevelDebug,
		AddSource:  true, // Add source file:line info
		JSONFormat: false,
	})

	// Create telemetry instance with the slog logger
	// The telemetry system will:
	// 1. Automatically set ServiceName/ServiceVersion in the logger
	// 2. Add OTel integration if OTEL_EXPORTER_OTLP_ENDPOINT is set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "slog-example",
		ServiceVersion: "1.0.0",
		Logger:         slogLog,
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	log.Info().Str("example", "new").Msg("Using New() to create logger")

	// Use the logger with standard interface methods
	log.Info().Str("status", "started").Msg("Application started with slog")
	log.Debug().Int("workers", 5).Msg("Configured workers")
	log.Warn().Msg("This is a warning")

	// Example with error
	log.Error().Str("component", "processor").Msg("Example error log")

	// Access full slog capabilities by type asserting
	if slogLog, ok := log.(*sloglogger.Logger); ok {
		// Use full slog API
		slogLog.Logger.Info("Processing request with full slog API",
			slog.String("user_id", "user-123"),
			slog.String("request_id", "req-456"),
			slog.String("method", "GET"),
		)

		// Use slog-specific features like groups
		slogLog.Logger.Info("Request completed",
			slog.Group("request",
				slog.Int("duration_ms", 150),
				slog.Bool("success", true),
			),
		)

		// Structured logging with multiple field types
		slogLog.Logger.Info("Complex event",
			slog.String("event", "user_action"),
			slog.Int64("timestamp", 1234567890),
			slog.Float64("score", 98.5),
			slog.Any("tags", []string{"important", "verified"}),
		)
	}

	log.Info().Msg("Example 1 finished")
}

func exampleWithWrap(ctx context.Context) {
	// Create your own slog logger with custom configuration
	mySlog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))

	// Wrap your existing slog logger - telemetry will configure it automatically
	wrappedLog := sloglogger.Wrap(mySlog, sloglogger.WrapOptions{})

	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "slog-example-wrap",
		ServiceVersion: "1.0.0",
		Logger:         wrappedLog,
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	log.Info().Str("example", "wrap").Msg("Using Wrap() to wrap existing logger")

	// Your existing slog logger is now integrated
	log.Info().Str("status", "wrapped").Msg("Using wrapped slog logger")

	// Access full slog capabilities
	if slogLog, ok := log.(*sloglogger.Logger); ok {
		slogLog.Logger.Info("This is the original logger",
			slog.Bool("wrapped", true),
		)
	}

	log.Info().Msg("Example 2 finished")
}
