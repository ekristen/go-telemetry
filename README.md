# go-telemetry

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/ekristen/go-telemetry/v2.svg)](https://pkg.go.dev/github.com/ekristen/go-telemetry/v2)

**OpenTelemetry-first** Go telemetry library focused on non-invasive instrumentation.

**Philosophy**: *"Create your logger, attach our hooks"* - not "use our wrapper to create a logger"

## Features

- **Non-invasive**: Attach OTel hooks/cores/handlers to your existing loggers - no wrappers
- **Multiple loggers**: Zap, Zerolog, Logrus, Slog with accurate caller reporting
- **Optional OTel**: Toggle on/off via environment variables, zero overhead when disabled
- **Flexible metrics**: OTLP (push) and Prometheus (pull) exporters
- **Standard OTel**: Uses OpenTelemetry environment variables for configuration

## Installation

```bash
go get github.com/ekristen/go-telemetry/v2
```

## Quick Start

Create your logger, initialize telemetry, attach the OTel hook:

```go
package main

import (
    "context"
    "github.com/ekristen/go-telemetry/v2"
    logrushook "github.com/ekristen/go-telemetry/hooks/logrus/v2"
    "github.com/sirupsen/logrus"
)

func main() {
    ctx := context.Background()

    // Create your logger with caller reporting
    log := logrus.New()
    log.SetReportCaller(true)
    log.SetFormatter(&logrus.JSONFormatter{})

    // Initialize telemetry
    t, _ := telemetry.New(ctx, &telemetry.Options{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
    })
    defer t.Shutdown(ctx)

    // Attach OTel hook
    if t.LoggerProvider() != nil {
        if hook := logrushook.New(t.ServiceName(), t.ServiceVersion(), t.LoggerProvider()); hook != nil {
            log.AddHook(hook)
        }
    }

    // Use your logger - logs go to console AND OTel
    log.WithFields(logrus.Fields{"status": "running"}).Info("Service started")
}
```

## Logger Integration

All loggers follow the same pattern: create your logger, initialize telemetry, attach the OTel integration. See [examples/](./examples/) for complete working examples.

| Logger | Integration | Package |
|--------|-------------|---------|
| **Logrus** | Hook | `github.com/ekristen/go-telemetry/hooks/logrus/v2` |
| **Zap** | Core | `github.com/ekristen/go-telemetry/hooks/zap/v2` |
| **Zerolog** | Hook | `github.com/ekristen/go-telemetry/hooks/zerolog/v2` |
| **Slog** | Handler | `github.com/ekristen/go-telemetry/hooks/slog/v2` |

**Caller Reporting**: All loggers support accurate caller info when using the external hook/handler pattern. Enable caller reporting in your logger before attaching the OTel integration.

## Configuration

OpenTelemetry is **automatically enabled** when standard OTel environment variables are set:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_SERVICE_NAME=my-service

# Disable specific signals
export OTEL_TRACES_EXPORTER=none   # or OTEL_METRICS_EXPORTER=none, OTEL_LOGS_EXPORTER=none
export OTEL_SDK_DISABLED=true      # Disable entire SDK
```

### Options

Key options available in `telemetry.Options`:

- **ServiceName/ServiceVersion**: Service identification
- **BatchExport**: `false` (default, immediate) for dev/debug, `true` (batched) for high-volume production
- **MetricsExporter**: `"otlp"` (default), `"prometheus"`, `"prometheus,otlp"` (dual), or `"none"`
- **PrometheusPort/PrometheusPath**: Prometheus endpoint configuration (default: `9090`, `"/metrics"`)
- **PrometheusServer**: `true` to enable built-in HTTP server, `false` (default) to use `PrometheusHandler()` with your own server

Pass `nil` to use defaults: `telemetry.New(ctx, nil)`

## Metrics

Supports OTLP (push) and Prometheus (pull) metrics exporters.

**OTLP (default):**
```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

**Prometheus (use your HTTP server):**
```go
t, _ := telemetry.New(ctx, &telemetry.Options{MetricsExporter: "prometheus"})
handler := t.PrometheusHandler()  // Add to your mux/router
```

**Prometheus (built-in server):**
```go
t, _ := telemetry.New(ctx, &telemetry.Options{
    MetricsExporter:  "prometheus",
    PrometheusServer: true,
    PrometheusPort:   9090,
})
```

**Dual export:** `MetricsExporter: "prometheus,otlp"`

**Create and use metrics:**
```go
meter := t.MeterProvider().Meter("my-component")
counter, _ := meter.Int64Counter("requests_total")
counter.Add(ctx, 1)
```

## Tracing

```go
ctx, span := t.StartSpan(ctx, "operation-name")
defer span.End()

// Logger hooks automatically extract trace context from ctx
log.WithContext(ctx).Info("Processing within span")
```


## Examples

See [examples/](./examples/) directory:
- Logger integrations: `logrus`, `zap`, `zerolog`, `slog`
- Metrics: `metrics` (OTLP), `metrics-prometheus`, `metrics-prometheus-custom-server`, `metrics-dual-export`


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Erik Kristensen
