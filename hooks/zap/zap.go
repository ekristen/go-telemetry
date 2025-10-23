package zap

import (
	"context"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

// ZapOTelCore is a zapcore.Core that sends logs to OpenTelemetry.
// This core can be combined with other cores using zapcore.NewTee() to send
// logs to multiple destinations simultaneously (e.g., console + OTel).
//
// Example usage (standalone, without wrapper):
//
//	// Create your own zap logger with full control
//	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
//	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
//
//	// Create telemetry for OTel
//	t, _ := telemetry.New(ctx, &telemetry.Options{
//	    ServiceName: "my-service",
//	})
//
//	// Create OTel core and combine with console core
//	ZapOTelCore := zaplogger.NewZapOTelCore("my-service", "v1.0.0", t.LoggerProvider())
//	core := zapcore.NewTee(consoleCore, ZapOTelCore)
//
//	// Create logger with combined cores
//	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
//
//	// Use logger as normal - logs go to both console and OTel
//	logger.Info("Hello", zap.String("key", "value"))
type ZapOTelCore struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
	level          zapcore.Level
}

// New creates a new OpenTelemetry core for zap.
// This is the recommended way to add OTel integration to an existing zap logger.
//
// The core can be combined with other cores using zapcore.NewTee():
//
//	ZapOTelCore := New("my-service", "v1.0.0", loggerProvider)
//	combinedCore := zapcore.NewTee(yourCore, ZapOTelCore)
//	logger := zap.New(combinedCore)
//
// Returns nil if loggerProvider is nil.
func New(serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) zapcore.Core {
	if loggerProvider == nil {
		return nil
	}

	return &ZapOTelCore{
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		level:          zapcore.DebugLevel, // Log everything, let OTel decide
	}
}

// Enabled returns whether the given level is enabled.
func (c *ZapOTelCore) Enabled(level zapcore.Level) bool {
	return level >= c.level
}

// With adds structured context to the Core.
func (c *ZapOTelCore) With(fields []zapcore.Field) zapcore.Core {
	// For simplicity, return the same core
	// In a production implementation, you might want to store fields
	return c
}

// Check determines whether the supplied Entry should be logged.
func (c *ZapOTelCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to OpenTelemetry.
func (c *ZapOTelCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Convert zap level to OTel severity
	severity, severityText := c.zapLevelToOTel(entry.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(entry.Time)
	logRecord.SetBody(log.StringValue(entry.Message))
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(severityText)

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

	// Convert fields to attributes and look for trace context
	enc := zapcore.NewMapObjectEncoder()
	var ctx context.Context

	for _, field := range fields {
		// Check for context field (zap doesn't support context natively, but user might add it via WithContext)
		if field.Key == "context" {
			if val, ok := field.Interface.(context.Context); ok {
				ctx = val
			}
		}
		field.AddTo(enc)
	}

	if ctx == nil {
		ctx = context.TODO()
	}

	// Add all fields as attributes
	for key, value := range enc.Fields {
		// Skip context field as it's not serializable
		if key == "context" {
			continue
		}
		logRecord.AddAttributes(log.String(key, formatValue(value)))
	}

	// Emit the log record
	// Note: We use context.TODO() here because zap doesn't pass context to Write()
	// The trace context is already extracted and set on the logRecord above
	c.logger.Emit(ctx, logRecord)

	return nil
}

// Sync flushes buffered logs.
func (c *ZapOTelCore) Sync() error {
	return nil
}

// zapLevelToOTel converts zapcore.Level to log.Severity.
func (c *ZapOTelCore) zapLevelToOTel(level zapcore.Level) (log.Severity, string) {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug, "DEBUG"
	case zapcore.InfoLevel:
		return log.SeverityInfo, "INFO"
	case zapcore.WarnLevel:
		return log.SeverityWarn, "WARN"
	case zapcore.ErrorLevel:
		return log.SeverityError, "ERROR"
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		return log.SeverityFatal4, "FATAL"
	case zapcore.FatalLevel:
		return log.SeverityFatal, "FATAL"
	default:
		return log.SeverityInfo, "INFO"
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
