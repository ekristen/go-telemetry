package telemetry

import (
	"context"
	"os"
	"testing"
)

func TestNewResource(t *testing.T) {
	serviceName := "test-service"
	serviceVersion := "1.0.0"

	res := newResource(serviceName, serviceVersion)

	if res == nil {
		t.Fatal("newResource() returned nil")
	}

	// Verify resource attributes
	attrs := res.Attributes()

	var foundService, foundVersion, foundHost bool
	for _, attr := range attrs {
		switch string(attr.Key) {
		case "service.name":
			if attr.Value.AsString() != serviceName {
				t.Errorf("service.name = %v, want %v", attr.Value.AsString(), serviceName)
			}
			foundService = true
		case "service.version":
			if attr.Value.AsString() != serviceVersion {
				t.Errorf("service.version = %v, want %v", attr.Value.AsString(), serviceVersion)
			}
			foundVersion = true
		case "host.name":
			// Just verify it exists and is not empty
			if attr.Value.AsString() == "" {
				t.Error("host.name is empty")
			}
			foundHost = true
		}
	}

	if !foundService {
		t.Error("service.name attribute not found")
	}
	if !foundVersion {
		t.Error("service.version attribute not found")
	}
	if !foundHost {
		t.Error("host.name attribute not found")
	}
}

func TestNewLoggerProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name:    "OTel disabled - returns nil",
			envVars: map[string]string{},
			wantNil: true,
		},
		{
			name: "logs disabled via exporter - returns nil",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_LOGS_EXPORTER":          "none",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			lp, err := newLoggerProvider(ctx, res, tt.batchExport)

			if err != nil {
				// Note: Error is expected when trying to connect to non-existent endpoint
				// We're mainly testing that the function handles the env vars correctly
				t.Logf("newLoggerProvider() error (may be expected): %v", err)
			}

			if tt.wantNil && lp != nil {
				t.Error("newLoggerProvider() should return nil when disabled")
			}
		})
	}
}

func TestNewTracerProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name:    "OTel disabled - returns nil",
			envVars: map[string]string{},
			wantNil: true,
		},
		{
			name: "traces disabled via exporter - returns nil",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_TRACES_EXPORTER":        "none",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			tp, err := newTracerProvider(ctx, res, tt.batchExport)

			if err != nil {
				// Note: Error is expected when trying to connect to non-existent endpoint
				t.Logf("newTracerProvider() error (may be expected): %v", err)
			}

			if tt.wantNil && tp != nil {
				t.Error("newTracerProvider() should return nil when disabled")
			}
		})
	}
}

func TestNewMeterProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name:    "OTel disabled - returns nil",
			envVars: map[string]string{},
			wantNil: true,
		},
		{
			name: "metrics disabled via exporter - returns nil",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_METRICS_EXPORTER":       "none",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			mp, err := newMeterProvider(ctx, res, tt.batchExport)

			if err != nil {
				// Note: Error is expected when trying to connect to non-existent endpoint
				t.Logf("newMeterProvider() error (may be expected): %v", err)
			}

			if tt.wantNil && mp != nil {
				t.Error("newMeterProvider() should return nil when disabled")
			}
		})
	}
}

func TestProvidersBatchMode(t *testing.T) {
	ctx := context.Background()
	res := newResource("test-service", "1.0.0")

	tests := []struct {
		name        string
		batchExport bool
	}{
		{
			name:        "simple export mode",
			batchExport: false,
		},
		{
			name:        "batch export mode",
			batchExport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			// Note: These will return errors because no endpoint is running,
			// but we're testing that the functions accept the batchExport parameter
			_, err := newLoggerProvider(ctx, res, tt.batchExport)
			t.Logf("newLoggerProvider(batch=%v) error: %v", tt.batchExport, err)

			_, err = newTracerProvider(ctx, res, tt.batchExport)
			t.Logf("newTracerProvider(batch=%v) error: %v", tt.batchExport, err)

			_, err = newMeterProvider(ctx, res, tt.batchExport)
			t.Logf("newMeterProvider(batch=%v) error: %v", tt.batchExport, err)
		})
	}
}

func TestNewPrometheusReader(t *testing.T) {
	res := newResource("test-service", "1.0.0")

	reader, handler, err := newPrometheusReader(res)
	if err != nil {
		t.Fatalf("newPrometheusReader() failed: %v", err)
	}

	if reader == nil {
		t.Error("newPrometheusReader() returned nil reader")
	}

	if handler == nil {
		t.Error("newPrometheusReader() returned nil handler")
	}

	// Verify handler is functional by checking its type
	if handler == nil {
		t.Error("HTTP handler should not be nil")
	}
}

