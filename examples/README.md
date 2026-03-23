# OpenTelemetry Examples

This folder contains minimal OpenTelemetry examples that demonstrate how to instrument HTTP services and export telemetry signals.

## What is OpenTelemetry?

OpenTelemetry (often abbreviated as OTel) is an open standard and SDK ecosystem for observability. It provides a vendor-neutral way to instrument services and export telemetry to different backends without changing your application logic for each platform.

In practice, OpenTelemetry helps you answer questions such as:

- Which service or endpoint is slow?
- Where did a request fail across multiple components?
- Is error rate or latency increasing over time?

It helps applications produce:

- **Traces**: end-to-end request flow and latency across components.
- **Metrics**: numeric time series (for example request counters and error rates).
- **Logs**: structured event records from application execution.

In these examples, services export telemetry through OTLP (OpenTelemetry Protocol) to an OpenTelemetry Collector endpoint, which redirects it to the respective backend. The collector maintained by CMS Monitoring supports metrics through a VictoriaMetrics backend, logs through Opensearch and traces through Grafana Tempo.

## Documentation links

- OpenTelemetry overview: [https://opentelemetry.io/docs/what-is-opentelemetry/](https://opentelemetry.io/docs/what-is-opentelemetry/)
- OpenTelemetry docs home: [https://opentelemetry.io/docs/](https://opentelemetry.io/docs/)
- OpenTelemetry specification: [https://opentelemetry.io/docs/specs/](https://opentelemetry.io/docs/specs/)
- OTLP specification: [https://opentelemetry.io/docs/specs/otlp/](https://opentelemetry.io/docs/specs/otlp/)
- OpenTelemetry Collector: [https://opentelemetry.io/docs/collector/](https://opentelemetry.io/docs/collector/)
- Python instrumentation docs: [https://opentelemetry.io/docs/languages/python/](https://opentelemetry.io/docs/languages/python/)
- Go instrumentation docs: [https://opentelemetry.io/docs/languages/go/](https://opentelemetry.io/docs/languages/go/)

## Layout of the provided examples

- `python_app_otel_example.py`
  - General Python worker-style process (non-web example).
  - Emits traces, metrics, and logs in a loop.

- `flask_app_otel_example.py`
  - Python HTTP server using Flask (`GET /`).
  - Creates one span per request and increments a request counter.

- `cherrypy_app_otel_example.py`
  - Python HTTP server using CherryPy (`GET /`).
  - Creates one span per request and increments a request counter.

- `go_app_otel_example.go`
  - Go HTTP server (`GET /` on `127.0.0.1:8081`).
  - Creates one span per request, increments a request counter, and emits logs.

## Common configuration

All examples support these environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP gRPC collector endpoint (default is set in each example).
- `OTEL_SERVICE_NAME`: logical service name shown in telemetry backends.
- `OPENTELEMETRY_USERNAME` / `OPENTELEMETRY_PASSWORD`: optional basic auth credentials used to build OTLP headers.
- `OTEL_METRIC_EXPORT_INTERVAL`: export interval in milliseconds (Python examples).

## Running the instrumented HTTP server examples

From this directory:

- Flask:
  - `python3 flask_app_otel_example.py`
  - open `http://127.0.0.1:5000/`
- CherryPy:
  - `python3 cherrypy_app_otel_example.py`
  - open `http://127.0.0.1:8080/`
- Go:
  - `go mod init examples/go-otel-toy`
  - `go mod tidy`
  - `go run go_app_otel_example.go`
  - open `http://127.0.0.1:8081/`

## Client usage

Any HTTP client can generate traffic for these instrumented servers and trigger telemetry generation. For example:

- `curl http://127.0.0.1:5000/` (Flask)
- `curl http://127.0.0.1:8080/` (CherryPy)
- `curl http://127.0.0.1:8081/` (Go)

The resulting metrics and traces can then be consumed from your observability backend (for example via OpenTelemetry Collector pipelines into Prometheus/Grafana, OpenSearch, Jaeger, or another OTLP-compatible system).
