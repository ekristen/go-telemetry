# go-telemetry

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/ekristen/go-telemetry/v2.svg)](https://pkg.go.dev/github.com/ekristen/go-telemetry/v2)

**OpenTelemetry-first** Go telemetry library focused on non-invasive instrumentation. Create your logger externally, attach OTel hooks/cores/handlers, maintain full control and accurate caller reporting.

**Philosophy**: *"Create your logger, attach our hooks"* - not "use our wrapper to create a logger"

**Recommended**: All loggers support accurate caller reporting with the **external hook pattern**. See [examples](#examples) below.

## AI

Some code and documentation in this project were created or refined with the assistance of AI tools. All contributions are reviewed and verified by human maintainers.

## Features

- **OpenTelemetry First**: Logs, traces, and metrics instrumentation at the forefront
- **Non-Invasive Integration**: Attach OTel hooks/cores/handlers to your existing loggers
- **Multiple Logger Backends**: Zap, Zerolog, Logrus, Slog
- **No Wrappers**: Uses hooks/cores/handlers for OTel integration, not wrapper layers
- **Optional OpenTelemetry**: Toggle OTel on/off via environment variables
- **Full Logger Access**: Use the complete API of your chosen logger
- **Multiple Metric Exporters**: Support for OTLP (push) and Prometheus (pull) metrics
- **Zero Overhead**: No OTel overhead when disabled
- **Flexible Configuration**: Environment variables and functional options

## Installation

```bash
go get github.com/ekristen/go-telemetry/v2
```

## Quick Start

### Recommended: External Hook Pattern

The **external hook pattern** is the recommended approach for all loggers. You create and configure your logger, then attach OTel integration externally.

**Benefits:**
- ✅ **Accurate caller reporting** for all loggers
- ✅ **Full control** over logger configuration
- ✅ **Non-invasive** - OTel doesn't modify your logger
- ✅ **Use native API** - idiomatic logger usage

**Example with Logrus:**

```go
package main

import (
    "context"

    "github.com/ekristen/go-telemetry/v2"
    logrushook "github.com/ekristen/go-telemetry/v2/hooks/logrus"
    "github.com/sirupsen/logrus"
)

func main() {
    ctx := context.Background()

    // Step 1: Create YOUR logger (full control)
    log := logrus.New()
    log.SetReportCaller(true)  // Caller info will be accurate!
    log.SetFormatter(&logrus.JSONFormatter{})

    // Step 2: Initialize OTel
    t, _ := telemetry.New(ctx, &telemetry.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
    })
    defer t.Shutdown(ctx)

    // Step 3: Attach OTel hook (non-invasive!)
    if t.LoggerProvider() != nil {
        hook := logrushook.New(
            t.ServiceName(),
            t.ServiceVersion(),
            t.LoggerProvider(),
        )
        if hook != nil {
            log.AddHook(hook)
        }
    }

    // Step 4: Use native API - logs go to console AND OTel!
    log.WithFields(logrus.Fields{
        "status": "running",
    }).Info("Service started")  // ✅ Accurate caller: yourfile.go:42
}
```

See [Hook Pattern](#hook-pattern-recommended) section below for all loggers.

## Hook Pattern (Recommended)

For **maximum control and accurate caller reporting**, use the external hook pattern where you create your logger externally and attach OTel hooks afterwards.

### Logrus External Hook Example

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/ekristen/go-telemetry/v2"
    logrushook "github.com/ekristen/go-telemetry/v2/hooks/logrus"
)

// Step 1: Create and configure YOUR logrus logger
log := logrus.New()
log.SetReportCaller(true)  // Caller info will be accurate!
log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})
log.SetLevel(logrus.DebugLevel)

// Step 2: Initialize OpenTelemetry
t, _ := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})
defer t.Shutdown(ctx)

