package zerolog

import (
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// New is a zerolog hook that sends logs to OpenTelemetry.
// This hook can be attached to any zerolog logger without affecting caller
// reporting or other zerolog functionality.
//
// Example usage (standalone, without wrapper):
//
//	// Create your own zerolog logger with full control
//	log := zerolog.New(os.Stdout).
//	    With().
//	    Timestamp().
//	    Caller().  // Caller info will be accurate
//	    Logger()
//
//	// Create telemetry for OTel
//	t, _ := telemetry.New(ctx, &telemetry.Options{
//	    ServiceName: "my-service",
//	})
//
//	// Attach OTel hook to existing logger
//	hook := zerologger.New("my-service", "v1.0.0", t.LoggerProvider())
//	log = log.Hook(hook)
//
//	// Use logger as normal - logs go to both console and OTel
//	log.Info().Str("key", "value").Msg("Hello")
type ZerologOTelHook struct {
	logger         log.Logger
	serviceName    string
	serviceVersion string
}

// New creates a new OpenTelemetry hook for zerolog.
// This is the recommended way to add OTel integration to an existing zerolog logger.
//
// The hook can be attached to any zerolog.Logger using the Hook() method:
//
//	hook := New("my-service", "v1.0.0", loggerProvider)
//	logger := logger.Hook(hook)
//
// Returns nil if loggerProvider is nil.
func New(serviceName, serviceVersion string, loggerProvider *sdklog.LoggerProvider) *ZerologOTelHook {
	if loggerProvider == nil {
		return nil
	}

	return &ZerologOTelHook{
		logger:         loggerProvider.Logger(serviceName),
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
	}
}

// Run implements the zerolog.Hook interface.
func (h *ZerologOTelHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h == nil {
		return
	}

	// Convert zerolog level to OTel severity
	severity, severityText := h.zerologLevelToOTel(level)

	// Create OTel log record
	var logRecord log.Record
	logRecord.SetTimestamp(time.Now())
	logRecord.SetBody(log.StringValue(msg))
	logRecord.SetSeverity(severity)
	logRecord.SetSeverityText(severityText)

	// Emit the log record
	h.logger.Emit(e.GetCtx(), logRecord)
}

// zerologLevelToOTel converts zerolog.Level to log.Severity.
func (h *ZerologOTelHook) zerologLevelToOTel(level zerolog.Level) (log.Severity, string) {
	switch level {
	case zerolog.TraceLevel:
		return log.SeverityTrace, "TRACE"
	case zerolog.DebugLevel:
		return log.SeverityDebug, "DEBUG"
	case zerolog.InfoLevel:
		return log.SeverityInfo, "INFO"
	case zerolog.WarnLevel:
		return log.SeverityWarn, "WARN"
	case zerolog.ErrorLevel:
		return log.SeverityError, "ERROR"
	case zerolog.FatalLevel:
		return log.SeverityFatal, "FATAL"
	case zerolog.PanicLevel:
		return log.SeverityFatal4, "FATAL"
	default:
		return log.SeverityInfo, "INFO"
	}
}
