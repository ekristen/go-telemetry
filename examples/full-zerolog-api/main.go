package main

import (
	"context"
	"errors"
	"time"

	"github.com/ekristen/go-telemetry"
	zerologger "github.com/ekristen/go-telemetry/logger/zerolog"
)

func main() {
	ctx := context.Background()

	t, err := telemetry.New(ctx, &telemetry.Options{
		ServiceName:    "zerolog-showcase",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		panic(err)
	}
	defer t.Shutdown(ctx)

	logger := t.Logger()

	// The logger interface provides common logging methods
	logger.Info().Str("app", "showcase").Msg("Starting application")

	// To access full zerolog API, use type assertion
	if zlog, ok := logger.(*zerologger.Logger); ok {
		// Now you have full access to zerolog's Logger through the embedded field
		// Use all of zerolog's capabilities directly

		// Advanced field types
		zlog.Logger.Info().
			Str("string", "value").
			Int("int", 42).
			Int64("int64", 123456789).
			Uint64("uint64", 987654321).
			Float64("float", 3.14159).
			Bool("bool", true).
			Time("time", time.Now()).
			Dur("duration", 5*time.Second).
			Msg("All field types")

		// Nested objects
		zlog.Logger.Info().
			Interface("user", map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			}).
			Msg("User information")

		// Arrays
		zlog.Logger.Info().
			Strs("tags", []string{"go", "telemetry", "otel"}).
			Ints("counts", []int{1, 2, 3, 4, 5}).
			Msg("Array fields")

		// Error with stack trace
		err := errors.New("something went wrong")
		zlog.Logger.Error().
			Err(err).
			Str("component", "processor").
			Msg("Failed to process")

		// Conditional logging with context
		contextLogger := zlog.Logger.With().
			Str("request_id", "req-123").
			Str("user_id", "user-456").
			Logger()

		contextLogger.Info().Msg("Request processing")
		contextLogger.Debug().Msg("Debug details")

		// Multiple log entries showing different capabilities
		for i := 0; i < 5; i++ {
			zlog.Logger.Debug().
				Int("iteration", i).
				Str("status", "processing").
				Msg("Loop iteration")
		}
	}

	logger.Info().Msg("Application finished")
}
