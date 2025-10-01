# go-telemetry

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/ekristen/go-telemetry.svg)](https://pkg.go.dev/github.com/ekristen/go-telemetry)

A flexible Go telemetry library with OpenTelemetry support that can be toggled on/off. Provides a standard logging interface while exposing full capabilities of underlying logging frameworks. I needed a telemetry library that was modern and yet flexible. I needed it to be easy to use and integrate with my existing codebase.

## AI

Some code and documentation in this project were created or refined with the assistance of AI tools. All contributions are reviewed and verified by human maintainers.

## Features

- **Multiple Logger Backends**: Support for zerolog, logrus, zap, and slog
- **Optional OpenTelemetry**: Toggle OTel on/off via environment variables
- **Full Logger Access**: Use the complete API of your chosen logger
- **OTel Integration**: Seamless integration with OTel logs, traces, and metrics when enabled
- **Zero Overhead**: No OTel overhead when disabled
- **Flexible Configuration**: Environment variables and functional options
- **Standard Interface**: Common logging interface across different backends

## Installation

```bash
go get github.com/ekristen/go-telemetry
```

## Quick Start

### Basic Usage (OTel Disabled)

```go
package main

import (
    "context"
    "github.com/ekristen/go-telemetry"
)

func main() {
    ctx := context.Background()

    t, err := telemetry.New(ctx, &telemetry.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
    })
    if err != nil {
        panic(err)
    }
    defer t.Shutdown(ctx)

    logger := t.Logger()
    logger.Info().Str("status", "running").Msg("Service started")
}
```

### With OpenTelemetry Enabled

OpenTelemetry is **automatically enabled** when standard OTel environment variables are set:

```bash
# Enable OTel by setting the OTLP endpoint
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# Optional: Set service info via environment
export OTEL_SERVICE_NAME=my-service
```

```go
// OTel auto-enabled if OTEL_EXPORTER_OTLP_ENDPOINT is set
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})
```

### Controlling Individual Signals

```bash
# Disable specific signals
export OTEL_TRACES_EXPORTER=none    # Disable traces
export OTEL_METRICS_EXPORTER=none   # Disable metrics
export OTEL_LOGS_EXPORTER=none      # Disable logs

# Force disable entire SDK
export OTEL_SDK_DISABLED=true
```

## Configuration Options

The `telemetry.Options` struct provides all configuration options:

```go
type Options struct {
    // ServiceName is the name of the service
    ServiceName string

    // ServiceVersion is the version of the service
    ServiceVersion string

    // Logger is the logger implementation to use (zerolog, logrus, zap, slog)
    // If nil, a default zerolog logger will be created
    Logger Logger

    // LogConsoleOutput controls whether logs are written to console (default: true)
    // Only used if Logger is nil
    LogConsoleOutput bool

    // LogConsoleColor controls whether console logs use colors (default: true)
    // Only used if Logger is nil
    LogConsoleColor bool

    // BatchExport controls whether telemetry is exported in batches or immediately
    // When true: Uses batch processors for better performance (higher latency)
    // When false (default): Uses simple/synchronous processors for immediate export
    // Recommended: false for development/debugging, true for high-volume production
    BatchExport bool
}
```

**OpenTelemetry is auto-configured via environment variables** - no manual enable flags needed!

You can pass `nil` to use defaults:
```go
t, err := telemetry.New(ctx, nil) // Uses default options
```

### Standard OpenTelemetry Environment Variables

The library follows the [OpenTelemetry specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for environment variables:

**SDK Control:**
- `OTEL_SDK_DISABLED` - Set to `true` to disable the entire SDK (default: `false`)

**Service Identity:**
- `OTEL_SERVICE_NAME` - Service name (can also be set in Options)
- `OTEL_RESOURCE_ATTRIBUTES` - Additional resource attributes

**Exporter Configuration:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (e.g., `http://localhost:4317`)
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` - Traces-specific endpoint
- `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` - Metrics-specific endpoint
- `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` - Logs-specific endpoint

**Signal Control:**
- `OTEL_TRACES_EXPORTER` - Traces exporter (default: `otlp`, set to `none` to disable)
- `OTEL_METRICS_EXPORTER` - Metrics exporter (default: `otlp`, set to `none` to disable)
- `OTEL_LOGS_EXPORTER` - Logs exporter (default: `otlp`, set to `none` to disable)

**How it works:**
- OTel is **disabled by default** (no-op providers)
- OTel is **automatically enabled** when any of the above endpoint/exporter variables are set
- Individual signals can be disabled by setting their exporter to `none`

### Batch vs Simple Export

The `BatchExport` option controls how telemetry data is sent to your OTel collector:

**Simple Export (default: `BatchExport: false`)**
- **Pros**: Immediate export, logs appear instantly, better for debugging
- **Cons**: Higher network overhead, more CPU usage per log/span
- **Use case**: Development, debugging, low-volume applications

**Batch Export (`BatchExport: true`)**
- **Pros**: Better performance, lower resource usage, higher throughput
- **Cons**: Delays of up to 30 seconds before export, data loss if app crashes
- **Use case**: High-volume production workloads

```go
// Development/debugging - see logs immediately
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    BatchExport:    false, // Default - immediate export
})

