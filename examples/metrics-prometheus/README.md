# Prometheus Metrics Example

This example demonstrates how to use the telemetry library with Prometheus metrics exporter.

## What it does

- Creates a telemetry instance configured with Prometheus exporter
- Starts an HTTP server on port 9090 exposing metrics at `/metrics`
- Simulates HTTP request workload with various metrics:
  - **Counter**: `http_requests_total` - Total number of HTTP requests
  - **UpDownCounter**: `http_connections_active` - Active HTTP connections
  - **Histogram**: `http_request_duration_milliseconds` - Request duration distribution
  - **Gauge**: `system_memory_usage_bytes` - Current memory usage

## Running the example

```bash
cd examples/metrics-prometheus
go run main.go
```

The application will:
1. Start the Prometheus metrics server on `http://localhost:9090/metrics`
2. Begin simulating HTTP request workload
3. Update metrics continuously until you press Ctrl+C

## Viewing metrics

While the application is running, you can view the Prometheus metrics:

```bash
curl http://localhost:9090/metrics
```

You should see metrics in Prometheus exposition format:

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{http_method="GET",http_route="/api/users",http_status_code="200"} 42

# HELP http_request_duration_milliseconds HTTP request duration
# TYPE http_request_duration_milliseconds histogram
http_request_duration_milliseconds_bucket{http_method="POST",http_route="/api/orders",http_status_code="201",le="0"} 0
http_request_duration_milliseconds_bucket{http_method="POST",http_route="/api/orders",http_status_code="201",le="5"} 0
...
```

## Integrating with Prometheus

To scrape these metrics with Prometheus, add this job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'telemetry-example'
    static_configs:
      - targets: ['localhost:9090']
```

## Configuration

The example uses these configuration options:

```go
&telemetry.Options{
    ServiceName:     "metrics-prometheus-example",
    ServiceVersion:  "1.0.0",
    MetricsExporter: "prometheus",  // Use Prometheus exporter
    PrometheusPort:  9090,           // HTTP server port
    PrometheusPath:  "/metrics",     // Metrics endpoint path
}
```

You can also configure via environment variables:

```bash
export OTEL_SERVICE_NAME=my-service
export OTEL_METRICS_EXPORTER=prometheus
export PROMETHEUS_PORT=9090
export PROMETHEUS_PATH=/metrics
```

## Metric Types

### Counter (http_requests_total)
Monotonically increasing counter for total requests. Labeled with:
- `http_method`: HTTP method (GET, POST, etc.)
- `http_route`: API endpoint
- `http_status_code`: Response status code

### UpDownCounter (http_connections_active)
Can increase and decrease, tracking active connections. Labeled with:
- `protocol`: Connection protocol (http)

### Histogram (http_request_duration_milliseconds)
Distribution of request durations with buckets. Labeled with:
- `http_method`: HTTP method
- `http_route`: API endpoint
- `http_status_code`: Response status code

### Gauge (system_memory_usage_bytes)
Current value observed at scrape time. Labeled with:
- `type`: Memory type (heap)

## Comparing with OTLP

Unlike OTLP which pushes metrics to a collector, Prometheus uses a pull-based model:

**OTLP (Push)**:
```go
MetricsExporter: "otlp"  // Pushes to OTEL_EXPORTER_OTLP_ENDPOINT
```

**Prometheus (Pull)**:
```go
MetricsExporter: "prometheus"  // Prometheus scrapes /metrics endpoint
```

Both exporters support the same OpenTelemetry metric instruments!
