package telemetry

import (
	"os"
	"strconv"
)

// Options holds configuration for the telemetry system.
type Options struct {
	// ServiceName is the name of the service.
	ServiceName string
	// ServiceVersion is the version of the service.
	ServiceVersion string

	// BatchExport controls whether telemetry data is exported in batches or immediately.
	// When true, uses batch processors/exporters for better performance (higher latency).
	// When false (default), uses simple/synchronous processors for immediate export (lower latency).
	// Batch mode is recommended for high-volume production workloads.
	// Simple mode is recommended for development and debugging.
	BatchExport bool

	// MetricsExporter specifies which metrics exporter to use: "otlp", "prometheus", or "none".
	// When empty, defaults to "otlp" if OTel is enabled via environment variables.
	// Can be overridden by OTEL_METRICS_EXPORTER environment variable.
	MetricsExporter string

	// PrometheusPort is the HTTP port for the Prometheus metrics endpoint (default: 9090).
	// Only used when MetricsExporter is "prometheus".
	// Can be overridden by PROMETHEUS_PORT environment variable.
	PrometheusPort int

	// PrometheusPath is the HTTP path for the Prometheus metrics endpoint (default: "/metrics").
	// Only used when MetricsExporter is "prometheus".
	// Can be overridden by PROMETHEUS_PATH environment variable.
	PrometheusPath string

	// PrometheusServer enables the built-in Prometheus HTTP server.
	// When false (default), use PrometheusHandler() to get the handler and register it
	// with your own HTTP server. Only used when MetricsExporter is "prometheus".
	PrometheusServer bool
}

// DefaultOptions returns Options with default values.
func DefaultOptions() *Options {
	return &Options{
		ServiceName:    "unknown",
		ServiceVersion: "unknown",
		BatchExport:    false, // Default to simple/immediate export
		PrometheusPort: 9090,
		PrometheusPath: "/metrics",
	}
}

// applyEnvVars applies environment variable overrides to the options.
// Standard OpenTelemetry environment variables:
// - OTEL_SERVICE_NAME: service name
// - OTEL_SERVICE_VERSION: service version (if supported)
// - OTEL_METRICS_EXPORTER: metrics exporter type (otlp, prometheus, none)
// - PROMETHEUS_PORT: Prometheus HTTP port (default: 9090)
// - PROMETHEUS_PATH: Prometheus HTTP path (default: /metrics)
func (o *Options) applyEnvVars() {
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		o.ServiceName = v
	}
	// Note: OTEL_SERVICE_VERSION is not a standard OTel env var,
	// but we support it for convenience alongside OTEL_RESOURCE_ATTRIBUTES
	if v := os.Getenv("OTEL_SERVICE_VERSION"); v != "" {
		o.ServiceVersion = v
	}
	if v := os.Getenv("OTEL_METRICS_EXPORTER"); v != "" {
		o.MetricsExporter = v
	}
	if v := os.Getenv("PROMETHEUS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			o.PrometheusPort = port
		}
	}
	if v := os.Getenv("PROMETHEUS_PATH"); v != "" {
		o.PrometheusPath = v
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
