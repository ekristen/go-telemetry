package main

import (
	"context"
	"os"

	zaphook "github.com/ekristen/go-telemetry/hooks/zap/v2"
	"github.com/ekristen/go-telemetry/v2"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// This example demonstrates the OTel-first, external core pattern for Zap.
// By creating your zap logger externally and combining cores with the OTel core,
// you maintain full control and accurate caller reporting.
//
// This is the RECOMMENDED pattern for production use with Zap.

func main() {
	ctx := context.Background()

	// Step 1: Create YOUR zap core with full control
	// This example uses a console encoder for readable output
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	// You could also create a JSON core, file core, or any custom core
	// jsonEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	// jsonCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(file), zapcore.InfoLevel)

	// Step 2: Initialize OpenTelemetry providers
	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "zap-external-core-example",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	// Step 3: Create OTel core and combine with your core
	// This is like adding a hook - logs go to BOTH your core AND OTel
	var core zapcore.Core
	if t.LoggerProvider() != nil {
		otelCore := zaphook.New(
			t.ServiceName(),
			t.ServiceVersion(),
			t.LoggerProvider(),
		)
		if otelCore != nil {
			// Tee combines cores - logs go to both!
			core = zapcore.NewTee(consoleCore, otelCore)
		} else {
			core = consoleCore
		}
	} else {
		core = consoleCore
	}

	// Step 4: Create zap logger with combined cores
	// AddCaller enables caller reporting
	// AddCallerSkip(0) means no additional skip (direct zap API usage)
	log := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	defer log.Sync()

	// Step 5: Use zap's NATIVE API directly
	log.Info("Application started",
		zap.String("status", "running"),
		zap.String("environment", "production"),
	)

	log.Debug("Debug message with accurate caller info",
		zap.Int("count", 42),
		zap.Bool("enabled", true),
	)

	// Create a span for distributed tracing
	ctx, span := t.StartSpan(ctx, "process-request")
	defer span.End()

	// Log within span context
	// Note: Zap doesn't have native context support, but OTel core can extract it
	log.Info("Processing request within span",
		zap.String("request_id", "req-12345"),
		zap.String("user_id", "user-67890"),
		zap.Any("context", ctx), // OTel core extracts trace info from this
	)

	log.Info("Request completed successfully",
		zap.Int("duration_ms", 150),
		zap.Int("status_code", 200),
	)

	// Warning message
	log.Warn("Cache miss - falling back to database",
		zap.String("component", "cache"),
		zap.String("key", "user:123"),
	)

	// Error message
	log.Error("Database connection failed",
		zap.String("error", "connection timeout"),
		zap.String("component", "database"),
		zap.Int("retry", 3),
	)

	// Using zap's sugared logger for printf-style
	sugar := log.Sugar()
	sugar.Infow("Using sugared logger",
		"key", "value",
		"count", 99,
	)

	sugar.Infof("Formatted message: count=%d, status=%s", 100, "ok")

	log.Info("Application finished - check your OTel collector for exported logs!")
}