// Step 3: Attach OTel hook to YOUR logger (non-invasive!)
if t.LoggerProvider() != nil {
    otelHook := logrushook.New(
        t.ServiceName(),
        t.ServiceVersion(),
        t.LoggerProvider(),
    )
    if otelHook != nil {
        log.AddHook(otelHook)
    }
}

// Step 4: Use YOUR logger directly - logs go to console AND OTel!
log.WithFields(logrus.Fields{
    "user_id": "123",
    "action":  "login",
}).Info("User logged in")  // ✅ Accurate caller: yourfile.go:42
```

**Why this pattern?**

✅ **Accurate caller reporting** - `SetReportCaller(true)` works correctly
✅ **Full control** - You configure your logger exactly as needed
✅ **Non-invasive** - OTel hook added externally, not during logger creation
✅ **Separation of concerns** - Logging configuration separate from observability
✅ **No wrapper interface** - Use logrus/zap/zerolog/slog directly with full API

See [examples/logrus](./examples/logrus) for a complete working example.

### Similar Patterns for Other Loggers

**Zap External Core:**
```go
import (
    "github.com/ekristen/go-telemetry/v2"
    zaphook "github.com/ekristen/go-telemetry/v2/hooks/zap"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// Create your zap logger
encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

// Initialize telemetry
t, _ := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})

// Add OTel core
otelCore := zaphook.New(t.ServiceName(), t.ServiceVersion(), t.LoggerProvider())
core := zapcore.NewTee(consoleCore, otelCore)
logger := zap.New(core, zap.AddCaller())
```

**Zerolog External Hook:**
```go
import (
    "github.com/ekristen/go-telemetry/v2"
    zerologhook "github.com/ekristen/go-telemetry/v2/hooks/zerolog"
    "github.com/rs/zerolog"
)

// Create your zerolog logger
log := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

// Initialize telemetry
t, _ := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})

// Add OTel hook
hook := zerologhook.New(t.ServiceName(), t.ServiceVersion(), t.LoggerProvider())
log = log.Hook(hook)
```

**Slog External Handler:**
```go
import (
    "github.com/ekristen/go-telemetry/v2"
    sloghook "github.com/ekristen/go-telemetry/v2/hooks/slog"
    "log/slog"
)

// Create your slog handler
baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    AddSource: true,  // Enable caller info
    Level:     slog.LevelDebug,
})

// Initialize telemetry
t, _ := telemetry.New(ctx, &telemetry.Options{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
})

// Wrap with OTel handler
otelHandler := sloghook.New(baseHandler, t.ServiceName(), t.ServiceVersion(), t.LoggerProvider())
log := slog.New(otelHandler)

