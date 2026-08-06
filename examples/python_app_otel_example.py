#!/usr/bin/env python3
"""
Toy OpenTelemetry example for a general Python application.

What this shows:
- Traces: creates a span around each unit of work.
- Metrics: increments a counter for each processed work item.
- Logs: emits logs through OpenTelemetry to the collector.

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
    OTEL_SERVICE_NAME=python-toy-app python3 python_app_otel_example.py
"""

from __future__ import annotations

import base64
import logging
import os
import signal
import time

from opentelemetry import metrics, trace
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor


def _otlp_headers() -> dict[str, str]:
    username = os.getenv("OPENTELEMETRY_USERNAME", "")
    password = os.getenv("OPENTELEMETRY_PASSWORD", "")
    token = base64.b64encode(f"{username}:{password}".encode("utf-8")).decode("utf-8")
    return {"authorization": f"Basic {token}"}


def configure_otel():
    endpoint = os.getenv(
        "OTEL_EXPORTER_OTLP_ENDPOINT",
        "opentelemetry-collector.opentelemetry.svc.cluster.local:4317",
    )
    service_name = os.getenv("OTEL_SERVICE_NAME", "python-toy-app")
    metric_interval_ms = int(os.getenv("OTEL_METRIC_EXPORT_INTERVAL", "15000"))
    headers = _otlp_headers()

    resource = Resource.create({"service.name": service_name})

    trace_provider = TracerProvider(resource=resource)
    trace_provider.add_span_processor(
        BatchSpanProcessor(OTLPSpanExporter(endpoint=endpoint, insecure=True, headers=headers))
    )
    trace.set_tracer_provider(trace_provider)

    metric_reader = PeriodicExportingMetricReader(
        OTLPMetricExporter(endpoint=endpoint, insecure=True, headers=headers),
        export_interval_millis=metric_interval_ms,
    )
    meter_provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
    metrics.set_meter_provider(meter_provider)

    logger_provider = LoggerProvider(resource=resource)
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(
            OTLPLogExporter(endpoint=endpoint, insecure=True, headers=headers)
        )
    )
    set_logger_provider(logger_provider)

    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)
    root_logger.handlers.clear()
    root_logger.addHandler(logging.StreamHandler())
    root_logger.addHandler(LoggingHandler(logger_provider=logger_provider))

    return trace_provider, meter_provider, logger_provider


def main() -> None:
    trace_provider, meter_provider, logger_provider = configure_otel()
    tracer = trace.get_tracer("examples.python.app")
    meter = metrics.get_meter("examples.python.app")
    jobs_counter = meter.create_counter("toy_jobs_processed_total")
    logger = logging.getLogger("python-toy-app")

    running = True

    def _stop(_sig, _frame):
        nonlocal running
        running = False

    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)

    logger.info("Toy app started")
    iteration = 0
    while running:
        iteration += 1
        with tracer.start_as_current_span("process_work_item") as span:
            span.set_attribute("app.iteration", iteration)
            jobs_counter.add(1, {"app.component": "worker"})
            logger.info("Processed toy work item %s", iteration)
        time.sleep(2)

    logger.info("Toy app shutting down")
    logger_provider.force_flush()
    trace_provider.force_flush()
    meter_provider.force_flush()


if __name__ == "__main__":
    main()
