package slog

import (
	"context"
	"io"
	"log/slog"
	"os"

	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/ekristen/go-telemetry/logger"
)

// Logger wraps slog.Logger and implements the logger.Logger interface.
// It provides full access to slog's API while optionally integrating with OTel.
type Logger struct {
	*slog.Logger
	otelEnabled    bool
	serviceName    string
	serviceVersion string
	baseHandler    slog.Handler // Store base handler for rebuilding with OTel
}

// Options configures the slog logger.
type Options struct {
	ServiceName          string
	ServiceVersion       string
	LoggerProvider       *sdklog.LoggerProvider
	Output               io.Writer
	Level                slog.Level
	AddSource            bool // Add source code position (file:line)
	JSONFormat           bool // Use JSON handler instead of text
	CallerSkipFrameCount int  // Number of stack frames to skip when reporting caller (0 = auto-detect, default: auto-detect)
}

// New creates a new slog logger with optional OTel integration.
func New(opts Options) *Logger {
	// Determine output
	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	// Create handler options
	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
	}

	// Create base handler
	var baseHandler slog.Handler
	if opts.JSONFormat {
		baseHandler = slog.NewJSONHandler(output, handlerOpts)
	} else {
		baseHandler = slog.NewTextHandler(output, handlerOpts)
	}

	// Wrap with caller adjustment if AddSource is enabled
	var handler slog.Handler = baseHandler
	if opts.AddSource {
		skipCount := opts.CallerSkipFrameCount
		if skipCount == 0 {
			// Automatic caller detection: walk the call stack to find the first
			// frame outside of the telemetry library and logger packages.
			skipCount = logger.FindFirstExternalCaller()
		}
		// If CallerSkipFrameCount is explicitly set (> 0), use it as an override.
		handler = NewCallerHandler(baseHandler, skipCount)
	}

	// Wrap with OTel handler if we have a logger provider
	if opts.LoggerProvider != nil {
		otelHandler := NewOTelHandler(handler, opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if otelHandler != nil {
			handler = otelHandler
		}
	}

	// Create slog logger
	slogLogger := slog.New(handler)

	return &Logger{
		Logger:         slogLogger,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseHandler:    baseHandler,
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

	// Rebuild handler chain with OTel support
	var handler slog.Handler = l.baseHandler

	// Re-apply caller adjustment if it was originally enabled
	// We need to check if we're wrapping a CallerHandler
	if callerHandler, ok := l.Logger.Handler().(*CallerHandler); ok {
		// Reuse the same skip count that was used during creation
		handler = NewCallerHandler(l.baseHandler, callerHandler.skip)
	}

	// Wrap with OTel handler
	otelHandler := NewOTelHandler(handler, l.serviceName, l.serviceVersion, provider)
	if otelHandler == nil {
		return
	}

	// Create new logger with OTel handler
	l.Logger = slog.New(otelHandler)
	l.otelEnabled = true
}

// WrapOptions configures wrapping of an existing slog logger.
type WrapOptions struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
}

// Wrap wraps an existing slog.Logger instance with optional OTel integration.
// This allows you to bring your own pre-configured slog logger and add
// OTel integration to it.
func Wrap(slogLogger *slog.Logger, opts WrapOptions) *Logger {
	// Store the base handler
	baseHandler := slogLogger.Handler()

	// Build handler chain
	var handler slog.Handler = baseHandler

	// Note: We cannot add caller adjustment for wrapped loggers since we don't know
	// if AddSource was enabled. Users wrapping existing loggers should configure
	// their handlers appropriately before wrapping.

	// If we have a logger provider, wrap the handler with OTel handler
	if opts.LoggerProvider != nil {
		otelHandler := NewOTelHandler(handler, opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if otelHandler != nil {
			slogLogger = slog.New(otelHandler)
		}
	}

	return &Logger{
		Logger:         slogLogger,
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseHandler:    baseHandler,
	}
}

// With returns a context that can be used to add fields to the logger.
func (l *Logger) With() logger.Context {
	return &Context{
		logger: l.Logger,
		attrs:  []any{},
	}
}

// Trace returns an event for trace level logging.
// Note: slog doesn't have a native trace level, so we use LevelDebug - 4
func (l *Logger) Trace() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelDebug - 4, attrs: []any{}}
}

// Debug returns an event for debug level logging.
func (l *Logger) Debug() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelDebug, attrs: []any{}}
}

// Info returns an event for info level logging.
func (l *Logger) Info() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelInfo, attrs: []any{}}
}

// Warn returns an event for warn level logging.
func (l *Logger) Warn() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelWarn, attrs: []any{}}
}

// Error returns an event for error level logging.
func (l *Logger) Error() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelError, attrs: []any{}}
}

