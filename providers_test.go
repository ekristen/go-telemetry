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
