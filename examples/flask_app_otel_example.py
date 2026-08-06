#!/usr/bin/env python3
"""
Toy OpenTelemetry example for a Flask application.

What this shows:
- Traces: creates one span per HTTP request.
- Metrics: increments a request counter per endpoint.
- Logs: emits request logs through OpenTelemetry to the collector.

Prerequisites:
1) Python 3.10+
2) Install dependencies:
    pip install flask opentelemetry-api opentelemetry-sdk \
    opentelemetry-exporter-otlp-proto-grpc

Collector configuration:
- Uses OTLP gRPC endpoint from OTEL_EXPORTER_OTLP_ENDPOINT.
- Defaults to: opentelemetry-collector.opentelemetry.svc.cluster.local:4317
- Basic authentication:
    - OPENTELEMETRY_USERNAME
    - OPENTELEMETRY_PASSWORD

Run:
    OTEL_SERVICE_NAME=flask-toy-app python3 flask_app_otel_example.py
Then open:
    http://127.0.0.1:5000/
"""

from __future__ import annotations

import base64
import logging
import os
import time

from flask import Flask, request
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
    service_name = os.getenv("OTEL_SERVICE_NAME", "flask-toy-app")
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
TRACER = trace.get_tracer("examples.flask.app")
METER = metrics.get_meter("examples.flask.app")
REQUEST_COUNTER = METER.create_counter("toy_http_requests_total")
LOGGER = logging.getLogger("flask-toy-app")

app = Flask(__name__)


@app.route("/")
def index() -> str:
    with TRACER.start_as_current_span("GET /") as span:
        span.set_attribute("http.method", request.method)
        span.set_attribute("http.route", "/")
        REQUEST_COUNTER.add(1, {"http.route": "/"})
        LOGGER.info("Handled Flask request", extra={"path": "/"})
        time.sleep(0.05)
        return "Hello from Flask + OpenTelemetry toy example\n"


if __name__ == "__main__":
    LOGGER.info("Starting Flask toy app")
    try:
        app.run(host="127.0.0.1", port=5000, debug=False)
    finally:
        LOGGER_PROVIDER.force_flush()
        TRACE_PROVIDER.force_flush()
        METER_PROVIDER.force_flush()
