package logrus

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/ekristen/go-telemetry/logger"
)

// Logger wraps logrus.Logger and implements the logger.Logger interface.
// It provides full access to logrus's API while optionally integrating with OTel.
type Logger struct {
	*logrus.Logger
	otelEnabled    bool
	serviceName    string
	serviceVersion string
}

// Options configures the logrus logger.
type Options struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
	Output         io.Writer
	EnableColor    bool
	JSONFormat     bool
}

// New creates a new logrus logger with optional OTel integration.
func New(opts Options) *Logger {
	log := logrus.New()

	if opts.Output != nil {
		log.SetOutput(opts.Output)
	}

	// Set formatter
	if opts.JSONFormat {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   opts.EnableColor,
		})
	}

	// Add caller information
	log.SetReportCaller(true)

	// If we have a logger provider, add our OTel hook
	if opts.LoggerProvider != nil {
		hook := NewOTelHook(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if hook != nil {
			log.AddHook(hook)
		}
	}

	return &Logger{
		Logger:         log,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
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

	// Create and add OTel hook
	hook := NewOTelHook(l.serviceName, l.serviceVersion, provider)
	if hook != nil {
		l.AddHook(hook)
		l.otelEnabled = true
	}
}

// WrapOptions configures wrapping of an existing logrus logger.
type WrapOptions struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
}

// Wrap wraps an existing logrus.Logger instance with optional OTel integration.
// This allows you to bring your own pre-configured logrus logger and add
// OTel integration to it.
func Wrap(log *logrus.Logger, opts WrapOptions) *Logger {
	// If we have a logger provider, add our OTel hook
	if opts.LoggerProvider != nil {
		hook := NewOTelHook(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if hook != nil {
			log.AddHook(hook)
		}
	}

	return &Logger{
		Logger:         log,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
	}
}

// With returns a context that can be used to add fields to the logger.
func (l *Logger) With() logger.Context {
	return &Context{
		entry: logrus.NewEntry(l.Logger),
	}
}

// Trace returns an event for trace level logging.
func (l *Logger) Trace() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.TraceLevel}
}

// Debug returns an event for debug level logging.
func (l *Logger) Debug() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.DebugLevel}
}

// Info returns an event for info level logging.
func (l *Logger) Info() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.InfoLevel}
}

// Warn returns an event for warn level logging.
func (l *Logger) Warn() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.WarnLevel}
}

// Error returns an event for error level logging.
func (l *Logger) Error() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.ErrorLevel}
}

// Fatal returns an event for fatal level logging.
func (l *Logger) Fatal() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.FatalLevel}
}

// Panic returns an event for panic level logging.
func (l *Logger) Panic() logger.Event {
	return &Event{entry: l.Logger.WithFields(logrus.Fields{}), level: logrus.PanicLevel}
}

// Level returns the current log level.
func (l *Logger) Level() logger.Level {
	return toLoggerLevel(l.Logger.GetLevel())
}

// SetLevel sets the log level.
func (l *Logger) SetLevel(level logger.Level) {
	l.Logger.SetLevel(toLogrusLevel(level))
}

// Output returns a new logger with the given output writer.
func (l *Logger) Output(w io.Writer) logger.Logger {
	newLog := logrus.New()
	newLog.SetOutput(w)
	newLog.SetFormatter(l.Logger.Formatter)
	newLog.SetLevel(l.Logger.GetLevel())
	newLog.SetReportCaller(l.Logger.ReportCaller)

	// Copy hooks
	for k, v := range l.Logger.Hooks {
		newLog.Hooks[k] = v
	}

	return &Logger{
		Logger:      newLog,
		otelEnabled: l.otelEnabled,
	}
}

// WithContext returns a logger with the given context.
func (l *Logger) WithContext(ctx context.Context) logger.Logger {
	return &Logger{
		Logger:      l.Logger.WithContext(ctx).Logger,
		otelEnabled: l.otelEnabled,
	}
}

// Context wraps logrus.Entry for building context.
type Context struct {
	entry *logrus.Entry
}

// Logger returns the logger with the added context.
func (c *Context) Logger() logger.Logger {
	return &Logger{Logger: c.entry.Logger}
}

// Str adds a string field.
func (c *Context) Str(key, val string) logger.Context {
	c.entry = c.entry.WithField(key, val)
	return c
}

// Int adds an int field.
func (c *Context) Int(key string, val int) logger.Context {
	c.entry = c.entry.WithField(key, val)
	return c
}

// Bool adds a bool field.
func (c *Context) Bool(key string, val bool) logger.Context {
	c.entry = c.entry.WithField(key, val)
	return c
}

// Err adds an error field.
func (c *Context) Err(err error) logger.Context {
	c.entry = c.entry.WithError(err)
	return c
}

// Ctx adds context for distributed tracing.
func (c *Context) Ctx(ctx context.Context) logger.Context {
	c.entry = c.entry.WithContext(ctx)
	return c
}

// Event wraps logrus.Entry for logging events.
type Event struct {
	entry *logrus.Entry
	level logrus.Level
}

// Msg sends the event with the given message.
func (e *Event) Msg(msg string) {
	e.entry.Log(e.level, msg)
}

// Msgf sends the event with a formatted message.
func (e *Event) Msgf(format string, v ...interface{}) {
	e.entry.Logf(e.level, format, v...)
}

// Send sends the event without a message.
func (e *Event) Send() {
	e.entry.Log(e.level, "")
}

// Str adds a string field to the event.
func (e *Event) Str(key, val string) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Int adds an int field to the event.
func (e *Event) Int(key string, val int) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Int64 adds an int64 field to the event.
func (e *Event) Int64(key string, val int64) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Uint64 adds a uint64 field to the event.
func (e *Event) Uint64(key string, val uint64) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, val float64) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Bool adds a bool field to the event.
func (e *Event) Bool(key string, val bool) logger.Event {
	e.entry = e.entry.WithField(key, val)
	return e
}

// Err adds an error field to the event.
func (e *Event) Err(err error) logger.Event {
	e.entry = e.entry.WithError(err)
	return e
}

// Ctx adds context for distributed tracing.
func (e *Event) Ctx(ctx context.Context) logger.Event {
	e.entry = e.entry.WithContext(ctx)
	return e
}

// toLogrusLevel converts logger.Level to logrus.Level.
func toLogrusLevel(level logger.Level) logrus.Level {
	switch level {
	case logger.TraceLevel:
		return logrus.TraceLevel
	case logger.DebugLevel:
		return logrus.DebugLevel
	case logger.InfoLevel:
		return logrus.InfoLevel
	case logger.WarnLevel:
		return logrus.WarnLevel
	case logger.ErrorLevel:
		return logrus.ErrorLevel
	case logger.FatalLevel:
		return logrus.FatalLevel
	case logger.PanicLevel:
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// toLoggerLevel converts logrus.Level to logger.Level.
func toLoggerLevel(level logrus.Level) logger.Level {
	switch level {
	case logrus.TraceLevel:
		return logger.TraceLevel
	case logrus.DebugLevel:
		return logger.DebugLevel
	case logrus.InfoLevel:
		return logger.InfoLevel
	case logrus.WarnLevel:
		return logger.WarnLevel
	case logrus.ErrorLevel:
		return logger.ErrorLevel
	case logrus.FatalLevel:
		return logger.FatalLevel
	case logrus.PanicLevel:
		return logger.PanicLevel
	default:
		return logger.InfoLevel
	}
}
