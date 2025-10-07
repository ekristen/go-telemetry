package zerolog

import (
	"context"
	"io"

	"github.com/rs/zerolog"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/ekristen/go-telemetry/logger"
)

// Logger wraps zerolog.Logger and implements the logger.Logger interface.
// It provides full access to zerolog's API while optionally integrating with OTel.
type Logger struct {
	zerolog.Logger
	otelEnabled    bool
	serviceName    string
	serviceVersion string
	baseLogger     zerolog.Logger // Store base logger for rebuilding with OTel
}

// Options configures the zerolog logger.
type Options struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
	Output         io.Writer
	EnableCaller   bool
	EnableColor    bool
}

// New creates a new zerolog logger with optional OTel integration.
func New(opts Options) *Logger {
	var zlog zerolog.Logger

	if opts.Output != nil {
		zlog = zerolog.New(opts.Output)
	} else {
		zlog = zerolog.New(io.Discard)
	}

	// Add timestamp and caller if requested
	zlog = zlog.With().Timestamp().Logger()
	if opts.EnableCaller {
		// Skip 3 frames: runtime.Caller -> zerolog internals -> our Event wrapper -> actual caller
		zlog = zlog.With().CallerWithSkipFrameCount(3).Logger()
	}

	// Store base logger before adding hooks
	baseLogger := zlog

	// If we have a logger provider, add our OTel hook
	if opts.LoggerProvider != nil {
		hook := NewOTelHook(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if hook != nil {
			zlog = zlog.Hook(hook)
		}
	}

	return &Logger{
		Logger:         zlog,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseLogger:     baseLogger,
	}
}

// SetOptions implements logger.LoggerOptionsUpdater.
// It updates the logger's service name and version.
func (l *Logger) SetOptions(serviceName, serviceVersion string) {
	l.serviceName = serviceName
	l.serviceVersion = serviceVersion
}

// UpdateLoggerProvider implements logger.LoggerProviderUpdater.
// It updates the logger to use the provided OTel logger provider.
func (l *Logger) UpdateLoggerProvider(provider *sdklog.LoggerProvider) {
	if provider == nil {
		return
	}

	// Create OTel hook
	hook := NewOTelHook(l.serviceName, l.serviceVersion, provider)
	if hook == nil {
		return
	}

	// Add hook to base logger
	l.Logger = l.baseLogger.Hook(hook)
	l.otelEnabled = true
}

// WrapOptions configures wrapping of an existing zerolog logger.
type WrapOptions struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
}

