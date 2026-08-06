#!/usr/bin/env python3
"""
Toy OpenTelemetry example for a fork-based Python multiprocessing workload.

Mirrors the pattern used by the spider query cronjob
(docker/spider-query-cronjob/src/otel_setup.py):

Why a special setup is needed
-----------------------------
setup_opentelemetry() starts OTLP gRPC exporters with background threads.
ProcessPoolExecutor (fork) children inherit that state, which is unsafe:
gRPC channels break after fork, and OpenTelemetry refuses to replace global
MeterProvider / TracerProvider / LoggerProvider ("Overriding ... is not allowed").

The fix
-------
1. Set GRPC_ENABLE_FORK_SUPPORT=true and GRPC_POLL_STRATEGY=poll.
2. Pass an initializer to ProcessPoolExecutor that:
   - shuts down inherited providers
   - resets OpenTelemetry's set-once flags
   - installs a fresh OTLP stack in the child
3. Rebuild metric instruments per process (lazy, PID-keyed cache).
4. Drop inherited OTLP LoggingHandlers before attaching a new one
   (avoids duplicated log export).

Prerequisites:
1) Python 3.10+
2) Install dependencies:
    pip install opentelemetry-api opentelemetry-sdk \
    opentelemetry-exporter-otlp-proto-grpc

Collector configuration:
- Uses OTLP gRPC endpoint from OTEL_EXPORTER_OTLP_ENDPOINT.
- Defaults to: opentelemetry-collector.opentelemetry.svc.cluster.local:4317
- Basic authentication:
    - OPENTELEMETRY_USERNAME
    - OPENTELEMETRY_PASSWORD

Run:
    OTEL_SERVICE_NAME=python-mp-toy-app python3 python_multiprocessing_otel_example.py
"""

from __future__ import annotations

import base64
import logging
import os
import time
from concurrent.futures import ProcessPoolExecutor, as_completed
from typing import Any, Callable

# Prefer gRPC's fork-aware poller when processes will be forked after OTLP init.
os.environ.setdefault("GRPC_ENABLE_FORK_SUPPORT", "true")
os.environ.setdefault("GRPC_POLL_STRATEGY", "poll")

import opentelemetry._logs._internal as _logs_internal
import opentelemetry.metrics._internal as _metrics_internal
import opentelemetry.trace as _trace_api
from opentelemetry import metrics, trace
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.metrics import Meter
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import Tracer

ENDPOINT = os.getenv(
    "OTEL_EXPORTER_OTLP_ENDPOINT",
    "opentelemetry-collector.opentelemetry.svc.cluster.local:4317",
)
SERVICE_NAME = os.getenv("OTEL_SERVICE_NAME", "python-mp-toy-app")
METRIC_INTERVAL_MS = int(os.getenv("OTEL_METRIC_EXPORT_INTERVAL", "15000"))
WORKER_COUNT = int(os.getenv("EXAMPLE_WORKER_PROCESSES", "2"))
TASK_COUNT = int(os.getenv("EXAMPLE_WORK_ITEMS", "4"))

global_meter: Meter
global_tracer: Tracer


def _otlp_headers() -> dict[str, str]:
    username = os.getenv("OPENTELEMETRY_USERNAME", "")
    password = os.getenv("OPENTELEMETRY_PASSWORD", "")
    token = base64.b64encode(f"{username}:{password}".encode("utf-8")).decode("utf-8")
    return {"authorization": f"Basic {token}"}


def _otel_resource() -> Resource:
    return Resource.create(
        {
            "service.name": SERVICE_NAME,
            "process.pid": os.getpid(),
        }
    )


def _remove_otel_log_handlers() -> None:
    root = logging.getLogger()
    for handler in root.handlers[:]:
        if isinstance(handler, LoggingHandler):
            root.removeHandler(handler)


def _setup_meter(resource: Resource) -> tuple[Meter, MeterProvider]:
    headers = _otlp_headers()
    metric_reader = PeriodicExportingMetricReader(
        OTLPMetricExporter(endpoint=ENDPOINT, insecure=True, headers=headers),
        export_interval_millis=METRIC_INTERVAL_MS,
    )
    provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
    metrics.set_meter_provider(provider)
    return metrics.get_meter("examples.python.multiprocessing"), provider


def _setup_tracer(resource: Resource) -> tuple[Tracer, TracerProvider]:
    headers = _otlp_headers()
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(
        BatchSpanProcessor(OTLPSpanExporter(endpoint=ENDPOINT, insecure=True, headers=headers))
    )
    trace.set_tracer_provider(provider)
    return trace.get_tracer("examples.python.multiprocessing"), provider


def _setup_logging(resource: Resource) -> LoggerProvider:
    _remove_otel_log_handlers()

    headers = _otlp_headers()
    logger_provider = LoggerProvider(resource=resource)
    set_logger_provider(logger_provider)
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(
            OTLPLogExporter(endpoint=ENDPOINT, insecure=True, headers=headers)
        )
    )

    root = logging.getLogger()
    root.setLevel(logging.INFO)
    if not any(isinstance(h, logging.StreamHandler) and not isinstance(h, LoggingHandler)
               for h in root.handlers):
        root.addHandler(logging.StreamHandler())
    root.addHandler(LoggingHandler(logger_provider=logger_provider))
    return logger_provider


