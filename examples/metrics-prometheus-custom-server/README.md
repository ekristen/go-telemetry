# Prometheus Metrics with Custom HTTP Server Example

This example demonstrates how to use the telemetry library with Prometheus metrics exporter while providing your own HTTP server instead of using the built-in one.

## Why Use a Custom HTTP Server?

You might want to use your own HTTP server when:

- You already have an existing HTTP server in your application
- You need custom middleware or authentication on your metrics endpoint
- You want to serve metrics alongside other application routes
- You need more control over server configuration (TLS, timeouts, etc.)

## What it does

- Creates a telemetry instance with Prometheus exporter and **disables the built-in HTTP server**
- Retrieves the Prometheus handler using `t.PrometheusHandler()`
- Creates a custom HTTP server with multiple routes:
  - `GET /` - Welcome page
  - `GET /health` - Health check endpoint
  - `GET /metrics` - Prometheus metrics endpoint
- Simulates HTTP request workload with metrics collection

## Running the example

```bash
cd examples/metrics-prometheus-custom-server
go run main.go
```

The application will:
1. Start a custom HTTP server on `http://localhost:8080`
2. Expose Prometheus metrics at `http://localhost:8080/metrics`
3. Provide additional routes for your application
4. Update metrics continuously until you press Ctrl+C

## Accessing the endpoints

While the application is running:

### Home page
```bash
curl http://localhost:8080/
```

### Health check
```bash
curl http://localhost:8080/health
```

### Prometheus metrics
```bash
curl http://localhost:8080/metrics
```

You should see metrics in Prometheus exposition format with your custom server serving them!

## Configuration

The key configuration difference from the basic Prometheus example:

```go
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:                    "my-service",
    ServiceVersion:                 "1.0.0",
    MetricsExporter:                "prometheus",
    DisableBuiltinPrometheusServer: true, // Important!
})

// Get the handler to register with your server
promHandler := t.PrometheusHandler()

// Add to your custom HTTP server
mux := http.NewServeMux()
mux.Handle("/metrics", promHandler)
```

## Integrating into Your Application

To integrate this pattern into your existing application:

```go
// 1. Create telemetry with built-in server disabled
t, err := telemetry.New(ctx, &telemetry.Options{
    ServiceName:                    "my-app",
    MetricsExporter:                "prometheus",
    DisableBuiltinPrometheusServer: true,
})

// 2. Get the Prometheus handler
promHandler := t.PrometheusHandler()

// 3. Register with your existing HTTP server
yourMux.Handle("/metrics", promHandler)

// Or with popular frameworks:

// Gin
r := gin.Default()
r.GET("/metrics", gin.WrapH(promHandler))

// Echo
e := echo.New()
e.GET("/metrics", echo.WrapHandler(promHandler))

// Chi
r := chi.NewRouter()
r.Handle("/metrics", promHandler)

// Gorilla Mux
r := mux.NewRouter()
r.Handle("/metrics", promHandler)
```

## Comparison with Built-in Server

| Feature | Built-in Server | Custom Server |
|---------|----------------|---------------|
| Setup | Automatic | Manual |
| Configuration | `PrometheusPort`, `PrometheusPath` | Full control |
| Integration | Separate server | Same server as app |
| Middleware | Not supported | Full support |
| Custom routes | Not possible | Full support |
| TLS/Auth | Basic | Full control |

## Security Considerations

When exposing metrics on your custom server:

1. **Authentication**: Add middleware to protect the metrics endpoint
   ```go
   mux.Handle("/metrics", authMiddleware(promHandler))
   ```

2. **Network isolation**: Consider binding to localhost or internal network only
   ```go
   server := &http.Server{
       Addr: "127.0.0.1:8080", // Localhost only
   }
   ```

3. **Rate limiting**: Prevent abuse of the metrics endpoint

4. **TLS**: Use HTTPS in production environments

## See Also

- [metrics-prometheus](../metrics-prometheus) - Example with built-in HTTP server
- [metrics](../metrics) - OTLP metrics example
