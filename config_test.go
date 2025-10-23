package telemetry

import (
	"os"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.ServiceName != "unknown" {
		t.Errorf("DefaultOptions().ServiceName = %v, want 'unknown'", opts.ServiceName)
	}

	if opts.ServiceVersion != "unknown" {
		t.Errorf("DefaultOptions().ServiceVersion = %v, want 'unknown'", opts.ServiceVersion)
	}

	if opts.BatchExport {
		t.Error("DefaultOptions().BatchExport = true, want false")
	}
}

func TestOptions_applyEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantService string
		wantVersion string
	}{
		{
			name:        "no env vars",
			envVars:     map[string]string{},
			wantService: "test-service",
			wantVersion: "1.0.0",
		},
		{
			name: "OTEL_SERVICE_NAME set",
			envVars: map[string]string{
				"OTEL_SERVICE_NAME": "env-service",
			},
			wantService: "env-service",
			wantVersion: "1.0.0",
		},
		{
			name: "OTEL_SERVICE_VERSION set",
			envVars: map[string]string{
				"OTEL_SERVICE_VERSION": "2.0.0",
			},
			wantService: "test-service",
			wantVersion: "2.0.0",
		},
		{
			name: "both env vars set",
			envVars: map[string]string{
				"OTEL_SERVICE_NAME":    "env-service",
				"OTEL_SERVICE_VERSION": "2.0.0",
			},
			wantService: "env-service",
			wantVersion: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			opts := &Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			}
			opts.applyEnvVars()

			if opts.ServiceName != tt.wantService {
				t.Errorf("ServiceName = %v, want %v", opts.ServiceName, tt.wantService)
			}

			if opts.ServiceVersion != tt.wantVersion {
				t.Errorf("ServiceVersion = %v, want %v", opts.ServiceVersion, tt.wantVersion)
			}
		})
	}
}