// Production - optimize for performance
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    BatchExport:    true, // Batch for better performance
})
```

**What gets batched:**
- **Logs**: Simple processor (immediate) vs Batch processor (periodic)
- **Traces**: Syncer (immediate) vs Batcher (periodic)
- **Metrics**: Always uses PeriodicReader (inherently batched)

## Logger Backends

The library supports multiple logger backends: zerolog (default), logrus, zap, and slog.

### Simplified Logger Pattern ✨

**No more repetition!** Create your logger with just logger-specific settings - the telemetry system automatically handles:
- ✅ Setting service name and version
- ✅ Adding OTel integration when `OTEL_EXPORTER_OTLP_ENDPOINT` is set
- ✅ Managing the logger provider lifecycle

**Example:**
```go
// Create logger with just logger config (no service info needed!)
zapLog := zaplogger.New(zaplogger.Options{
    Output:       os.Stdout,
    EnableCaller: true,
    Development:  true,
})

// Telemetry sets everything else automatically
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",  // Set once here
    ServiceVersion: "1.0.0",       // Set once here
    Logger:         zapLog,
})
```

This works for all logger backends: zerolog, logrus, zap, and slog!

### Zerolog (Default)

The library uses zerolog by default and exposes the full zerolog API:

```go
import zerologger "github.com/ekristen/go-telemetry/logger/zerolog"

logger := t.Logger()

// Type assert to access full zerolog capabilities
if zlog, ok := logger.(*zerologger.Logger); ok {
    // Full zerolog API access through the embedded Logger field
    zlog.Logger.Info().
        Str("user", "john").
        Int("age", 30).
        Time("timestamp", time.Now()).
        Msg("User logged in")

    // Use any zerolog feature
    contextLogger := zlog.Logger.With().
        Str("request_id", "req-123").
        Logger()
}
```

### Logrus

To use logrus instead of zerolog:

```go
import (
    "os"
    logruslogger "github.com/ekristen/go-telemetry/logger/logrus"
)

// Create logrus logger with just logger settings
// ServiceName/Version will be set automatically by telemetry
logrusLog := logruslogger.New(logruslogger.Options{
    Output:      os.Stdout,
    EnableColor: true,
    JSONFormat:  false,
})

// Pass to telemetry - it handles service info and OTel integration
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Logger:         logrusLog,
})

// Access full logrus API
log := t.Logger()
if logrusLogger, ok := log.(*logruslogger.Logger); ok {
    logrusLogger.Logger.WithFields(map[string]interface{}{
        "user_id": "123",
        "action":  "login",
    }).Info("User action")
}
```

### Zap

To use Uber's zap logger:

```go
import (
    "os"
    zaplogger "github.com/ekristen/go-telemetry/logger/zap"
    "go.uber.org/zap"
)