def setup_opentelemetry() -> tuple[Meter, Tracer]:
    """Initialize OTLP exporters in the current process (parent or reinitialized child)."""
    resource = _otel_resource()
    meter, _ = _setup_meter(resource)
    tracer, _ = _setup_tracer(resource)
    _setup_logging(resource)
    logging.getLogger(__name__).info(
        "OpenTelemetry initialized in pid=%s service=%s endpoint=%s",
        os.getpid(),
        SERVICE_NAME,
        ENDPOINT,
    )
    return meter, tracer


def _shutdown_inherited_otel() -> None:
    """Stop inherited parent export threads and allow a fresh provider install after fork."""
    meter_provider = _metrics_internal.get_meter_provider()
    if isinstance(meter_provider, MeterProvider):
        meter_provider.shutdown()

    tracer_provider = _trace_api.get_tracer_provider()
    if isinstance(tracer_provider, TracerProvider):
        tracer_provider.shutdown()

    logger_provider = _logs_internal.get_logger_provider()
    if isinstance(logger_provider, LoggerProvider):
        logger_provider.shutdown()

    # OpenTelemetry providers are set-once; clear the guards so children can replace
    # the broken inherited providers. This touches private API that spider also uses.
    _metrics_internal._METER_PROVIDER_SET_ONCE._done = False
    _trace_api._TRACER_PROVIDER_SET_ONCE._done = False
    _logs_internal._LOGGER_PROVIDER_SET_ONCE._done = False


def reinit_otel_in_worker() -> None:
    """Recreate OTLP export in forked workers; inherited gRPC state is unsafe after fork."""
    global global_meter, global_tracer

    _shutdown_inherited_otel()
    global_meter, global_tracer = setup_opentelemetry()


def lazy_instruments(factory: Callable[[Meter], dict[str, Any]]) -> dict[str, Any]:
    """Return instruments for the current process; recreate after fork/reinit."""
    cache = getattr(factory, "_cache", None)
    pid = os.getpid()
    if cache is None or cache[0] != pid:
        cache = (pid, factory(global_meter))
        factory._cache = cache
    return cache[1]


def _create_worker_instruments(meter: Meter) -> dict[str, Any]:
    return {
        "tasks": meter.create_counter(
            "toy_mp_tasks_processed_total",
            description="Work items processed by multiprocessing workers",
        ),
    }


def _worker_metrics() -> dict[str, Any]:
    return lazy_instruments(_create_worker_instruments)


def process_work_item(item_id: int) -> dict[str, Any]:
    """Worker entrypoint: emit a span, metric, and log for one item."""
    logger = logging.getLogger("python-mp-toy-app")
    with global_tracer.start_as_current_span("process_work_item") as span:
        span.set_attribute("app.item_id", item_id)
        span.set_attribute("process.pid", os.getpid())
        _worker_metrics()["tasks"].add(1, {"app.component": "mp-worker"})
        logger.info("Worker pid=%s processed item %s", os.getpid(), item_id)
        time.sleep(0.1)
    return {"item_id": item_id, "pid": os.getpid()}


# Parent process initializes OTel at import time (same pattern as spider).
global_meter, global_tracer = setup_opentelemetry()


def main() -> None:
    logger = logging.getLogger("python-mp-toy-app")
    logger.info(
        "Parent pid=%s starting ProcessPoolExecutor workers=%s tasks=%s",
        os.getpid(),
        WORKER_COUNT,
        TASK_COUNT,
    )

    with global_tracer.start_as_current_span("multiprocessing_batch") as span:
        span.set_attribute("app.task_count", TASK_COUNT)
        span.set_attribute("app.worker_count", WORKER_COUNT)

        results: list[dict[str, Any]] = []
        # initializer re-creates exporters in each forked child before any work runs.
        with ProcessPoolExecutor(
            max_workers=WORKER_COUNT,
            initializer=reinit_otel_in_worker,
        ) as executor:
            futures = {
                executor.submit(process_work_item, item_id): item_id
                for item_id in range(1, TASK_COUNT + 1)
            }
            for future in as_completed(futures):
                results.append(future.result())

        span.set_attribute("app.completed_count", len(results))

    logger.info("Completed %s tasks from pids=%s", len(results), sorted({r["pid"] for r in results}))

    # Flush parent exporters before exit.
    meter_provider = _metrics_internal.get_meter_provider()
    tracer_provider = _trace_api.get_tracer_provider()
    logger_provider = _logs_internal.get_logger_provider()
    if isinstance(logger_provider, LoggerProvider):
        logger_provider.force_flush()
    if isinstance(tracer_provider, TracerProvider):
        tracer_provider.force_flush()
    if isinstance(meter_provider, MeterProvider):
        meter_provider.force_flush()


if __name__ == "__main__":
    main()