// Use native API for accurate caller
log.Info("message", slog.String("key", "value"))  // ✅ Accurate!
```

## Logger Comparison

| Feature | Zap | Zerolog | Logrus | Slog |
|---------|-----|---------|--------|------|
| **Accurate Caller (External Hook)** | ✅ Yes | ✅ Yes | ✅ Yes* | ✅ Yes** |
| **Performance** | Excellent | Excellent | Good | Good |
| **OTel Integration** | Core | Hook | Hook | Handler |
| **Allocation** | Low | Zero | Medium | Low |
| **Recommendation** | **Best choice** | Zero-alloc needs | **Use external hook** | **Use external handler** |

\* **Logrus**: Accurate caller with external hook pattern (SetReportCaller before AddHook)
\*\* **Slog**: Accurate caller with external handler pattern (must use native API)

**See [CALLER_REPORTING.md](CALLER_REPORTING.md) for detailed caller behavior explanation.**

### Why Caller Reporting Matters

Accurate caller information shows the exact file and line in **your code** where logs originated, not library internals. This is crucial for debugging.

**Using external hook/handler pattern (recommended):**
- **All loggers**: ✅ Report `yourfile.go:123` (your actual code)
- **Logrus**: Must use `SetReportCaller(true)` before `AddHook()`
- **Slog**: Must use native API (`log.Info("msg", slog.String(...))`)

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

    // BatchExport controls whether telemetry is exported in batches or immediately
    // When true: Uses batch processors for better performance (higher latency)
    // When false (default): Uses simple/synchronous processors for immediate export
    // Recommended: false for development/debugging, true for high-volume production
    BatchExport bool

    // MetricsExporter specifies which metrics exporter(s) to use: "otlp", "prometheus", or "none"
    // Supports multiple exporters via comma-separated list: "prometheus,otlp"
    // When empty, defaults to "otlp" if OTel is enabled
    // Can be overridden by OTEL_METRICS_EXPORTER environment variable
    MetricsExporter string

    // PrometheusPort is the HTTP port for the Prometheus metrics endpoint (default: 9090)
    // Only used when MetricsExporter is "prometheus"
    // Can be overridden by PROMETHEUS_PORT environment variable
    PrometheusPort int

    // PrometheusPath is the HTTP path for the Prometheus metrics endpoint (default: "/metrics")
    // Only used when MetricsExporter is "prometheus"
    // Can be overridden by PROMETHEUS_PATH environment variable
    PrometheusPath string

    // PrometheusServer enables the built-in Prometheus HTTP server
    // When false (default), use PrometheusHandler() to get the handler for your own server
    // When true, starts an HTTP server on PrometheusPort serving metrics at PrometheusPath
    // Only used when MetricsExporter is "prometheus"
    PrometheusServer bool
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
- `OTEL_SERVICE_VERSION` - Service version (can also be set in Options)
- `OTEL_RESOURCE_ATTRIBUTES` - Additional resource attributes

**Exporter Configuration:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (e.g., `http://localhost:4317`)
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` - Traces-specific endpoint
- `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` - Metrics-specific endpoint
- `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` - Logs-specific endpoint

**Signal Control:**
- `OTEL_TRACES_EXPORTER` - Traces exporter (default: `otlp`, set to `none` to disable)
- `OTEL_METRICS_EXPORTER` - Metrics exporter (options: `otlp`, `prometheus`, `none`)
- `OTEL_LOGS_EXPORTER` - Logs exporter (default: `otlp`, set to `none` to disable)

**Prometheus-Specific:**
- `PROMETHEUS_PORT` - HTTP port for Prometheus metrics endpoint (default: `9090`)
- `PROMETHEUS_PATH` - HTTP path for Prometheus metrics endpoint (default: `/metrics`)

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

## Logger Integration

The library provides OTel integration for four popular Go logging libraries through hooks, cores, and handlers in the `hooks/` package:

### Integration Packages

- `github.com/ekristen/go-telemetry/v2/hooks/logrus` - Logrus hook integration
- `github.com/ekristen/go-telemetry/v2/hooks/zap` - Zap core integration
- `github.com/ekristen/go-telemetry/v2/hooks/zerolog` - Zerolog hook integration
- `github.com/ekristen/go-telemetry/v2/hooks/slog` - Slog handler integration

### Integration Pattern

All integrations follow the same pattern:

1. **Create your logger** with your preferred configuration
2. **Initialize telemetry** with service name and version
3. **Attach OTel integration** using the appropriate hook/core/handler
4. **Use your logger** natively - logs go to both your output and OTel

See the [examples](#examples) directory for complete working examples of each logger.

## Metrics

The library supports both OTLP (push-based) and Prometheus (pull-based) metrics exporters.

### OTLP Metrics (Push-Based)

Push metrics to an OpenTelemetry collector:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    MetricsExporter: "otlp", // Default when OTel is enabled
})
```

Or via environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_METRICS_EXPORTER=otlp  # This is the default
```

### Prometheus Metrics (Pull-Based)

Expose metrics via HTTP for Prometheus to scrape. **By default**, the Prometheus handler is created but you must integrate it into your own HTTP server:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    MetricsExporter: "prometheus",
})

// Get the handler and add to your HTTP server
handler := t.PrometheusHandler()
mux := http.NewServeMux()
mux.Handle("/metrics", handler)
```

