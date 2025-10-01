package telemetry

import (
	"context"
	"io"
	"testing"

	"github.com/ekristen/go-telemetry/logger"
	"go.opentelemetry.io/otel/trace"
)

func TestNew(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "nil options uses defaults",
			opts: nil,
		},
		{
			name: "basic options",
			opts: &Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			},
		},
		{
			name: "with batch export",
			opts: &Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				BatchExport:    true,
			},
		},
		{
			name: "with console output disabled",
			opts: &Options{
				ServiceName:      "test-service",
				ServiceVersion:   "1.0.0",
				LogConsoleOutput: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := New(ctx, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			defer tel.Shutdown(ctx)

			// Verify telemetry instance is valid
			if tel == nil {
				t.Fatal("New() returned nil telemetry")
			}

			// Verify logger is initialized
			if tel.Logger() == nil {
				t.Error("Logger() returned nil")
			}

			// Verify tracer is initialized
			if tel.Tracer() == nil {
				t.Error("Tracer() returned nil")
			}
		})
	}
}

func TestTelemetry_Logger(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	logger := tel.Logger()
	if logger == nil {
		t.Error("Logger() returned nil")
	}
}

func TestTelemetry_Tracer(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	tracer := tel.Tracer()
	if tracer == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestTelemetry_StartSpan(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	newCtx, span := tel.StartSpan(ctx, "test-span")
	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}
	defer span.End()

	if newCtx == ctx {
		t.Error("StartSpan() should return a new context")
	}

	// Note: When OTel is disabled, noop tracer creates invalid span contexts
	// This is expected behavior - the span still works but doesn't record
	t.Logf("Span context valid: %v", span.SpanContext().IsValid())
}

func TestTelemetry_StartSpanWithLogger(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	newCtx, span, logger := tel.StartSpanWithLogger(ctx, "test-span")
	if span == nil {
		t.Fatal("StartSpanWithLogger() returned nil span")
	}
	defer span.End()

	if logger == nil {
		t.Fatal("StartSpanWithLogger() returned nil logger")
	}

	if newCtx == ctx {
		t.Error("StartSpanWithLogger() should return a new context")
	}

	// Note: When OTel is disabled, noop tracer creates invalid span contexts
	// This is expected behavior - the span still works but doesn't record
	t.Logf("Span context valid: %v", span.SpanContext().IsValid())
}

func TestTelemetry_Providers(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	// LoggerProvider may be nil if OTel logs are disabled
	lp := tel.LoggerProvider()
	t.Logf("LoggerProvider: %v", lp)

	// MeterProvider may be nil if OTel metrics are disabled
	mp := tel.MeterProvider()
	t.Logf("MeterProvider: %v", mp)

	// TracerProvider may be nil if OTel traces are disabled
	tp := tel.TracerProvider()
	t.Logf("TracerProvider: %v", tp)
}

func TestTelemetry_Shutdown(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "shutdown with nil providers",
			opts: &Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "shutdown with batch export",
			opts: &Options{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				BatchExport:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := New(ctx, tt.opts)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			// Should not error even if providers are nil
			if err := tel.Shutdown(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("Shutdown() failed: %v", err)
				}
			}

			// Should be safe to call multiple times
			if err := tel.Shutdown(ctx); err != nil {
				if !tt.wantErr {
					t.Errorf("Shutdown() second call failed: %v", err)
				}
			}
		})
	}
}

func TestTelemetry_ShutdownWithContext(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Test shutdown with a context
	shutdownCtx := context.Background()
	if err := tel.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown() with context failed: %v", err)
	}
}

func TestTelemetry_NestedSpans(t *testing.T) {
	ctx := context.Background()
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	// Start root span
	ctx, rootSpan := tel.StartSpan(ctx, "root")
	defer rootSpan.End()

	rootSpanContext := trace.SpanFromContext(ctx).SpanContext()
	t.Logf("Root span context valid: %v", rootSpanContext.IsValid())

	// Start child span
	ctx, childSpan := tel.StartSpan(ctx, "child")
	defer childSpan.End()

	childSpanContext := trace.SpanFromContext(ctx).SpanContext()
	t.Logf("Child span context valid: %v", childSpanContext.IsValid())

	// When OTel is enabled (traces provider exists), verify span relationships
	if tel.TracerProvider() != nil {
		if !rootSpanContext.IsValid() {
			t.Error("Root span context should be valid when traces are enabled")
		}
		if !childSpanContext.IsValid() {
			t.Error("Child span context should be valid when traces are enabled")
		}

		// Verify they are different spans
		if rootSpanContext.SpanID() == childSpanContext.SpanID() {
			t.Error("Root and child spans should have different span IDs")
		}

		// They should share the same trace ID
		if rootSpanContext.TraceID() != childSpanContext.TraceID() {
			t.Error("Root and child spans should share the same trace ID")
		}
	}
}

