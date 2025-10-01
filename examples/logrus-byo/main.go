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

	// Create your own logrus logger with custom configuration
	myLogrus := logrus.New()
	myLogrus.SetOutput(os.Stdout)
	myLogrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	})
	myLogrus.SetLevel(logrus.DebugLevel)
	myLogrus.SetReportCaller(true)

	// Add custom hooks, fields, etc.
	myLogrus.AddHook(&customHook{})

	// Wrap your existing logrus logger to integrate with telemetry
	// OTel will be auto-enabled if OTEL_EXPORTER_OTLP_ENDPOINT or other OTel env vars are set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "logrus-byo-example",
		ServiceVersion: "1.0.0",
		Logger: logruslogger.Wrap(myLogrus, logruslogger.WrapOptions{
			ServiceName:    "logrus-byo-example",
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

	// Use the logger with standard interface methods
	log.Info().Str("status", "started").Msg("Application started with BYO logrus")
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

		// Your custom hook will also be called
		logrusLog.Logger.WithField("custom", "data").Info("Custom hook will fire")
	}

	log.Info().Msg("Application finished")
}

// customHook is an example custom logrus hook
type customHook struct{}

func (h *customHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *customHook) Fire(entry *logrus.Entry) error {
	// Your custom hook logic here
	// For example: send to external service, modify entry, etc.
	return nil
}