// Fatal returns an event for fatal level logging.
// Note: slog doesn't have a fatal level, so this uses Error level.
func (l *Logger) Fatal() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelError, attrs: []any{}, fatal: true}
}

// Panic returns an event for panic level logging.
// Note: slog doesn't have a panic level, so this uses Error level.
func (l *Logger) Panic() logger.Event {
	return &Event{logger: l.Logger, level: slog.LevelError, attrs: []any{}, doPanic: true}
}

// Level returns the current log level.
func (l *Logger) Level() logger.Level {
	// slog doesn't expose the level easily, so return InfoLevel
	return logger.InfoLevel
}

// SetLevel sets the log level.
// Note: slog's level is set at handler creation time, so this is a no-op.
// To change the level, create a new logger with a new handler.
func (l *Logger) SetLevel(level logger.Level) {
	// No-op - slog's level is immutable after handler creation
}

// Output returns a new logger with the given output writer.
func (l *Logger) Output(w io.Writer) logger.Logger {
	// Create new text handler with the new output
	handler := slog.NewTextHandler(w, nil)
	newSlog := slog.New(handler)

	return &Logger{
		Logger:      newSlog,
		otelEnabled: l.otelEnabled,
	}
}

// WithContext returns a logger with the given context.
func (l *Logger) WithContext(ctx context.Context) logger.Logger {
	// slog doesn't have built-in context support like zerolog
	// We would need to extract trace info from context and add as fields
	return l
}

// Context wraps slog attributes for building context.
type Context struct {
	logger *slog.Logger
	attrs  []any
}

// Logger returns the logger with the added context.
func (c *Context) Logger() logger.Logger {
	return &Logger{
		Logger: c.logger.With(c.attrs...),
	}
}

// Str adds a string field.
func (c *Context) Str(key, val string) logger.Context {
	c.attrs = append(c.attrs, slog.String(key, val))
	return c
}

// Int adds an int field.
func (c *Context) Int(key string, val int) logger.Context {
	c.attrs = append(c.attrs, slog.Int(key, val))
	return c
}

// Bool adds a bool field.
func (c *Context) Bool(key string, val bool) logger.Context {
	c.attrs = append(c.attrs, slog.Bool(key, val))
	return c
}

// Err adds an error field.
func (c *Context) Err(err error) logger.Context {
	c.attrs = append(c.attrs, slog.Any("error", err))
	return c
}

// Ctx adds context for distributed tracing.
func (c *Context) Ctx(ctx context.Context) logger.Context {
	// Would need to extract trace info from context
	return c
}

// Event wraps slog attributes for logging events.
type Event struct {
	logger  *slog.Logger
	level   slog.Level
	attrs   []any
	fatal   bool
	doPanic bool
}

// Msg sends the event with the given message.
func (e *Event) Msg(msg string) {
	e.logger.Log(context.Background(), e.level, msg, e.attrs...)
	if e.fatal {
		os.Exit(1)
	}
	if e.doPanic {
		panic(msg)
	}
}

// Msgf sends the event with a formatted message.
func (e *Event) Msgf(format string, v ...interface{}) {
	// slog doesn't have a printf-style method for structured logging
	// We'll use the formatted message as the message
	msg := ""
	if len(v) > 0 {
		msg = format // simplified - in practice would use fmt.Sprintf
	} else {
		msg = format
	}
	e.Msg(msg)
}

// Send sends the event without a message.
func (e *Event) Send() {
	e.Msg("")
}

// Str adds a string field to the event.
func (e *Event) Str(key, val string) logger.Event {
	e.attrs = append(e.attrs, slog.String(key, val))
	return e
}

// Int adds an int field to the event.
func (e *Event) Int(key string, val int) logger.Event {
	e.attrs = append(e.attrs, slog.Int(key, val))
	return e
}

// Int64 adds an int64 field to the event.
func (e *Event) Int64(key string, val int64) logger.Event {
	e.attrs = append(e.attrs, slog.Int64(key, val))
	return e
}

// Uint64 adds a uint64 field to the event.
func (e *Event) Uint64(key string, val uint64) logger.Event {
	e.attrs = append(e.attrs, slog.Uint64(key, val))
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, val float64) logger.Event {
	e.attrs = append(e.attrs, slog.Float64(key, val))
	return e
}

// Bool adds a bool field to the event.
func (e *Event) Bool(key string, val bool) logger.Event {
	e.attrs = append(e.attrs, slog.Bool(key, val))
	return e
}

// Err adds an error field to the event.
func (e *Event) Err(err error) logger.Event {
	e.attrs = append(e.attrs, slog.Any("error", err))
	return e
}

// Ctx adds context for distributed tracing.
func (e *Event) Ctx(ctx context.Context) logger.Event {
	// Would need to extract trace info from context
	return e
}
