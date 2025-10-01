package logger

import (
	"context"
	"io"

	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// Logger is a common interface for structured loggers.
// This allows swapping between different logging implementations
// while maintaining a consistent API.
type Logger interface {
	// With returns a child logger with additional context fields.
	With() Context

	// Trace logs a trace level message (more verbose than debug).
	Trace() Event
	// Debug logs a debug level message.
	Debug() Event
	// Info logs an info level message.
	Info() Event
	// Warn logs a warn level message.
	Warn() Event
	// Error logs an error level message.
	Error() Event
	// Fatal logs a fatal level message and exits.
	Fatal() Event
	// Panic logs a panic level message and panics.
	Panic() Event

	// Level returns the current log level.
	Level() Level
	// SetLevel sets the log level.
	SetLevel(level Level)

	// Output returns a new logger with the given output writer.
	Output(w io.Writer) Logger

	// WithContext returns a logger with the given context.
	WithContext(ctx context.Context) Logger
}

// Context allows setting context fields on a logger.
type Context interface {
	// Logger returns the logger with the added context.
	Logger() Logger

	// Str adds a string field.
	Str(key, val string) Context
	// Int adds an int field.
	Int(key string, val int) Context
	// Bool adds a bool field.
	Bool(key string, val bool) Context
	// Err adds an error field.
	Err(error) Context
	// Ctx adds context for distributed tracing.
	Ctx(context.Context) Context
}

// Event represents a logging event.
type Event interface {
	// Msg sends the event with the given message.
	Msg(msg string)
	// Msgf sends the event with a formatted message.
	Msgf(format string, v ...interface{})
	// Send sends the event without a message.
	Send()

	// Str adds a string field to the event.
	Str(key, val string) Event
	// Int adds an int field to the event.
	Int(key string, val int) Event
	// Int64 adds an int64 field to the event.
	Int64(key string, val int64) Event
	// Uint64 adds a uint64 field to the event.
	Uint64(key string, val uint64) Event
	// Float64 adds a float64 field to the event.
	Float64(key string, val float64) Event
	// Bool adds a bool field to the event.
	Bool(key string, val bool) Event
	// Err adds an error field to the event.
	Err(error) Event
	// Ctx adds context for distributed tracing.
	Ctx(context.Context) Event
}

// Level represents a log level.
type Level int8

const (
	// TraceLevel is for trace messages (more verbose than debug).
	TraceLevel Level = iota - 2
	// DebugLevel is for debug messages.
	DebugLevel
	// InfoLevel is for info messages.
	InfoLevel
	// WarnLevel is for warning messages.
	WarnLevel
	// ErrorLevel is for error messages.
	ErrorLevel
	// FatalLevel is for fatal messages.
	FatalLevel
	// PanicLevel is for panic messages.
	PanicLevel
	// Disabled disables logging.
	Disabled
)

// LoggerProviderUpdater is an optional interface that loggers can implement
// to allow updating the OTel logger provider after creation.
// This enables simpler logger instantiation without needing to pass
// service name, version, and logger provider upfront.
type LoggerProviderUpdater interface {
	// UpdateLoggerProvider updates the logger's OTel provider
	UpdateLoggerProvider(provider *sdklog.LoggerProvider)
}

// LoggerOptionsUpdater is an optional interface that loggers can implement
// to allow updating service name and version after creation.
// This enables the simplest instantiation pattern where service info
// is only specified once in telemetry.Options.
type LoggerOptionsUpdater interface {
	// SetOptions updates the logger's service name and version
	SetOptions(serviceName, serviceVersion string)
}
