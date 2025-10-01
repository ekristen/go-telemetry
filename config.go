package telemetry

import (
	"os"
	"strconv"

	"github.com/ekristen/go-telemetry/logger"
)

// Logger is an alias for logger.Logger to allow users to specify custom loggers.
type Logger = logger.Logger

// Options holds configuration for the telemetry system.
type Options struct {
	// ServiceName is the name of the service.
	ServiceName string
	// ServiceVersion is the version of the service.
	ServiceVersion string

	// Logger is the logger implementation to use.
	// If nil, a default zerolog logger will be created.
	Logger Logger

	// LogConsoleOutput controls whether logs are written to console.
	// Only used if Logger is nil.
	LogConsoleOutput bool
	// LogConsoleColor controls whether console logs use colors.
	// Only used if Logger is nil.
	LogConsoleColor bool

	// BatchExport controls whether telemetry data is exported in batches or immediately.
	// When true, uses batch processors/exporters for better performance (higher latency).
	// When false (default), uses simple/synchronous processors for immediate export (lower latency).
	// Batch mode is recommended for high-volume production workloads.
	// Simple mode is recommended for development and debugging.
	BatchExport bool
}

// DefaultOptions returns Options with default values.
func DefaultOptions() *Options {
	return &Options{
		ServiceName:      "unknown",
		ServiceVersion:   "unknown",
		LogConsoleOutput: true,
		LogConsoleColor:  true,
		BatchExport:      false, // Default to simple/immediate export
	}
}

// applyEnvVars applies environment variable overrides to the options.
// Standard OpenTelemetry environment variables:
// - OTEL_SERVICE_NAME: service name
// - OTEL_SERVICE_VERSION: service version (if supported)
func (o *Options) applyEnvVars() {
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		o.ServiceName = v
	}
	// Note: OTEL_SERVICE_VERSION is not a standard OTel env var,
	// but we support it for convenience alongside OTEL_RESOURCE_ATTRIBUTES
	if v := os.Getenv("OTEL_SERVICE_VERSION"); v != "" {
		o.ServiceVersion = v
	}
}

// shouldEnableOTel determines if OpenTelemetry should be enabled based on
// standard OpenTelemetry environment variables.
// Returns false (no-op) by default, following OTel spec.
func shouldEnableOTel() bool {
	// Check OTEL_SDK_DISABLED first - if true, disable everything
	if disabled, _ := strconv.ParseBool(os.Getenv("OTEL_SDK_DISABLED")); disabled {
		return false
	}

	// Enable if OTLP endpoint is configured
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		return true
	}

	// Enable if any signal-specific OTLP endpoints are configured
	if os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		return true
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != "" {
		return true
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT") != "" {
		return true
	}

	// Enable if any exporter is explicitly configured (and not "none")
	if exp := os.Getenv("OTEL_TRACES_EXPORTER"); exp != "" && exp != "none" {
		return true
	}
	if exp := os.Getenv("OTEL_METRICS_EXPORTER"); exp != "" && exp != "none" {
		return true
	}
	if exp := os.Getenv("OTEL_LOGS_EXPORTER"); exp != "" && exp != "none" {
		return true
	}

	// No-op by default (follows OTel spec)
	return false
}

// shouldEnableTraces determines if trace collection should be enabled.
func shouldEnableTraces() bool {
	if !shouldEnableOTel() {
		return false
	}
	exp := os.Getenv("OTEL_TRACES_EXPORTER")
	// Enable if not explicitly set to "none", default is "otlp"
	return exp != "none"
}

// shouldEnableMetrics determines if metric collection should be enabled.
func shouldEnableMetrics() bool {
	if !shouldEnableOTel() {
		return false
	}
	exp := os.Getenv("OTEL_METRICS_EXPORTER")
	// Enable if not explicitly set to "none", default is "otlp"
	return exp != "none"
}

// shouldEnableLogs determines if log collection should be enabled.
func shouldEnableLogs() bool {
	if !shouldEnableOTel() {
		return false
	}
	exp := os.Getenv("OTEL_LOGS_EXPORTER")
	// Enable if not explicitly set to "none", default is "otlp"
	return exp != "none"
}
