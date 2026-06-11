# CMS Monitoring Python base image

Dockerfile for generating the base image for most non-Spark Python services in CMS Monitoring infrastructure.

## Characteristics

### Base Image

- Python 3.9 slim base image from Docker Hub via CERN pull-through cache
- Working directory: `/data`
- Environment variables:
  - `PYTHONPATH` includes `/data`
  - `LC_ALL=C.UTF-8` and `LANG=C.UTF-8` for proper Unicode support

### Pre-installed Packages

- `click` - Command line interface creation toolkit
- `opentelemetry-exporter-otlp~=1.39.0` - OTLP export for metrics, traces, and logs
- `pandas` - Data manipulation and analysis library
- `rucio-clients` - CMS data management system client
- `schema` - Data validation library
- `stomp.py==7.0.0` - STOMP protocol implementation for message queuing

## Usage

This image functions as a base image for CMS Monitoring services that require Python but are not Spark-related. It is designed to be extended rather than run directly, and should be used as a base layer in Docker images that contain the specific scripts and code for individual services.

## OpenTelemetry in child images

Child images can send metrics, traces, and logs to the in-cluster OpenTelemetry Collector via `helpers/otel_setup.py`. Instrumentation is opt-in: set `OTEL_ENABLED=true` and `OTEL_SERVICE_NAME` in the workload environment (and `OTEL_EXPORTER_OTLP_ENDPOINT` if not using the default `opentelemetry-collector.opentelemetry.svc.cluster.local:4317`).

Import `otel_setup` at startup (before `logging.getLogger()`). It configures stdout logging for `kubectl logs` and, when `OTEL_ENABLED=true`, also exports logs to the collector. Jobs should use `logging` (not `print`):

```python
import logging

import helpers.otel_setup  # noqa: F401 — configures stdout + optional OTLP
from helpers.otel_setup import global_meter, shutdown_opentelemetry, trace_span

logger = logging.getLogger(__name__)

@trace_span("my_job_step")
def run_job():
    logger.info("processing started")
    global_meter.create_counter("jobs_processed").add(1)

if __name__ == "__main__":
    try:
        run_job()
    finally:
        shutdown_opentelemetry()
```

See `helpers/otel_setup.py` for the full list of supported environment variables.

Stomp loggers (`stomp.py`, `StompAMQ`, `StompyListener`) are set to `WARNING` during OpenTelemetry setup so `INFO` and below are not exported.
