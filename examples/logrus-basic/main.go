package main

import (
	"context"
	"os"

	"github.com/ekristen/go-telemetry"
	logruslogger "github.com/ekristen/go-telemetry/logger/logrus"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	// Example 1: Using New() to create a new logrus logger
	exampleWithNew(ctx)

	// Example 2: Using Wrap() to wrap an existing logrus logger
	exampleWithWrap(ctx)
}

func exampleWithNew(ctx context.Context) {
	// Create telemetry instance with New() - creates a new logrus logger
	// OTel will be auto-enabled if OTEL_EXPORTER_OTLP_ENDPOINT or other OTel env vars are set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "logrus-example",
		ServiceVersion: "1.0.0",
		Logger: logruslogger.New(logruslogger.Options{
			ServiceName:    "logrus-example",
			ServiceVersion: "1.0.0",
			LoggerProvider: nil, // Will be set automatically if OTel is enabled
			Output:         os.Stdout,
			EnableColor:    true,
			JSONFormat:     false,
		}),
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	log.Info().Str("example", "new").Msg("Using New() to create logger")

	// Use the logger with standard interface methods
	log.Info().Str("status", "started").Msg("Application started with logrus")
	log.Debug().Int("workers", 5).Msg("Configured workers")
	log.Warn().Msg("This is a warning")

	// Example with error
	log.Error().Str("component", "processor").Msg("Example error log")

	// Access full logrus capabilities by type asserting
	if logrusLog, ok := log.(*logruslogger.Logger); ok {
		logrusLog.Logger.WithFields(map[string]interface{}{
			"user_id":    "user-123",
			"request_id": "req-456",
			"method":     "GET",
		}).Info("Processing request with full logrus API")

		// Use logrus-specific features
		logrusLog.Logger.WithField("duration_ms", 150).Info("Request completed")
	}

	log.Info().Msg("Example 1 finished")
}

func exampleWithWrap(ctx context.Context) {
	// Create your own logrus logger with custom configuration
	myLogrus := logrus.New()
	myLogrus.SetOutput(os.Stdout)
	myLogrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	myLogrus.SetLevel(logrus.InfoLevel)

	// Wrap your existing logrus logger to integrate with telemetry
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "logrus-example-wrap",
		ServiceVersion: "1.0.0",
		Logger: logruslogger.Wrap(myLogrus, logruslogger.WrapOptions{
			ServiceName:    "logrus-example-wrap",
			ServiceVersion: "1.0.0",
			LoggerProvider: nil, // Will be set automatically if OTel is enabled
		}),
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Get the logger from telemetry
	log := t.Logger()

	log.Info().Str("example", "wrap").Msg("Using Wrap() to wrap existing logger")

	// Your existing logrus logger is now integrated
	log.Info().Str("status", "wrapped").Msg("Using wrapped logrus logger")

	// Access full logrus capabilities
	if logrusLog, ok := log.(*logruslogger.Logger); ok {
		logrusLog.Logger.WithField("wrapped", true).Info("This is the original logger")
	}

	log.Info().Msg("Example 2 finished")
}