Or via environment variables:

```bash
export OTEL_METRICS_EXPORTER=prometheus
```

#### Using the Built-in HTTP Server (Optional)

If you want the library to automatically start an HTTP server for you:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:      "my-service",
    ServiceVersion:   "1.0.0",
    MetricsExporter:  "prometheus",
    PrometheusServer: true,  // Enable built-in HTTP server
    PrometheusPort:   9090,
    PrometheusPath:   "/metrics",
})
// Metrics will be available at http://localhost:9090/metrics
```

Or via environment variables:

```bash
export OTEL_METRICS_EXPORTER=prometheus
export PROMETHEUS_PORT=9090
export PROMETHEUS_PATH=/metrics
```

See the [examples/metrics-prometheus](./examples/metrics-prometheus) example for a complete working example with the built-in server.

#### Integrating with Popular Frameworks

The default behavior (built-in server disabled) makes it easy to integrate with any web framework:

```go
// Get the handler (built-in server is OFF by default)
handler := t.PrometheusHandler()

// Gin:    r.GET("/metrics", gin.WrapH(handler))
// Echo:   e.GET("/metrics", echo.WrapHandler(handler))
// Chi:    r.Handle("/metrics", handler)
// Gorilla: r.Handle("/metrics", handler)
```

See the [examples/metrics-prometheus-custom-server](./examples/metrics-prometheus-custom-server) example for a complete working example.

### Using Metrics

Both exporters support the same OpenTelemetry metric instruments:

```go
mp := t.MeterProvider()
meter := mp.Meter("my-component")

// Counter - monotonically increasing
counter, _ := meter.Int64Counter("requests_total")
counter.Add(ctx, 1)

// Histogram - distribution of values
histogram, _ := meter.Float64Histogram("request_duration_ms")
histogram.Record(ctx, 123.45)

// UpDownCounter - can increase or decrease
upDownCounter, _ := meter.Int64UpDownCounter("active_connections")
upDownCounter.Add(ctx, 1)

// Gauge - current value via callback
gauge, _ := meter.Int64ObservableGauge("memory_usage_bytes",
    metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
        observer.Observe(getMemoryUsage())
        return nil
    }),
)
```

### Prometheus vs OTLP

| Feature | OTLP | Prometheus |
|---------|------|------------|
| Model | Push | Pull |
| Endpoint | Collector (gRPC) | HTTP `/metrics` |
| Configuration | `OTEL_EXPORTER_OTLP_ENDPOINT` | `PROMETHEUS_PORT`, `PROMETHEUS_PATH` |
| Best for | Cloud-native, distributed systems | Traditional monitoring, simple setups |
| Format | Protobuf (OTLP) | Prometheus exposition format |

See the [examples/metrics-prometheus](./examples/metrics-prometheus) example for a complete working example.

### Dual Export (Prometheus + OTLP)

You can export metrics to both Prometheus and OTLP simultaneously:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:      "my-service",
    ServiceVersion:   "1.0.0",
    MetricsExporter:  "prometheus,otlp", // Both exporters!
    PrometheusServer: true,  // Optional: enable built-in HTTP server
    PrometheusPort:   9090,
})
```

Or via environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_METRICS_EXPORTER=prometheus,otlp
export PROMETHEUS_PORT=9090
```

This allows you to:
- Expose metrics via HTTP for Prometheus scraping
- Push the same metrics to an OTLP collector
- Maintain compatibility with both monitoring systems
- No duplicate instrumentation code needed

**Use cases:**
- Migration from Prometheus to OTLP (or vice versa)
- Hybrid monitoring setups
- Different teams using different observability platforms
- A/B testing between monitoring solutions

See the [examples/metrics-dual-export](./examples/metrics-dual-export) example for a complete working example.

## Tracing

### Basic Tracing

```go
ctx, span := t.StartSpan(ctx, "operation-name")
defer span.End()

