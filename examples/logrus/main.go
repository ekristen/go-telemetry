package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	logrushook "github.com/ekristen/go-telemetry/hooks/logrus/v2"
	"github.com/ekristen/go-telemetry/v2"
)

// This example demonstrates the recommended OTel-first, non-invasive approach to
// integrating logrus with OpenTelemetry. By creating your logrus logger externally
// and attaching the OTel hook afterwards, you maintain:
//
// 1. Full control over logger configuration
// 2. Accurate caller reporting (SetReportCaller works correctly)
// 3. Ability to add custom hooks before/after OTel
// 4. Clear separation between logging and observability concerns
//
// This is the RECOMMENDED pattern for production use.

func main() {
	ctx := context.Background()

	// Step 1: Create and configure your logrus logger externally
	// You have complete control over formatter, output, level, and caller settings
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:     true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(logrus.DebugLevel)

	// IMPORTANT: SetReportCaller BEFORE adding hooks
	// This ensures caller information is captured from your code, not the hook
	log.SetReportCaller(true)

	// Optional: Add your custom hooks before OTel
	log.AddHook(&metricsHook{})

	// Step 2: Initialize OpenTelemetry providers
	// This creates the OTel logger provider, meter provider, and tracer provider
	// OTel will be auto-enabled if OTEL_EXPORTER_OTLP_ENDPOINT is set
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "logrus-example",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Step 3: Attach OpenTelemetry hook to your existing logger (non-invasive!)
	// This is where the magic happens - logs now go to both console AND OTel
	if t.LoggerProvider() != nil {
		otelHook := logrushook.New(
			t.ServiceName(),
			t.ServiceVersion(),
			t.LoggerProvider(),
		)
		if otelHook != nil {
			log.AddHook(otelHook)
			log.Info("OpenTelemetry hook attached successfully")
		}
	}

	// Step 4: Use your logger as normal!
	// Logs go to console (via formatter) AND OpenTelemetry (via hook)
	// Caller information is accurate because SetReportCaller was called before hooks
	log.WithFields(logrus.Fields{
		"status":      "started",
		"environment": "production",
	}).Info("Application started")

	log.Debug("Debug message with accurate caller info")

	// Create a span for distributed tracing
	ctx, span := t.StartSpan(ctx, "process-request")
	defer span.End()

	// Log within span context (trace info automatically added by OTel hook)
	log.WithContext(ctx).WithFields(logrus.Fields{
		"request_id": "req-12345",
		"user_id":    "user-67890",
	}).Info("Processing request within span")

	log.WithFields(logrus.Fields{
		"duration_ms": 150,
		"status_code": 200,
	}).Info("Request completed successfully")

	// Example warning
	log.WithField("component", "cache").Warn("Cache miss - falling back to database")

	// Example error
	log.WithFields(logrus.Fields{
		"error":     "connection timeout",
		"component": "database",
		"retry":     3,
	}).Error("Database connection failed")

	log.Info("Application finished - check your OTel collector for exported logs!")
}

// metricsHook is an example custom hook that could track log metrics
// This demonstrates that you can have multiple hooks working together
type metricsHook struct {
	errorCount int
	warnCount  int
}

func (h *metricsHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
}

func (h *metricsHook) Fire(entry *logrus.Entry) error {
	// Track errors and warnings
	switch entry.Level {
	case logrus.ErrorLevel:
		h.errorCount++
	case logrus.WarnLevel:
		h.warnCount++
	}

	// You could export these to StatsD, etc.
	return nil
}
