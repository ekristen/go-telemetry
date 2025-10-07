package main

import (
	"os"

	zerologger "github.com/ekristen/go-telemetry/logger/zerolog"
)

func main() {
	// Example 1: Automatic caller detection (recommended)
	// The logger automatically detects the correct caller location
	log := zerologger.New(zerologger.Options{
		Output:       os.Stdout,
		EnableCaller: true,
		EnableColor:  true,
	})

	log.Info().Msg("Direct call with automatic caller detection")

	// Example 2: Works correctly even through wrapper functions
	// The caller location will show databaseOperation(), not the logger internals
	databaseOperation(log)

	// Example 3: Works through multiple wrapper layers
	serviceLayer(log)

	// Example 4: Manual override (rarely needed)
	// Only use this for edge cases or performance-critical code
	manualLog := zerologger.New(zerologger.Options{
		Output:               os.Stdout,
		EnableCaller:         true,
		EnableColor:          true,
		CallerSkipFrameCount: 5, // Manual override
	})

	manualLog.Info().Msg("Call with manual skip count override")
}

// databaseOperation simulates a database wrapper function
func databaseOperation(log *zerologger.Logger) {
	// This will correctly show databaseOperation() as the caller,
	// not the logger wrapper internals
	log.Info().Msg("Database operation called")
}

// serviceLayer simulates a service layer that wraps the database layer
func serviceLayer(log *zerologger.Logger) {
	// This will correctly show serviceLayer() as the caller
	log.Warn().Msg("Service layer processing")

	// Call through to another layer
	helperFunction(log)
}

// helperFunction simulates yet another wrapper layer
func helperFunction(log *zerologger.Logger) {
	// Even through multiple layers, this correctly shows helperFunction()
	log.Error().Msg("Helper function encountered an error")
}