// Your operation here
```

### Tracing with Logger and Context

The logger hooks/handlers will automatically extract trace information from the context when logging:

```go
ctx, span := t.StartSpan(ctx, "operation-name")
defer span.End()

// Logger extracts trace context from the span in ctx
log.WithContext(ctx).Info("Processing within span")
```

## Architecture

```
telemetry/
├── config.go           # Configuration management and env var handling
├── telemetry.go        # Main telemetry struct and public API
├── providers.go        # OTel provider initialization
├── interface.go        # ITelemetry interface
├── hooks/
│   ├── logrus/
│   │   └── logrus.go   # Logrus hook for OTel integration
│   ├── zap/
│   │   └── zap.go      # Zap core for OTel integration
│   ├── zerolog/
│   │   └── zerolog.go  # Zerolog hook for OTel integration
│   └── slog/
│       └── slog.go     # Slog handler for OTel integration
└── examples/
    ├── logrus/         # Logrus external hook example
    ├── zap/            # Zap external core example
    ├── zerolog/        # Zerolog external hook example
    ├── slog/           # Slog external handler example
    ├── metrics/        # OTLP metrics example
    ├── metrics-prometheus/  # Prometheus with built-in server
    ├── metrics-prometheus-custom-server/  # Prometheus with custom server
    └── metrics-dual-export/  # Prometheus + OTLP dual export
```

## Design Philosophy

1. **External Logger Creation**: You create and configure your logger - we provide hooks/cores/handlers to attach

2. **Non-Invasive Integration**: OTel hooks don't modify logger behavior or wrap your code

3. **Native API Usage**: Use your logger's idiomatic API directly - no wrapper interface needed

4. **Accurate Caller Reporting**: All loggers support accurate caller info with external hook pattern

5. **OTel is Optional**: Works perfectly without OTel. Enable only when you need observability.

6. **Zero Abstraction Overhead**: When OTel is disabled, there's no performance penalty.

## Examples

All examples demonstrate the **external hook/core/handler pattern** with accurate caller reporting:

- **[`logrus`](./examples/logrus)** - ✅ Create logrus logger, attach OTel hook externally
- **[`zap`](./examples/zap)** - ✅ Create zap core, combine with OTel core using NewTee
- **[`zerolog`](./examples/zerolog)** - ✅ Create zerolog logger, attach OTel hook externally
- **[`slog`](./examples/slog)** - ✅ Create slog handler, wrap with OTel handler
- **[`metrics`](./examples/metrics)** - Counter, histogram, gauge examples with OTLP
- **[`metrics-prometheus`](./examples/metrics-prometheus)** - Prometheus metrics with built-in HTTP server
- **[`metrics-prometheus-custom-server`](./examples/metrics-prometheus-custom-server)** - Prometheus with your own HTTP server
- **[`metrics-dual-export`](./examples/metrics-dual-export)** - Export metrics to both Prometheus and OTLP

All examples show **accurate caller reporting** and **native API usage**.

## Key Features

### Accurate Caller Reporting for All Loggers

All four major Go loggers now support **accurate caller reporting** with the external hook/core/handler pattern:

```go
// All of these show accurate caller info (yourfile.go:42):

// Logrus (with SetReportCaller before AddHook)
log.WithField("key", "val").Info("message")

// Zap (with zap.AddCaller())
log.Info("message", zap.String("key", "val"))

// Zerolog (with Caller())
log.Info().Str("key", "val").Msg("message")

// Slog (with AddSource: true)
log.Info("message", slog.String("key", "val"))
```

### Non-Invasive OpenTelemetry Integration

Logs go to **both** your configured output AND OpenTelemetry:

```go
// Your logger outputs to console
// OTel hook/core/handler sends to collector
// Same log, two destinations!
```

No code changes needed - just attach the hook/core/handler once.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Erik Kristensen
