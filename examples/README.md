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

- `python_multiprocessing_otel_example.py`
  - Fork-based `ProcessPoolExecutor` workload (same pattern as spider query cronjob).
  - Shows how to safely reinitialize OTLP exporters after `fork`, avoid log duplication,
    and recreate metric instruments per worker PID.
  - Requires `GRPC_ENABLE_FORK_SUPPORT=true` and `GRPC_POLL_STRATEGY=poll`.

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

Multiprocessing example also uses:

- `GRPC_ENABLE_FORK_SUPPORT` / `GRPC_POLL_STRATEGY` (defaulted to `true` / `poll` in the script).
- `EXAMPLE_WORKER_PROCESSES`: how many forked processes the demo `ProcessPoolExecutor` starts (default `2`).
- `EXAMPLE_WORK_ITEMS`: how many toy work items the demo submits to that pool (default `4`).

## Running the instrumented examples

From this directory:

- Plain Python worker:
  - `python3 python_app_otel_example.py`
- Multiprocessing (fork-safe OTel):
  - `python3 python_multiprocessing_otel_example.py`
  - optional: `EXAMPLE_WORKER_PROCESSES=2 EXAMPLE_WORK_ITEMS=4`
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

### Multiprocessing / fork note

If OpenTelemetry is initialized in the parent and you later create child processes with
`fork` (for example via `ProcessPoolExecutor`), inherited OTLP gRPC exporters are unsafe.
The spider query cronjob pattern is:

1. Export `GRPC_ENABLE_FORK_SUPPORT=true` and `GRPC_POLL_STRATEGY=poll`.
2. Use a pool `initializer` that shuts down inherited providers, resets OpenTelemetry's
   set-once flags, and installs a fresh OTLP stack in each worker.
3. Rebuild instruments per PID (see `lazy_instruments` in the multiprocessing example).
4. Remove inherited OTLP `LoggingHandler`s before attaching a new one to avoid duplicated logs.

See `python_multiprocessing_otel_example.py` and
`docker/spider-query-cronjob/src/otel_setup.py` (`reinit_otel_in_worker`).

## Client usage

Any HTTP client can generate traffic for these instrumented servers and trigger telemetry generation. For example:

- `curl http://127.0.0.1:5000/` (Flask)
- `curl http://127.0.0.1:8080/` (CherryPy)
- `curl http://127.0.0.1:8081/` (Go)

The resulting metrics and traces can then be consumed from your observability backend (for example via OpenTelemetry Collector pipelines into Prometheus/Grafana, OpenSearch, Jaeger, or another OTLP-compatible system).
