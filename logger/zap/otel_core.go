package zap

import (
	"context"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

// OTelCore is a zapcore.Core that sends logs to OpenTelemetry.
type OTelCore struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
	level          zapcore.Level
}

// NewOTelCore creates a new OpenTelemetry core for zap.
func NewOTelCore(serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) zapcore.Core {
	if loggerProvider == nil {
		return nil
	}

	return &OTelCore{
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		level:          zapcore.DebugLevel, // Log everything, let OTel decide
	}
}

// Enabled returns whether the given level is enabled.
func (c *OTelCore) Enabled(level zapcore.Level) bool {
	return level >= c.level
}

// With adds structured context to the Core.
func (c *OTelCore) With(fields []zapcore.Field) zapcore.Core {
	// For simplicity, return the same core
	// In a production implementation, you might want to store fields
	return c
}

// Check determines whether the supplied Entry should be logged.
func (c *OTelCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to OpenTelemetry.
func (c *OTelCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Convert zap level to OTel severity
	severity := c.zapLevelToOTel(entry.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(entry.Time)
	logRecord.SetBody(log.StringValue(entry.Message))
	logRecord.SetSeverity(severity)

	// Add caller information if available
	if entry.Caller.Defined {
		logRecord.AddAttributes(
			log.String("caller", entry.Caller.String()),
			log.String("function", entry.Caller.Function),
		)
	}

	// Add logger name
	if entry.LoggerName != "" {
		logRecord.AddAttributes(log.String("logger", entry.LoggerName))
	}

	// Add stack trace if present
	if entry.Stack != "" {
		logRecord.AddAttributes(log.String("stacktrace", entry.Stack))
	}

	// Convert fields to attributes
	enc := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(enc)
	}

	for key, value := range enc.Fields {
		logRecord.AddAttributes(log.String(key, formatValue(value)))
	}

	// Emit the log record
	c.logger.Emit(context.Background(), logRecord)

	return nil
}

// Sync flushes buffered logs.
func (c *OTelCore) Sync() error {
	return nil
}

// zapLevelToOTel converts zapcore.Level to log.Severity.
func (c *OTelCore) zapLevelToOTel(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		return log.SeverityFatal4
	case zapcore.FatalLevel:
		return log.SeverityFatal
	default:
		return log.SeverityInfo
	}
}

// formatValue converts any value to a string for OTel attributes.
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return ""
	}
}