// Create zap logger with just logger settings
// ServiceName/Version will be set automatically by telemetry
zapLog := zaplogger.New(zaplogger.Options{
    Output:       os.Stdout,
    EnableCaller: true,
    Development:  true,  // Pretty console output
    JSONFormat:   false,
})

// Pass to telemetry - it handles service info and OTel integration
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Logger:         zapLog,
})

// Access full zap API
log := t.Logger()
if zapLogger, ok := log.(*zaplogger.Logger); ok {
    zapLogger.Logger.Info("Processing request",
        zap.String("user_id", "123"),
        zap.String("action", "login"),
        zap.Int("duration_ms", 150),
    )

    // Use SugaredLogger for printf-style
    zapLogger.Logger.Sugar().Infow("User action",
        "user", "john",
        "action", "login",
    )
}
```

### Slog

To use Go's standard library slog logger:

```go
import (
    "log/slog"
    "os"
    sloglogger "github.com/ekristen/go-telemetry/logger/slog"
)

// Create slog logger with just logger settings
// ServiceName/Version will be set automatically by telemetry
slogLog := sloglogger.New(sloglogger.Options{
    Output:     os.Stdout,
    Level:      slog.LevelDebug,
    AddSource:  true, // Add source file:line info
    JSONFormat: false,
})

// Pass to telemetry - it handles service info and OTel integration
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Logger:         slogLog,
})

// Access full slog API
log := t.Logger()
if slogLogger, ok := log.(*sloglogger.Logger); ok {
    slogLogger.Logger.Info("Processing request",
        slog.String("user_id", "123"),
        slog.String("action", "login"),
    )

    // Use slog groups
    slogLogger.Logger.Info("Request completed",
        slog.Group("request",
            slog.Int("duration_ms", 150),
            slog.Bool("success", true),
        ),
    )
}
```

## Log Levels

The library supports standard log levels with a common interface across all logger implementations:

```go
logger := t.Logger()

// Trace - Most verbose, for detailed debugging (more verbose than debug)
logger.Trace().Str("detail", "very detailed info").Msg("Trace message")

// Debug - Debug-level messages
logger.Debug().Int("count", 5).Msg("Debug message")

// Info - Informational messages
logger.Info().Str("status", "running").Msg("Info message")

// Warn - Warning messages
logger.Warn().Msg("Warning message")

// Error - Error messages
logger.Error().Err(err).Msg("Error message")

// Fatal - Fatal messages (calls os.Exit(1))
logger.Fatal().Msg("Fatal error")

// Panic - Panic messages (calls panic())
logger.Panic().Msg("Panic message")
```

### Log Level Support by Backend

| Level | Zerolog | Logrus | Zap | Slog | Notes |
|-------|---------|--------|-----|------|-------|
| Trace | ✅ Native | ✅ Native | ⚠️ Custom | ⚠️ Custom | Zap/Slog use custom levels |
| Debug | ✅ | ✅ | ✅ | ✅ | |
| Info | ✅ | ✅ | ✅ | ✅ | |
| Warn | ✅ | ✅ | ✅ | ✅ | |
| Error | ✅ | ✅ | ✅ | ✅ | |
| Fatal | ✅ | ✅ | ✅ | ⚠️ Maps to Error | Slog doesn't have Fatal |
| Panic | ✅ | ✅ | ✅ | ⚠️ Maps to Error | Slog doesn't have Panic |

**Notes:**
- **Trace**: Zerolog and Logrus have native trace levels. Zap uses `DebugLevel - 1`, Slog uses `LevelDebug - 4`
- **Fatal/Panic**: Slog doesn't have fatal/panic levels, so they map to Error with additional behavior (os.Exit/panic)
- All levels work through the common interface regardless of native support

### Setting Log Level

```go
import "github.com/ekristen/go-telemetry/logger"

