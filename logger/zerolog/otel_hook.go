package zerolog

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// OTelHook is a zerolog hook that sends logs to OpenTelemetry.
type OTelHook struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// NewOTelHook creates a new OpenTelemetry hook for zerolog.
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

// Run implements the zerolog.Hook interface.
func (h *OTelHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h == nil {
		return
	}

	// Convert zerolog level to OTel severity
	severity := h.zerologLevelToOTel(level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(time.Now())
	logRecord.SetBody(log.StringValue(msg))
	logRecord.SetSeverity(severity)

	// Emit the log record
	h.logger.Emit(context.Background(), logRecord)
}

// zerologLevelToOTel converts zerolog.Level to log.Severity.
func (h *OTelHook) zerologLevelToOTel(level zerolog.Level) log.Severity {
	switch level {
	case zerolog.TraceLevel:
		return log.SeverityTrace
	case zerolog.DebugLevel:
		return log.SeverityDebug
	case zerolog.InfoLevel:
		return log.SeverityInfo
	case zerolog.WarnLevel:
		return log.SeverityWarn
	case zerolog.ErrorLevel:
		return log.SeverityError
	case zerolog.FatalLevel:
		return log.SeverityFatal
	case zerolog.PanicLevel:
		return log.SeverityFatal4
	default:
		return log.SeverityInfo
	}
}
