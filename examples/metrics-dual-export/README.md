# Dual Export Metrics Example (Prometheus + OTLP)

This example demonstrates how to export metrics to **both** Prometheus and OTLP simultaneously using a single telemetry configuration.

## Why Dual Export?

Exporting to multiple destinations is useful for:

- **Migration scenarios**: Gradually moving from Prometheus to OTLP (or vice versa)
- **Hybrid monitoring**: Different teams using different observability platforms
- **Compatibility**: Supporting both pull-based (Prometheus) and push-based (OTLP) monitoring
- **A/B testing**: Comparing different monitoring solutions side-by-side
- **Redundancy**: Having backup metrics collection if one system fails

## What it does

- Creates a single telemetry instance with **both Prometheus and OTLP exporters**
- Exposes metrics via HTTP at `http://localhost:9090/metrics` for Prometheus
- Pushes the same metrics to an OTLP collector (if configured)
- Simulates HTTP request workload with counter, histogram, and gauge metrics
- Shows that all metrics are available in both formats simultaneously

## Running the example

### Prerequisites

For full dual export functionality, you'll need:
1. The application (this example)
2. (Optional) An OTLP collector endpoint

### Basic run (Prometheus only)

Without an OTLP endpoint, only Prometheus export will work:

```bash
cd examples/metrics-dual-export
go run main.go
```

The application will:
- Export metrics to Prometheus HTTP endpoint (works immediately)
- Log a warning that OTLP is disabled
- Continue running and collecting metrics

View Prometheus metrics:
```bash
curl http://localhost:9090/metrics
```

### Full dual export (Prometheus + OTLP)

To enable both exporters, set the OTLP endpoint:

```bash
# If you have an OTLP collector running locally
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

cd examples/metrics-dual-export
go run main.go
```

Now metrics are being exported to:
1. ✅ **Prometheus HTTP**: `http://localhost:9090/metrics`
2. ✅ **OTLP Collector**: Configured endpoint (e.g., Jaeger, Grafana, etc.)

## Quick OTLP Collector Setup

If you want to test full dual export, here's a quick way to start an OTLP collector:

### Using Docker with Jaeger (all-in-one)

```bash
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Then run the example:
```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
go run main.go
```

View metrics in:
- **Prometheus format**: `http://localhost:9090/metrics`
- **Jaeger UI**: `http://localhost:16686`

## Configuration

The key configuration for dual export:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:     "my-service",
    ServiceVersion:  "1.0.0",
    MetricsExporter: "prometheus,otlp", // Comma-separated list!
    PrometheusPort:  9090,
    PrometheusPath:  "/metrics",
})
```

Or via environment variables:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317  # Enables OTLP
export OTEL_METRICS_EXPORTER=prometheus,otlp              # Both exporters
export PROMETHEUS_PORT=9090                                # Prometheus port
export PROMETHEUS_PATH=/metrics                            # Prometheus path
```

## Verification

### Verify Prometheus Export

```bash
curl -s http://localhost:9090/metrics | grep http_requests_total
```

You should see output like:
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{http_method="GET",http_route="/api/users"...} 42
```

### Verify OTLP Export

Check your OTLP collector/backend:
- **Jaeger**: Visit `http://localhost:16686`
- **Grafana**: Check your Grafana instance
- **Other**: Check your specific OTLP backend

## Metric Types Demonstrated

All metric types work with dual export:

1. **Counter** (`http_requests_total`): Monotonically increasing request count
2. **Histogram** (`http_request_duration_milliseconds`): Request duration distribution
3. **UpDownCounter** (`http_connections_active`): Active connection count (can go up/down)
4. **Gauge** (`system_memory_usage_bytes`): Current memory usage snapshot

## Use Cases

### Migration Scenario

**Before**: Using Prometheus
```go
MetricsExporter: "prometheus"
```

**During Migration**: Dual export while teams transition
```go
MetricsExporter: "prometheus,otlp"
```

**After**: Using OTLP only
```go
MetricsExporter: "otlp"
```

### Hybrid Organization

Different teams can use their preferred tools:
- **Ops team**: Uses Prometheus (familiar tooling)
- **Dev team**: Uses OTLP with modern observability platform
- **Same metrics**: No duplicate instrumentation needed

## Performance Considerations

Dual export has minimal overhead:
- Metrics are collected **once** by OpenTelemetry SDK
- Two **readers** process the same data
- Network overhead: 2x (one for each export)
- CPU overhead: ~5-10% additional processing

For high-volume applications:
- Consider using `BatchExport: true` for OTLP
- Prometheus pull model has no push overhead
- Monitor your collector capacity

## Troubleshooting

### OTLP export not working

Check:
1. `OTEL_EXPORTER_OTLP_ENDPOINT` is set correctly
2. OTLP collector is running and accessible
3. Network connectivity (try `telnet localhost 4317`)
4. Check application logs for connection errors

### Prometheus endpoint empty

Check:
1. Application is running
2. Port 9090 is not blocked
3. Accessing correct endpoint: `http://localhost:9090/metrics`
4. Give it a few seconds for metrics to populate

### Metrics missing in one exporter

- Both exporters see the **same** metrics from OpenTelemetry
- If one is missing metrics, it's likely a configuration issue with that exporter
- Check logs for exporter-specific errors

## See Also

- [metrics](../metrics) - OTLP-only metrics example
- [metrics-prometheus](../metrics-prometheus) - Prometheus-only with built-in server
- [metrics-prometheus-custom-server](../metrics-prometheus-custom-server) - Custom HTTP server integration
