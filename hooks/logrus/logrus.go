package logrus

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// LogrusOTelHook is a logrus hook that sends logs to OpenTelemetry.
// This hook is designed to be non-invasive and can be attached to any logrus logger
// without affecting caller reporting or other logrus functionality.
//
// Example usage (standalone, without wrapper):
//
//	// Create your own logrus logger with full control
//	log := logrus.New()
//	log.SetReportCaller(true)  // Caller info will be accurate
//	log.SetFormatter(&logrus.JSONFormatter{})
//
//	// Create telemetry for OTel
//	t, _ := telemetry.New(ctx, &telemetry.Options{
//	    ServiceName: "my-service",
//	})
//
//	// Attach OTel hook to existing logger
//	hook := logruslogger.NewLogrusOTelHook("my-service", "v1.0.0", t.LoggerProvider())
//	log.AddHook(hook)
//
//	// Use logger as normal - logs go to both console and OTel
//	log.WithFields(logrus.Fields{"key": "value"}).Info("Hello")
type LogrusOTelHook struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// New creates a new OpenTelemetry hook for logrus.
// This is the recommended way to add OTel integration to an existing logrus logger.
//
// The hook can be attached to any logrus.Logger instance using AddHook():
//
//	hook := New("my-service", "v1.0.0", loggerProvider)
//	myLogger.AddHook(hook)
//
// Returns nil if loggerProvider is nil.
func New(serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) *LogrusOTelHook {
	if loggerProvider == nil {
		return nil
	}

	return &LogrusOTelHook{
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

// Levels returns the log levels this hook should be triggered for.
func (h *LogrusOTelHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log event is fired.
func (h *LogrusOTelHook) Fire(entry *logrus.Entry) error {
	if h == nil {
		return nil
	}

	// Convert logrus level to OTel severity
	severity, severityText := h.logrusLevelToOTel(entry.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(entry.Time)
	logRecord.SetBody(log.StringValue(entry.Message))
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(severityText)

	// Add fields as attributes
	for key, value := range entry.Data {
		// Skip trace fields as they're already set on the record
		if key == "trace_id" || key == "span_id" {
			continue
		}

		// Convert value to OTel attribute
		logRecord.AddAttributes(log.String(key, formatValue(value)))
	}

	// Emit the log record
	// Use entry's context if available, otherwise background
	ctx := entry.Context
	if ctx == nil {
		ctx = context.TODO()
	}
	h.logger.Emit(ctx, logRecord)

	return nil
}

// logrusLevelToOTel converts logrus.Level to log.Severity.
func (h *LogrusOTelHook) logrusLevelToOTel(level logrus.Level) (log.Severity, string) {
	switch level {
	case logrus.TraceLevel:
		return log.SeverityTrace, "TRACE"
	case logrus.DebugLevel:
		return log.SeverityDebug, "DEBUG"
	case logrus.InfoLevel:
		return log.SeverityInfo, "INFO"
	case logrus.WarnLevel:
		return log.SeverityWarn, "WARN"
	case logrus.ErrorLevel:
		return log.SeverityError, "ERROR"
	case logrus.FatalLevel:
		return log.SeverityFatal, "FATAL"
	case logrus.PanicLevel:
		return log.SeverityFatal4, "FATAL"
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