// Wrap wraps an existing zerolog.Logger instance with optional OTel integration.
// This allows you to bring your own pre-configured zerolog logger and add
// OTel integration to it.
func Wrap(zlog zerolog.Logger, opts WrapOptions) *Logger {
	// Store base logger before adding hooks
	baseLogger := zlog

	// If we have a logger provider, add our OTel hook
	if opts.LoggerProvider != nil {
		hook := NewOTelHook(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if hook != nil {
			zlog = zlog.Hook(hook)
		}
	}

	return &Logger{
		Logger:         zlog,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseLogger:     baseLogger,
	}
}

// With returns a context that can be used to add fields to the logger.
func (l *Logger) With() logger.Context {
	return &Context{
		ctx: l.Logger.With(),
	}
}

// Trace returns an event for trace level logging.
func (l *Logger) Trace() logger.Event {
	return &Event{event: l.Logger.Trace()}
}

// Debug returns an event for debug level logging.
func (l *Logger) Debug() logger.Event {
	return &Event{event: l.Logger.Debug()}
}

// Info returns an event for info level logging.
func (l *Logger) Info() logger.Event {
	return &Event{event: l.Logger.Info()}
}

// Warn returns an event for warn level logging.
func (l *Logger) Warn() logger.Event {
	return &Event{event: l.Logger.Warn()}
}

// Error returns an event for error level logging.
func (l *Logger) Error() logger.Event {
	return &Event{event: l.Logger.Error()}
}

// Fatal returns an event for fatal level logging.
func (l *Logger) Fatal() logger.Event {
	return &Event{event: l.Logger.Fatal()}
}

// Panic returns an event for panic level logging.
func (l *Logger) Panic() logger.Event {
	return &Event{event: l.Logger.Panic()}
}

// Level returns the current log level.
func (l *Logger) Level() logger.Level {
	return toLoggerLevel(l.Logger.GetLevel())
}

// SetLevel sets the log level.
func (l *Logger) SetLevel(level logger.Level) {
	l.Logger = l.Logger.Level(toZerologLevel(level))
}

// Output returns a new logger with the given output writer.
func (l *Logger) Output(w io.Writer) logger.Logger {
	return &Logger{
		Logger:      l.Logger.Output(w),
		otelEnabled: l.otelEnabled,
	}
}

// WithContext returns a logger with the given context.
func (l *Logger) WithContext(ctx context.Context) logger.Logger {
	return &Logger{
		Logger:      l.Logger.With().Ctx(ctx).Logger(),
		otelEnabled: l.otelEnabled,
	}
}

// Context wraps zerolog.Context.
type Context struct {
	ctx zerolog.Context
}

// Logger returns the logger with the added context.
func (c *Context) Logger() logger.Logger {
	return &Logger{Logger: c.ctx.Logger()}
}

// Str adds a string field.
func (c *Context) Str(key, val string) logger.Context {
	c.ctx = c.ctx.Str(key, val)
	return c
}

// Int adds an int field.
func (c *Context) Int(key string, val int) logger.Context {
	c.ctx = c.ctx.Int(key, val)
	return c
}

// Bool adds a bool field.
func (c *Context) Bool(key string, val bool) logger.Context {
	c.ctx = c.ctx.Bool(key, val)
	return c
}

// Err adds an error field.
func (c *Context) Err(err error) logger.Context {
	c.ctx = c.ctx.Err(err)
	return c
}

// Ctx adds context for distributed tracing.
func (c *Context) Ctx(ctx context.Context) logger.Context {
	c.ctx = c.ctx.Ctx(ctx)
	return c
}

// Event wraps zerolog.Event.
type Event struct {
	event *zerolog.Event
}

// Msg sends the event with the given message.
func (e *Event) Msg(msg string) {
	e.event.Msg(msg)
}

// Msgf sends the event with a formatted message.
func (e *Event) Msgf(format string, v ...interface{}) {
	e.event.Msgf(format, v...)
}

// Send sends the event without a message.
func (e *Event) Send() {
	e.event.Send()
}

// Str adds a string field to the event.
func (e *Event) Str(key, val string) logger.Event {
	e.event = e.event.Str(key, val)
	return e
}

// Int adds an int field to the event.
func (e *Event) Int(key string, val int) logger.Event {
	e.event = e.event.Int(key, val)
	return e
}

// Int64 adds an int64 field to the event.
func (e *Event) Int64(key string, val int64) logger.Event {
	e.event = e.event.Int64(key, val)
	return e
}

// Uint64 adds a uint64 field to the event.
func (e *Event) Uint64(key string, val uint64) logger.Event {
	e.event = e.event.Uint64(key, val)
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, val float64) logger.Event {
	e.event = e.event.Float64(key, val)
	return e
}

// Bool adds a bool field to the event.
func (e *Event) Bool(key string, val bool) logger.Event {
	e.event = e.event.Bool(key, val)
	return e
}

// Err adds an error field to the event.
func (e *Event) Err(err error) logger.Event {
	e.event = e.event.Err(err)
	return e
}

// Ctx adds context for distributed tracing.
func (e *Event) Ctx(ctx context.Context) logger.Event {
	e.event = e.event.Ctx(ctx)
	return e
}

// toZerologLevel converts logger.Level to zerolog.Level.
func toZerologLevel(level logger.Level) zerolog.Level {
	switch level {
	case logger.TraceLevel:
		return zerolog.TraceLevel
	case logger.DebugLevel:
		return zerolog.DebugLevel
	case logger.InfoLevel:
		return zerolog.InfoLevel
	case logger.WarnLevel:
		return zerolog.WarnLevel
	case logger.ErrorLevel:
		return zerolog.ErrorLevel
	case logger.FatalLevel:
		return zerolog.FatalLevel
	case logger.PanicLevel:
		return zerolog.PanicLevel
	case logger.Disabled:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// toLoggerLevel converts zerolog.Level to logger.Level.
func toLoggerLevel(level zerolog.Level) logger.Level {
	switch level {
	case zerolog.TraceLevel:
		return logger.TraceLevel
	case zerolog.DebugLevel:
		return logger.DebugLevel
	case zerolog.InfoLevel:
		return logger.InfoLevel
	case zerolog.WarnLevel:
		return logger.WarnLevel
	case zerolog.ErrorLevel:
		return logger.ErrorLevel
	case zerolog.FatalLevel:
		return logger.FatalLevel
	case zerolog.PanicLevel:
		return logger.PanicLevel
	case zerolog.Disabled:
		return logger.Disabled
	default:
		return logger.InfoLevel
	}
}
