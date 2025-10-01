package slog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// OTelHandler is a slog handler that sends logs to OpenTelemetry.
// It wraps another handler and forwards logs to both the wrapped handler and OTel.
type OTelHandler struct {
	base           slog.Handler
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// NewOTelHandler creates a new OpenTelemetry handler for slog.
// It wraps the provided base handler and also sends logs to OTel.
func NewOTelHandler(base slog.Handler, serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) *OTelHandler {
	if loggerProvider == nil {
		return nil
	}

	return &OTelHandler{
		base:           base,
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *OTelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle handles the Record.
// It sends the log to both the base handler and OTel.
func (h *OTelHandler) Handle(ctx context.Context, record slog.Record) error {
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
func (h *OTelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTelHandler{
		base:           h.base.WithAttrs(attrs),
		logger:         h.logger,
		serviceName:    h.serviceName,
		serviceVersion: h.serviceVersion,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *OTelHandler) WithGroup(name string) slog.Handler {
	return &OTelHandler{
		base:           h.base.WithGroup(name),
		logger:         h.logger,
		serviceName:    h.serviceName,
		serviceVersion: h.serviceVersion,
	}
}

// sendToOTel sends the log record to OpenTelemetry.
func (h *OTelHandler) sendToOTel(ctx context.Context, record slog.Record) {
	// Convert slog level to OTel severity
	severity := h.slogLevelToOTel(record.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(record.Time)
	logRecord.SetBody(log.StringValue(record.Message))
	logRecord.SetSeverity(severity)

	// Add attributes from the slog record
	record.Attrs(func(attr slog.Attr) bool {
		// Convert slog.Attr to OTel attribute
		logRecord.AddAttributes(h.convertAttr(attr))
		return true
	})

	// Emit the log record
	h.logger.Emit(ctx, logRecord)
}

// slogLevelToOTel converts slog.Level to log.Severity.
func (h *OTelHandler) slogLevelToOTel(level slog.Level) log.Severity {
	switch {
	case level < slog.LevelInfo:
		return log.SeverityDebug
	case level < slog.LevelWarn:
		return log.SeverityInfo
	case level < slog.LevelError:
		return log.SeverityWarn
	default:
		return log.SeverityError
	}
}

// convertAttr converts a slog.Attr to an OTel log.KeyValue.
func (h *OTelHandler) convertAttr(attr slog.Attr) log.KeyValue {
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