func TestTelemetry_CustomLogger(t *testing.T) {
	ctx := context.Background()

	// Test with custom logger provided
	customLogger := &mockLogger{}
	tel, err := New(ctx, &Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Logger:         customLogger,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer tel.Shutdown(ctx)

	// Verify custom logger is used
	if tel.Logger() != customLogger {
		t.Error("Custom logger was not used")
	}
}

func TestTelemetry_ConsoleOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		opts *Options
	}{
		{
			name: "console output enabled with color",
			opts: &Options{
				ServiceName:      "test-service",
				ServiceVersion:   "1.0.0",
				LogConsoleOutput: true,
				LogConsoleColor:  true,
			},
		},
		{
			name: "console output enabled without color",
			opts: &Options{
				ServiceName:      "test-service",
				ServiceVersion:   "1.0.0",
				LogConsoleOutput: true,
				LogConsoleColor:  false,
			},
		},
		{
			name: "console output disabled",
			opts: &Options{
				ServiceName:      "test-service",
				ServiceVersion:   "1.0.0",
				LogConsoleOutput: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := New(ctx, tt.opts)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}
			defer tel.Shutdown(ctx)

			if tel.Logger() == nil {
				t.Error("Logger should not be nil")
			}
		})
	}
}

// mockLogger implements the logger.Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) With() logger.Context                          { return &mockContext{} }
func (m *mockLogger) Trace() logger.Event                           { return &mockEvent{} }
func (m *mockLogger) Debug() logger.Event                           { return &mockEvent{} }
func (m *mockLogger) Info() logger.Event                            { return &mockEvent{} }
func (m *mockLogger) Warn() logger.Event                            { return &mockEvent{} }
func (m *mockLogger) Error() logger.Event                           { return &mockEvent{} }
func (m *mockLogger) Fatal() logger.Event                           { return &mockEvent{} }
func (m *mockLogger) Panic() logger.Event                           { return &mockEvent{} }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) SetLevel(level logger.Level)                   {}
func (m *mockLogger) Level() logger.Level                           { return logger.InfoLevel }
func (m *mockLogger) Output(w io.Writer) logger.Logger              { return m }

// mockContext implements the logger.Context interface for testing
type mockContext struct{}

func (c *mockContext) Logger() logger.Logger                    { return &mockLogger{} }
func (c *mockContext) Str(key, val string) logger.Context       { return c }
func (c *mockContext) Int(key string, val int) logger.Context   { return c }
func (c *mockContext) Bool(key string, val bool) logger.Context { return c }
func (c *mockContext) Err(error) logger.Context                 { return c }
func (c *mockContext) Ctx(context.Context) logger.Context       { return c }

// mockEvent implements the logger.Event interface for testing
type mockEvent struct{}

func (e *mockEvent) Str(key, val string) logger.Event             { return e }
func (e *mockEvent) Int(key string, val int) logger.Event         { return e }
func (e *mockEvent) Int64(key string, val int64) logger.Event     { return e }
func (e *mockEvent) Uint64(key string, val uint64) logger.Event   { return e }
func (e *mockEvent) Float64(key string, val float64) logger.Event { return e }
func (e *mockEvent) Bool(key string, val bool) logger.Event       { return e }
func (e *mockEvent) Err(err error) logger.Event                   { return e }
func (e *mockEvent) Ctx(ctx context.Context) logger.Event         { return e }
func (e *mockEvent) Msg(msg string)                               {}
func (e *mockEvent) Msgf(format string, v ...interface{})         {}
func (e *mockEvent) Send()                                        {}

// Note: Tests with OTel enabled are not included because they require
// an actual OTLP endpoint running and would block/timeout without one.
// The provider creation code paths are indirectly tested through the
// providers_test.go file with nil checks when providers are disabled.
// Integration tests with real OTLP endpoints should be run separately.