// Set the minimum log level
logger.SetLevel(logger.TraceLevel)  // Show all logs including trace
logger.SetLevel(logger.DebugLevel)  // Show debug and above
logger.SetLevel(logger.InfoLevel)   // Show info and above (typical production)
logger.SetLevel(logger.WarnLevel)   // Show only warnings and errors
logger.SetLevel(logger.ErrorLevel)  // Show only errors
logger.SetLevel(logger.Disabled)    // Disable all logging

// Get current level
currentLevel := logger.Level()
```

## Tracing

### Basic Tracing

```go
ctx, span := t.StartSpan(ctx, "operation-name")
defer span.End()

// Your operation here
```

### Tracing with Logger

```go
ctx, span, logger := t.StartSpanWithLogger(ctx, "operation-name")
defer span.End()

// Logger has the span context attached
logger.Info().Msg("Processing within span")
```

## Architecture

```
telemetry/
├── config.go           # Configuration management
├── telemetry.go        # Main telemetry struct
├── providers.go        # OTel provider initialization
├── interface.go        # ITelemetry interface
├── logger/
│   ├── interface.go    # Common logger interface
│   ├── zerolog/
│   │   ├── zerolog.go  # Zerolog implementation
│   │   ├── otel_hook.go # OTel integration
│   │   └── console.go  # Console writer utilities
│   ├── logrus/
│   │   ├── logrus.go   # Logrus implementation
│   │   └── otel_hook.go # OTel integration
│   └── zap/
│       ├── zap.go      # Zap implementation
│       └── otel_core.go # OTel integration
└── examples/
    ├── basic/          # Basic usage without OTel
    ├── with-otel/      # Usage with OTel enabled
    ├── full-zerolog-api/ # Advanced zerolog features
    ├── logrus-basic/   # Logrus usage example
    └── zap-basic/      # Zap usage example
```

## Design Philosophy

1. **OTel is Optional**: The library works perfectly without OTel. Enable it only when you need distributed tracing and metrics.

2. **Full Logger Control**: You're not limited to a subset of logging features. Access the complete logger API.

3. **Zero Abstraction Overhead**: When OTel is disabled, there's no performance penalty.

4. **Swappable Backends**: Support for multiple logging frameworks (zerolog, logrus, and more).

## Local OTel Bridges

This library includes local implementations of OTel integrations for each logger backend:
- **Zerolog**: Custom OTel hook for zerolog integration
- **Logrus**: Custom OTel hook for logrus integration
- **Zap**: Custom OTel core for zap integration
- Allows you to customize integration behavior
- Keeps dependencies under control
- Ensures compatibility with your specific use case

## Examples

See the [examples](./examples) directory for complete working examples:

- [`basic`](./examples/basic) - Basic usage with zerolog, OTel disabled
- [`with-otel`](./examples/with-otel) - Full OTel integration with tracing
- [`traces-nested`](./examples/traces-nested) - Nested spans with attributes and events
- [`metrics`](./examples/metrics) - Counter, histogram, gauge examples
- [`full-zerolog-api`](./examples/full-zerolog-api) - Advanced zerolog features
- [`zerolog-basic`](./examples/zerolog-basic) - Basic zerolog usage
- [`logrus-basic`](./examples/logrus-basic) - Basic usage with logrus
- [`logrus-byo`](./examples/logrus-byo) - Bring your own logrus logger
- [`zap-basic`](./examples/zap-basic) - Basic usage with zap (simplified pattern)
- [`slog-basic`](./examples/slog-basic) - Basic usage with slog (simplified pattern)

## Known Issues

### Zerolog Attributes Not Passed to OTel

There is a bug in the zerolog hook handler that prevents log attributes (fields) from being passed to OpenTelemetry. This means that while logs are exported to OTel, any structured fields you add (like `.Str("key", "value")`) are not included in the OTel log records.

**Status**: An open PR exists to fix this issue: https://github.com/rs/zerolog/pull/682

**Workaround**: Until the fix is merged and released:
- Use a different logger backend (logrus, zap, or slog) if you need OTel log attributes
- Or wait for the zerolog fix to be merged and update your zerolog dependency

**What works**: Log messages and log levels are still correctly exported to OTel, only the additional attributes are missing.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Erik Kristensen