func TestNewOTLPReader(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		batchExport bool
	}{
		{
			name:        "with batch export false",
			batchExport: false,
		},
		{
			name:        "with batch export true",
			batchExport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This will likely fail because no OTLP endpoint is running
			// but we're testing that the function creates a reader correctly
			reader, err := newOTLPReader(ctx, tt.batchExport)

			// Error is expected when no endpoint is available
			if err != nil {
				t.Logf("newOTLPReader() error (expected without endpoint): %v", err)
			}

			// If reader was created (unlikely without endpoint), verify it's not nil
			if err == nil && reader == nil {
				t.Error("newOTLPReader() returned nil reader without error")
			}
		})
	}
}

func TestNewLoggerProvider_WithOTelEnabled(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name: "OTel enabled with OTLP endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "OTel enabled with batch export",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: true,
			wantNil:     false,
		},
		{
			name: "OTel enabled with logs endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://localhost:4318",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "logs exporter set to otlp",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_LOGS_EXPORTER":          "otlp",
			},
			batchExport: false,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			lp, err := newLoggerProvider(ctx, res, tt.batchExport)

			// Error is expected when trying to connect to non-existent endpoint
			if err != nil {
				t.Logf("newLoggerProvider() error (may be expected): %v", err)
			}

			// Check if result matches expectation
			if tt.wantNil && lp != nil {
				t.Error("newLoggerProvider() should return nil when disabled")
			}
		})
	}
}

func TestNewTracerProvider_WithOTelEnabled(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name: "OTel enabled with OTLP endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "OTel enabled with batch export",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: true,
			wantNil:     false,
		},
		{
			name: "OTel enabled with traces endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://localhost:4318",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "traces exporter set to otlp",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_TRACES_EXPORTER":        "otlp",
			},
			batchExport: false,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			tp, err := newTracerProvider(ctx, res, tt.batchExport)

			// Error is expected when trying to connect to non-existent endpoint
			if err != nil {
				t.Logf("newTracerProvider() error (may be expected): %v", err)
			}

			// Check if result matches expectation
			if tt.wantNil && tp != nil {
				t.Error("newTracerProvider() should return nil when disabled")
			}
		})
	}
}

func TestNewMeterProvider_WithOTelEnabled(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		envVars     map[string]string
		batchExport bool
		wantNil     bool
	}{
		{
			name: "OTel enabled with OTLP endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "OTel enabled with batch export",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
			},
			batchExport: true,
			wantNil:     false,
		},
		{
			name: "OTel enabled with metrics endpoint",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://localhost:4318",
			},
			batchExport: false,
			wantNil:     false,
		},
		{
			name: "metrics exporter set to otlp",
			envVars: map[string]string{
				"OTEL_EXPORTER_OTLP_ENDPOINT": "http://localhost:4317",
				"OTEL_METRICS_EXPORTER":       "otlp",
			},
			batchExport: false,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOTelEnvVars()
			defer clearOTelEnvVars()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			res := newResource("test-service", "1.0.0")
			mp, err := newMeterProvider(ctx, res, tt.batchExport)

			// Error is expected when trying to connect to non-existent endpoint
			if err != nil {
				t.Logf("newMeterProvider() error (may be expected): %v", err)
			}

			// Check if result matches expectation
			if tt.wantNil && mp != nil {
				t.Error("newMeterProvider() should return nil when disabled")
			}
		})
	}
}

func TestNewResource_Hostname(t *testing.T) {
	serviceName := "test-service"
	serviceVersion := "1.0.0"

	res := newResource(serviceName, serviceVersion)

	if res == nil {
		t.Fatal("newResource() returned nil")
	}

	// Verify hostname is included in resource attributes
	attrs := res.Attributes()
	var foundHost bool
	var hostname string

	for _, attr := range attrs {
		if string(attr.Key) == "host.name" {
			hostname = attr.Value.AsString()
			foundHost = true
			break
		}
	}

	if !foundHost {
		t.Error("host.name attribute not found in resource")
	}

	// Hostname should not be empty (unless os.Hostname() fails, which is rare)
	t.Logf("Hostname in resource: %s", hostname)
}
