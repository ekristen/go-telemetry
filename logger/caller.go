package logger

import (
	"runtime"
	"strings"
)

// FindFirstExternalCaller walks the call stack and finds the first frame
// that is outside of the telemetry library and underlying logger packages.
// This allows automatic caller detection without requiring manual skip count configuration.
//
// The function returns the number of frames to skip to reach the actual caller.
// If no external frame is found within 15 frames, it returns the default skip count of 3.
//
// Performance: ~200-400ns per call. This is called once at logger creation time
// and the result is cached, so the overhead is negligible.
func FindFirstExternalCaller() int {
	// Capture up to 15 program counters from the call stack
	// This should be sufficient for most use cases
	pc := make([]uintptr, 15)
	n := runtime.Callers(1, pc)
	if n == 0 {
		return 3 // Fallback to default if we can't get callers
	}

	// Convert program counters to frames
	frames := runtime.CallersFrames(pc[:n])

	skipCount := 0
	for {
		frame, more := frames.Next()
		skipCount++

		// Check if this frame is from user code (not internal)
		if !isInternalFrame(frame.File) {
			// Found the first external frame
			// We need to account for the additional frames between
			// runtime.Callers and the actual logging call
			return skipCount + 2
		}

		if !more {
			break
		}
	}

	// If we couldn't find an external frame, return default
	return 3
}

// isInternalFrame checks if a frame is from the telemetry library or
// underlying logger packages that should be skipped.
func isInternalFrame(file string) bool {
	// Skip frames from the telemetry library itself
	if strings.Contains(file, "go-telemetry/logger") {
		return true
	}

	// Skip frames from underlying logger libraries
	internalPackages := []string{
		"rs/zerolog",      // github.com/rs/zerolog
		"uber.org/zap",    // go.uber.org/zap
		"sirupsen/logrus", // github.com/sirupsen/logrus
		"log/slog",        // standard library slog
		"runtime/",        // Go runtime internals
	}

	for _, pkg := range internalPackages {
		if strings.Contains(file, pkg) {
			return true
		}
	}

	return false
}
