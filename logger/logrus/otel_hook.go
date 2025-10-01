package logrus

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// OTelHook is a logrus hook that sends logs to OpenTelemetry.
type OTelHook struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// NewOTelHook creates a new OpenTelemetry hook for logrus.
func NewOTelHook(serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) *OTelHook {
	if loggerProvider == nil {
		return nil
	}

	return &OTelHook{
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

// Levels returns the log levels this hook should be triggered for.
func (h *OTelHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log event is fired.
func (h *OTelHook) Fire(entry *logrus.Entry) error {
	if h == nil {
		return nil
	}

	// Convert logrus level to OTel severity
	severity := h.logrusLevelToOTel(entry.Level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(entry.Time)
	logRecord.SetBody(log.StringValue(entry.Message))
	logRecord.SetSeverity(severity)

	// Add fields as attributes
	for key, value := range entry.Data {
		// Convert value to OTel attribute
		logRecord.AddAttributes(log.String(key, formatValue(value)))
	}

	// Emit the log record
	h.logger.Emit(context.Background(), logRecord)

	return nil
}

// logrusLevelToOTel converts logrus.Level to log.Severity.
func (h *OTelHook) logrusLevelToOTel(level logrus.Level) log.Severity {
	switch level {
	case logrus.TraceLevel:
		return log.SeverityTrace
	case logrus.DebugLevel:
		return log.SeverityDebug
	case logrus.InfoLevel:
		return log.SeverityInfo
	case logrus.WarnLevel:
		return log.SeverityWarn
	case logrus.ErrorLevel:
		return log.SeverityError
	case logrus.FatalLevel:
		return log.SeverityFatal
	case logrus.PanicLevel:
		return log.SeverityFatal4
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
