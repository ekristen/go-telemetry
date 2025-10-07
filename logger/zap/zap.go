package zap

import (
	"context"
	"io"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ekristen/go-telemetry/logger"
)

// Logger wraps zap.Logger and implements the logger.Logger interface.
// It provides full access to zap's API while optionally integrating with OTel.
type Logger struct {
	*zap.Logger
	sugar          *zap.SugaredLogger
	otelEnabled    bool
	serviceName    string
	serviceVersion string
	baseCore       zapcore.Core // Store base core for rebuilding with OTel
	opts           Options      // Store options for rebuilding
}

// Options configures the zap logger.
type Options struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
	Output         io.Writer
	EnableCaller   bool
	Development    bool // Use development config (pretty printing)
	JSONFormat     bool // Use JSON encoder
}

// New creates a new zap logger with optional OTel integration.
func New(opts Options) *Logger {
	// Create encoder config
	var encoderCfg zapcore.EncoderConfig
	if opts.Development {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	// Create encoder
	var encoder zapcore.Encoder
	if opts.JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	// Create writer syncer
	var ws zapcore.WriteSyncer
	if opts.Output != nil {
		ws = zapcore.AddSync(opts.Output)
	} else {
		ws = zapcore.AddSync(io.Discard)
	}

	// Create core
	core := zapcore.NewCore(encoder, ws, zapcore.DebugLevel)

	// Add OTel core if we have a logger provider
	if opts.LoggerProvider != nil {
		otelCore := NewOTelCore(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if otelCore != nil {
			core = zapcore.NewTee(core, otelCore)
		}
	}

	// Create logger options
	zapOpts := []zap.Option{}
	if opts.EnableCaller {
		// AddCallerSkip(2) skips: our Event wrapper methods -> actual caller
		zapOpts = append(zapOpts, zap.AddCaller(), zap.AddCallerSkip(2))
	}
	zapOpts = append(zapOpts, zap.AddStacktrace(zapcore.ErrorLevel))

	// Create logger
	zapLogger := zap.New(core, zapOpts...)

	return &Logger{
		Logger:         zapLogger,
		sugar:          zapLogger.Sugar(),
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseCore:       zapcore.NewCore(encoder, ws, zapcore.DebugLevel),
		opts:           opts,
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

	// Create OTel core
	otelCore := NewOTelCore(l.serviceName, l.serviceVersion, provider)
	if otelCore == nil {
		return
	}

	// Combine base core with OTel core
	core := zapcore.NewTee(l.baseCore, otelCore)

	// Create logger options
	zapOpts := []zap.Option{}
	if l.opts.EnableCaller {
		// AddCallerSkip(2) skips: our Event wrapper methods -> actual caller
		zapOpts = append(zapOpts, zap.AddCaller(), zap.AddCallerSkip(2))
	}
	zapOpts = append(zapOpts, zap.AddStacktrace(zapcore.ErrorLevel))

	// Rebuild logger with new core
	zapLogger := zap.New(core, zapOpts...)

	// Update the logger
	l.Logger = zapLogger
	l.sugar = zapLogger.Sugar()
	l.otelEnabled = true
}

// WrapOptions configures wrapping of an existing zap logger.
type WrapOptions struct {
	ServiceName    string
	ServiceVersion string
	LoggerProvider *sdklog.LoggerProvider
}

// Wrap wraps an existing zap.Logger instance with optional OTel integration.
// This allows you to bring your own pre-configured zap logger and add
// OTel integration to it by adding an OTel core.
func Wrap(zapLogger *zap.Logger, opts WrapOptions) *Logger {
	// Extract the core from the existing logger to use as base
	baseCore := zapLogger.Core()

	// If we have a logger provider, add OTel core
	if opts.LoggerProvider != nil {
		otelCore := NewOTelCore(opts.ServiceName, opts.ServiceVersion, opts.LoggerProvider)
		if otelCore != nil {
			// Wrap the existing logger's core with OTel core
			zapLogger = zapLogger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				return zapcore.NewTee(c, otelCore)
			}))
		}
	}

	return &Logger{
		Logger:         zapLogger,
		sugar:          zapLogger.Sugar(),
		otelEnabled:    opts.LoggerProvider != nil,
		serviceName:    opts.ServiceName,
		serviceVersion: opts.ServiceVersion,
		baseCore:       baseCore,
		opts:           Options{}, // Empty options since this is wrapped
	}
}

// With returns a context that can be used to add fields to the logger.
func (l *Logger) With() logger.Context {
	return &Context{
		logger: l,
		fields: []zap.Field{},
	}
}

// Trace returns an event for trace level logging.
// Note: Zap doesn't have a native trace level, so we use DebugLevel - 1
func (l *Logger) Trace() logger.Event {
	return &Event{logger: l, level: zapcore.DebugLevel - 1}
}

// Debug returns an event for debug level logging.
func (l *Logger) Debug() logger.Event {
	return &Event{logger: l, level: zapcore.DebugLevel}
}

// Info returns an event for info level logging.
func (l *Logger) Info() logger.Event {
	return &Event{logger: l, level: zapcore.InfoLevel}
}

// Warn returns an event for warn level logging.
func (l *Logger) Warn() logger.Event {
	return &Event{logger: l, level: zapcore.WarnLevel}
}

// Error returns an event for error level logging.
func (l *Logger) Error() logger.Event {
	return &Event{logger: l, level: zapcore.ErrorLevel}
}

// Fatal returns an event for fatal level logging.
func (l *Logger) Fatal() logger.Event {
	return &Event{logger: l, level: zapcore.FatalLevel}
}

// Panic returns an event for panic level logging.
func (l *Logger) Panic() logger.Event {
	return &Event{logger: l, level: zapcore.PanicLevel}
}

