#!/usr/bin/env python3
"""
Toy OpenTelemetry example for a CherryPy application.

What this shows:
- Traces: creates one span per HTTP request.
- Metrics: increments a request counter per endpoint.
- Logs: emits request logs through OpenTelemetry to the collector.

Prerequisites:
1) Python 3.10+
2) Install dependencies:
    pip install CherryPy opentelemetry-api opentelemetry-sdk \
    opentelemetry-exporter-otlp-proto-grpc

Collector configuration:
- Uses OTLP gRPC endpoint from OTEL_EXPORTER_OTLP_ENDPOINT.
- Defaults to: http://cms-monitoring-test.cern.ch:30428
- Basic authentication:
    - OPENTELEMETRY_USERNAME
    - OPENTELEMETRY_PASSWORD

Run:
    OTEL_SERVICE_NAME=cherrypy-toy-app python3 cherrypy_app_otel_example.py
Then open:
    http://127.0.0.1:8080/
"""

from __future__ import annotations

import base64
import logging
import os
import time

import cherrypy
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
    service_name = os.getenv("OTEL_SERVICE_NAME", "cherrypy-toy-app")
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


TRACE_PROVIDER, METER_PROVIDER, LOGGER_PROVIDER = configure_otel()
TRACER = trace.get_tracer("examples.cherrypy.app")
METER = metrics.get_meter("examples.cherrypy.app")
REQUEST_COUNTER = METER.create_counter("toy_http_requests_total")
LOGGER = logging.getLogger("cherrypy-toy-app")


class ToyApp:
    @cherrypy.expose
    def index(self) -> str:
        with TRACER.start_as_current_span("GET /"):
            REQUEST_COUNTER.add(1, {"http.route": "/"})
            LOGGER.info("Handled CherryPy request", extra={"path": "/"})
            time.sleep(0.05)
            return "Hello from CherryPy + OpenTelemetry toy example\n"


if __name__ == "__main__":
    cherrypy.config.update({"server.socket_host": "127.0.0.1", "server.socket_port": 8080})
    LOGGER.info("Starting CherryPy toy app")
    cherrypy.quickstart(ToyApp())
    LOGGER_PROVIDER.force_flush()
    TRACE_PROVIDER.force_flush()
    METER_PROVIDER.force_flush()
