package slog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// SlogOTelHandler is a slog handler that sends logs to OpenTelemetry.
// It wraps another handler and forwards logs to both the wrapped handler and OTel.
//
// Example usage (standalone, without wrapper):
//
//	// Create your own slog logger with full control
//	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
//	    Level:     slog.LevelDebug,
//	    AddSource: true,  // Caller info (note: will show handler wrapper location)
//	})
//
//	// Create telemetry for OTel
//	t, _ := telemetry.New(ctx, &telemetry.Options{
//	    ServiceName: "my-service",
//	})
//
//	// Wrap handler with OTel handler
//	SlogOTelHandler := sloglogger.New(baseHandler, "my-service", "v1.0.0", t.LoggerProvider())
//	log := slog.New(SlogOTelHandler)
//
//	// Use logger as normal - logs go to both console and OTel
//	log.Info("Hello", slog.String("key", "value"))
type SlogOTelHandler struct {
	base           slog.Handler
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// New creates a new OpenTelemetry handler for slog.
// This is the recommended way to add OTel integration to an existing slog logger.
//
// It wraps the provided base handler and also sends logs to OTel:
//
//	SlogOTelHandler := New(yourHandler, "my-service", "v1.0.0", loggerProvider)
//	logger := slog.New(SlogOTelHandler)
//
// Note: Due to slog's architecture, AddSource caller information will point to the
// handler wrapper, not your actual code. For accurate caller info, use zap or zerolog.
//
// Returns nil if loggerProvider is nil.
func New(base slog.Handler, serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) *SlogOTelHandler {
	if loggerProvider == nil {
		return nil
	}

	return &SlogOTelHandler{
		base:           base,
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *SlogOTelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle handles the Record.
// It sends the log to both the base handler and OTel.
func (h *SlogOTelHandler) Handle(ctx context.Context, record slog.Record) error {
	// First, handle with the base handler
	if err := h.base.Handle(ctx, record); err != nil {
		return err
	}

	// Then send to OTel
	if h.logger != nil {
		h.sendToOTel(ctx, record)
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *SlogOTelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogOTelHandler{
		base:           h.base.WithAttrs(attrs),
		logger:         h.logger,
		serviceName:    h.serviceName,
		serviceVersion: h.serviceVersion,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *SlogOTelHandler) WithGroup(name string) slog.Handler {
	return &SlogOTelHandler{
		base:           h.base.WithGroup(name),
		logger:         h.logger,
		serviceName:    h.serviceName,
		serviceVersion: h.serviceVersion,
	}
}

// sendToOTel sends the log record to OpenTelemetry.
func (h *SlogOTelHandler) sendToOTel(ctx context.Context, record slog.Record) {
	// Convert slog level to OTel severity
	severity, severityText := h.slogLevelToOTel(record.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(record.Time)
	logRecord.SetBody(log.StringValue(record.Message))
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(severityText)

	// Add attributes from the slog record
	record.Attrs(func(attr slog.Attr) bool {
		// Skip trace fields as they're already set on the record
		if attr.Key == "trace_id" || attr.Key == "span_id" {
			return true
		}
		// Convert slog.Attr to OTel attribute
		logRecord.AddAttributes(h.convertAttr(attr))
		return true
	})

	// Emit the log record with the context
	h.logger.Emit(ctx, logRecord)
}

// slogLevelToOTel converts slog.Level to log.Severity.
func (h *SlogOTelHandler) slogLevelToOTel(level slog.Level) (log.Severity, string) {
	switch {
	case level < slog.LevelInfo:
		return log.SeverityDebug, "DEBUG"
	case level < slog.LevelWarn:
		return log.SeverityInfo, "INFO"
	case level < slog.LevelError:
		return log.SeverityWarn, "WARN"
	default:
		return log.SeverityError, "ERROR"
	}
}

// convertAttr converts a slog.Attr to an OTel log.KeyValue.
func (h *SlogOTelHandler) convertAttr(attr slog.Attr) log.KeyValue {
	key := attr.Key
	value := attr.Value

	switch value.Kind() {
	case slog.KindString:
		return log.String(key, value.String())
	case slog.KindInt64:
		return log.Int64(key, value.Int64())
	case slog.KindUint64:
		return log.Int64(key, int64(value.Uint64()))
	case slog.KindFloat64:
		return log.Float64(key, value.Float64())
	case slog.KindBool:
		return log.Bool(key, value.Bool())
	default:
		// For complex types, convert to string
		return log.String(key, value.String())
	}
}