// Level returns the current log level.
func (l *Logger) Level() logger.Level {
	// Zap doesn't have a simple way to get the level, so we return InfoLevel
	return logger.InfoLevel
}

// SetLevel sets the log level.
func (l *Logger) SetLevel(level logger.Level) {
	// Zap's level is set at core creation time, so this is a no-op
	// To properly support this, you'd need to use zap.NewAtomicLevel()
}

// Output returns a new logger with the given output writer.
func (l *Logger) Output(w io.Writer) logger.Logger {
	// Create new logger with new output
	ws := zapcore.AddSync(w)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		ws,
		zapcore.DebugLevel,
	)
	newZapLogger := zap.New(core)

	return &Logger{
		Logger:      newZapLogger,
		sugar:       newZapLogger.Sugar(),
		otelEnabled: l.otelEnabled,
	}
}

// WithContext returns a logger with the given context.
func (l *Logger) WithContext(ctx context.Context) logger.Logger {
	// Zap doesn't have built-in context support like zerolog
	// You would need to extract trace info from context and add as fields
	return l
}

// Context wraps fields for building context.
type Context struct {
	logger *Logger
	fields []zap.Field
}

// Logger returns the logger with the added context.
func (c *Context) Logger() logger.Logger {
	return &Logger{
		Logger:      c.logger.Logger.With(c.fields...),
		sugar:       c.logger.Logger.With(c.fields...).Sugar(),
		otelEnabled: c.logger.otelEnabled,
	}
}

// Str adds a string field.
func (c *Context) Str(key, val string) logger.Context {
	c.fields = append(c.fields, zap.String(key, val))
	return c
}

// Int adds an int field.
func (c *Context) Int(key string, val int) logger.Context {
	c.fields = append(c.fields, zap.Int(key, val))
	return c
}

// Bool adds a bool field.
func (c *Context) Bool(key string, val bool) logger.Context {
	c.fields = append(c.fields, zap.Bool(key, val))
	return c
}

// Err adds an error field.
func (c *Context) Err(err error) logger.Context {
	c.fields = append(c.fields, zap.Error(err))
	return c
}

// Ctx adds context for distributed tracing.
func (c *Context) Ctx(ctx context.Context) logger.Context {
	// Would need to extract trace info from context
	return c
}

// Event wraps zap fields for logging events.
type Event struct {
	logger *Logger
	level  zapcore.Level
	fields []zap.Field
}

// Msg sends the event with the given message.
func (e *Event) Msg(msg string) {
	if ce := e.logger.Logger.Check(e.level, msg); ce != nil {
		ce.Write(e.fields...)
	}
}

// Msgf sends the event with a formatted message.
func (e *Event) Msgf(format string, v ...interface{}) {
	e.logger.sugar.Logf(e.level, format, v...)
}

// Send sends the event without a message.
func (e *Event) Send() {
	e.Msg("")
}

// Str adds a string field to the event.
func (e *Event) Str(key, val string) logger.Event {
	e.fields = append(e.fields, zap.String(key, val))
	return e
}

// Int adds an int field to the event.
func (e *Event) Int(key string, val int) logger.Event {
	e.fields = append(e.fields, zap.Int(key, val))
	return e
}

// Int64 adds an int64 field to the event.
func (e *Event) Int64(key string, val int64) logger.Event {
	e.fields = append(e.fields, zap.Int64(key, val))
	return e
}

// Uint64 adds a uint64 field to the event.
func (e *Event) Uint64(key string, val uint64) logger.Event {
	e.fields = append(e.fields, zap.Uint64(key, val))
	return e
}

// Float64 adds a float64 field to the event.
func (e *Event) Float64(key string, val float64) logger.Event {
	e.fields = append(e.fields, zap.Float64(key, val))
	return e
}

// Bool adds a bool field to the event.
func (e *Event) Bool(key string, val bool) logger.Event {
	e.fields = append(e.fields, zap.Bool(key, val))
	return e
}

// Err adds an error field to the event.
func (e *Event) Err(err error) logger.Event {
	e.fields = append(e.fields, zap.Error(err))
	return e
}

// Ctx adds context for distributed tracing.
func (e *Event) Ctx(ctx context.Context) logger.Event {
	// Would need to extract trace info from context
	return e
}

// toZapLevel converts logger.Level to zapcore.Level.
func toZapLevel(level logger.Level) zapcore.Level {
	switch level {
	case logger.TraceLevel:
		// Zap doesn't have a native trace level, use DebugLevel - 1
		return zapcore.DebugLevel - 1
	case logger.DebugLevel:
		return zapcore.DebugLevel
	case logger.InfoLevel:
		return zapcore.InfoLevel
	case logger.WarnLevel:
		return zapcore.WarnLevel
	case logger.ErrorLevel:
		return zapcore.ErrorLevel
	case logger.FatalLevel:
		return zapcore.FatalLevel
	case logger.PanicLevel:
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// toLoggerLevel converts zapcore.Level to logger.Level.
func toLoggerLevel(level zapcore.Level) logger.Level {
	switch {
	case level < zapcore.DebugLevel:
		// Treat anything below Debug as Trace
		return logger.TraceLevel
	case level == zapcore.DebugLevel:
		return logger.DebugLevel
	case level == zapcore.InfoLevel:
		return logger.InfoLevel
	case level == zapcore.WarnLevel:
		return logger.WarnLevel
	case level == zapcore.ErrorLevel:
		return logger.ErrorLevel
	case level == zapcore.FatalLevel:
		return logger.FatalLevel
	case level == zapcore.PanicLevel:
		return logger.PanicLevel
	default:
		return logger.InfoLevel
	}
}