func TestShouldEnableOTel(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no env vars - disabled by default",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name: "OTEL_SDK_DISABLED=true",
			envVars: map[string]string{
				"OTEL_SDK_DISABLED": "true",
			},
			want: false,
		},
		{
			name: "OTEL_EXPORTER_OTLP_ENDPOINT set",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT set",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT set",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT set",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTEL_TRACES_EXPORTER=otlp",
			envVars: map[string]string{
				"OTEL_TRACES_EXPORTER": "otlp",
			},
			want: true,
		},
		{
			name: "OTEL_METRICS_EXPORTER=otlp",
			envVars: map[string]string{
				"OTEL_METRICS_EXPORTER": "otlp",
			},
			want: true,
		},
		{
			name: "OTEL_LOGS_EXPORTER=otlp",
			envVars: map[string]string{
				"OTEL_LOGS_EXPORTER": "otlp",
			},
			want: true,
		},
		{
			name: "OTEL_TRACES_EXPORTER=none",
			envVars: map[string]string{
				"OTEL_TRACES_EXPORTER": "none",
			},
			want: false,
		},
		{
			name: "OTEL_SDK_DISABLED overrides endpoint",
			envVars: map[string]string{
				"OTEL_SDK_DISABLED":           "true",
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all OTel env vars first
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := shouldEnableOTel()
			if got != tt.want {
				t.Errorf("shouldEnableOTel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldEnableTraces(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no env vars - disabled",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name: "OTel enabled, traces not disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTel enabled, traces explicitly disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_TRACES_EXPORTER":        "none",
			},
			want: false,
		},
		{
			name: "OTel disabled",
			envVars: map[string]string{
				"OTEL_SDK_DISABLED": "true",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := shouldEnableTraces()
			if got != tt.want {
				t.Errorf("shouldEnableTraces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldEnableMetrics(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no env vars - disabled",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name: "OTel enabled, metrics not disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTel enabled, metrics explicitly disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_METRICS_EXPORTER":       "none",
			},
			want: false,
		},
		{
			name: "OTel disabled",
			envVars: map[string]string{
				"OTEL_SDK_DISABLED": "true",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := shouldEnableMetrics()
			if got != tt.want {
				t.Errorf("shouldEnableMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldEnableLogs(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no env vars - disabled",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name: "OTel enabled, logs not disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			want: true,
		},
		{
			name: "OTel enabled, logs explicitly disabled",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_LOGS_EXPORTER":          "none",
			},
			want: false,
		},
		{
			name: "OTel disabled",
			envVars: map[string]string{
				"OTEL_SDK_DISABLED": "true",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := shouldEnableLogs()
			if got != tt.want {
				t.Errorf("shouldEnableLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptions_applyEnvVars_PrometheusSettings(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantPort int
		wantPath string
	}{
		{
			name:     "no prometheus env vars",
			envVars:  map[string]string{},
			wantPort: 9090, // default
			wantPath: "/metrics",
		},
		{
			name: "PROMETHEUS_PORT set",
			envVars: map[string]string{
				"PROMETHEUS_PORT": "8080",
			},
			wantPort: 8080,
			wantPath: "/metrics",
		},
		{
			name: "PROMETHEUS_PATH set",
			envVars: map[string]string{
				"PROMETHEUS_PATH": "/custom/metrics",
			},
			wantPort: 9090,
			wantPath: "/custom/metrics",
		},
		{
			name: "both prometheus settings",
			envVars: map[string]string{
				"PROMETHEUS_PORT": "7070",
				"PROMETHEUS_PATH": "/prom",
			},
			wantPort: 7070,
			wantPath: "/prom",
		},
		{
			name: "invalid port number - should keep default",
			envVars: map[string]string{
				"PROMETHEUS_PORT": "invalid",
			},
			wantPort: 9090, // should keep default on error
			wantPath: "/metrics",
		},
		{
			name: "empty port string - should keep default",
			envVars: map[string]string{
				"PROMETHEUS_PORT": "",
			},
			wantPort: 9090,
			wantPath: "/metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			os.Unsetenv("PROMETHEUS_PORT")
			os.Unsetenv("PROMETHEUS_PATH")
			defer func() {
				os.Unsetenv("PROMETHEUS_PORT")
				os.Unsetenv("PROMETHEUS_PATH")
			}()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			opts := DefaultOptions()
			opts.applyEnvVars()

			if opts.PrometheusPort != tt.wantPort {
				t.Errorf("PrometheusPort = %v, want %v", opts.PrometheusPort, tt.wantPort)
			}

			if opts.PrometheusPath != tt.wantPath {
				t.Errorf("PrometheusPath = %v, want %v", opts.PrometheusPath, tt.wantPath)
			}
		})
	}
}

func TestOptions_applyEnvVars_MetricsExporter(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		wantExporter string
	}{
		{
			name:         "no metrics exporter set",
			envVars:      map[string]string{},
			wantExporter: "",
		},
		{
			name: "OTEL_METRICS_EXPORTER=otlp",
			envVars: map[string]string{
				"OTEL_METRICS_EXPORTER": "otlp",
			},
			wantExporter: "otlp",
		},
		{
			name: "OTEL_METRICS_EXPORTER=prometheus",
			envVars: map[string]string{
				"OTEL_METRICS_EXPORTER": "prometheus",
			},
			wantExporter: "prometheus",
		},
		{
			name: "OTEL_METRICS_EXPORTER=none",
			envVars: map[string]string{
				"OTEL_METRICS_EXPORTER": "none",
			},
			wantExporter: "none",
		},
		{
			name: "OTEL_METRICS_EXPORTER with multiple exporters",
			envVars: map[string]string{
				"OTEL_METRICS_EXPORTER": "prometheus,otlp",
			},
			wantExporter: "prometheus,otlp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env var
			os.Unsetenv("OTEL_METRICS_EXPORTER")
			defer os.Unsetenv("OTEL_METRICS_EXPORTER")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			opts := DefaultOptions()
			opts.applyEnvVars()

			if opts.MetricsExporter != tt.wantExporter {
				t.Errorf("MetricsExporter = %v, want %v", opts.MetricsExporter, tt.wantExporter)
			}
		})
	}
}

func TestOptions_applyEnvVars_AllSettings(t *testing.T) {
	// Test that all environment variables work together
	envVars := map[string]string{
		"OTEL_SERVICE_NAME":     "env-service",
		"OTEL_SERVICE_VERSION":  "2.0.0",
		"OTEL_METRICS_EXPORTER": "prometheus",
		"PROMETHEUS_PORT":       "8888",
		"PROMETHEUS_PATH":       "/custom",
	}

	// Clear all env vars
	for k := range envVars {
		os.Unsetenv(k)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	// Set all env vars
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	opts := &Options{
		ServiceName:     "original-service",
		ServiceVersion:  "1.0.0",
		PrometheusPort:  9090,
		PrometheusPath:  "/metrics",
		MetricsExporter: "",
	}
	opts.applyEnvVars()

	if opts.ServiceName != "env-service" {
		t.Errorf("ServiceName = %v, want 'env-service'", opts.ServiceName)
	}
	if opts.ServiceVersion != "2.0.0" {
		t.Errorf("ServiceVersion = %v, want '2.0.0'", opts.ServiceVersion)
	}
	if opts.MetricsExporter != "prometheus" {
		t.Errorf("MetricsExporter = %v, want 'prometheus'", opts.MetricsExporter)
	}
	if opts.PrometheusPort != 8888 {
		t.Errorf("PrometheusPort = %v, want 8888", opts.PrometheusPort)
	}
	if opts.PrometheusPath != "/custom" {
		t.Errorf("PrometheusPath = %v, want '/custom'", opts.PrometheusPath)
	}
}

func TestDefaultOptions_Values(t *testing.T) {
	opts := DefaultOptions()

	// Verify all default values
	if opts.ServiceName != "unknown" {
		t.Errorf("DefaultOptions().ServiceName = %v, want 'unknown'", opts.ServiceName)
	}

	if opts.ServiceVersion != "unknown" {
		t.Errorf("DefaultOptions().ServiceVersion = %v, want 'unknown'", opts.ServiceVersion)
	}

	if opts.BatchExport {
		t.Error("DefaultOptions().BatchExport should be false")
	}

	if opts.PrometheusPort != 9090 {
		t.Errorf("DefaultOptions().PrometheusPort = %v, want 9090", opts.PrometheusPort)
	}

	if opts.PrometheusPath != "/metrics" {
		t.Errorf("DefaultOptions().PrometheusPath = %v, want '/metrics'", opts.PrometheusPath)
	}

	if opts.MetricsExporter != "" {
		t.Errorf("DefaultOptions().MetricsExporter = %v, want ''", opts.MetricsExporter)
	}
}

// Helper function to clear all OTel environment variables
func clearOTelEnvVars() {
	envVars := []string{
		"OTEL_SDK_DISABLED",
		"OTEL_SERVICE_NAME",
		"OTEL_SERVICE_VERSION",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_TRACES_EXPORTER",
		"OTEL_METRICS_EXPORTER",
		"OTEL_LOGS_EXPORTER",
		"PROMETHEUS_PORT",
		"PROMETHEUS_PATH",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
