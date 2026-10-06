#!/usr/bin/env python
"""
OpenTelemetry setup for sending metrics to CERN monitoring endpoint.
"""

import base64
import logging
import os
import uuid
import functools
import contextvars
import opentelemetry.metrics._internal as _metrics_internal
import opentelemetry.trace as _trace_api
import opentelemetry._logs._internal as _logs_internal
import opentelemetry
from opentelemetry.metrics import Meter
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.trace import Tracer, Status, StatusCode
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry import trace
from opentelemetry.trace import format_trace_id
from typing import Any, Callable, Dict, Tuple

import constants as const

logger = logging.getLogger(__name__)

# Global execution ID for this cronjob run
EXECUTION_ID = str(uuid.uuid4())
_root_span_context = contextvars.ContextVar("root_span_context", default=None)

global_meter: Meter
global_tracer: Tracer


def get_execution_id() -> str:
    """Return the current trace ID; fallback to process execution ID."""
    span_context = trace.get_current_span().get_span_context()
    if span_context and span_context.is_valid:
        return format_trace_id(span_context.trace_id)
    return EXECUTION_ID


def get_root_span_context():
    """Return the root span context for the current execution context, if any."""
    return _root_span_context.get()


class CustomLoggingHandler(LoggingHandler):
    """Custom LoggingHandler that adds service.name to log record attributes"""

    def __init__(self, service_name: str, service_version: str, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._service_name = service_name
        self._service_version = service_version

    def _get_attributes(self, record: logging.LogRecord):
        """Override to add stable log attributes with highest precedence."""
        attributes = super()._get_attributes(record)

        attributes["service.name"] = self._service_name
        attributes["service.version"] = self._service_version
        attributes["execution.id"] = get_execution_id()

        return attributes


def _otel_resource() -> Resource:
    return Resource.create({
        "service.name": const.OTEL_SERVICE_NAME,
        "service.version": const.DOCKER_TAG,
        "execution.id": EXECUTION_ID,
    })


def _otel_headers() -> Dict[str, str]:
    creds = f"{const.OTEL_USERNAME}:{const.OTEL_PASSWORD}".encode("utf-8")
    token = base64.b64encode(creds).decode("utf-8")
    return {"authorization": f"Basic {token}"}


def _otel_channel_options() -> Tuple[Tuple[str, Any], ...]:
    return (
        ("grpc.keepalive_time_ms", 20000),
        ("grpc.keepalive_timeout_ms", 10000),
        ("grpc.keepalive_permit_without_calls", False),
        ("grpc.http2.min_time_between_pings_ms", 30000),
    )


def _setup_meter(resource: Resource) -> Tuple[Meter, MeterProvider]:
    metric_exporter = OTLPMetricExporter(
        endpoint=const.OTEL_ENDPOINT,
        headers=_otel_headers(),
        channel_options=_otel_channel_options(),
        insecure=True,
    )
    metric_reader = PeriodicExportingMetricReader(
        exporter=metric_exporter,
        export_interval_millis=int(const.OTEL_METRIC_EXPORT_INTERVAL),
    )
    provider = MeterProvider(
        resource=resource,
        metric_readers=[metric_reader],
    )
    opentelemetry.metrics.set_meter_provider(provider)
    return opentelemetry.metrics.get_meter(__name__), provider


def _setup_tracer(resource: Resource) -> Tuple[Tracer, TracerProvider]:
    trace_exporter = OTLPSpanExporter(
        endpoint=const.OTEL_ENDPOINT,
        headers=_otel_headers(),
        channel_options=_otel_channel_options(),
        insecure=True,
    )
    trace_provider = TracerProvider(resource=resource)
    trace_provider.add_span_processor(BatchSpanProcessor(trace_exporter))
    opentelemetry.trace.set_tracer_provider(trace_provider)
    return opentelemetry.trace.get_tracer(__name__), trace_provider


def _remove_otel_log_handlers() -> None:
    root = logging.getLogger()
    for handler in root.handlers[:]:
        if isinstance(handler, CustomLoggingHandler):
            root.removeHandler(handler)


def _setup_logging(resource: Resource, register_global: bool = True) -> LoggerProvider:
    _remove_otel_log_handlers()

    logger_provider = LoggerProvider(resource=resource)
    if register_global:
        set_logger_provider(logger_provider)

    log_exporter = OTLPLogExporter(
        endpoint=const.OTEL_ENDPOINT,
        headers=_otel_headers(),
        channel_options=_otel_channel_options(),
        insecure=True,
    )
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(log_exporter)
    )

    root_logger = logging.getLogger()
    otel_handler = CustomLoggingHandler(
        service_name=const.OTEL_SERVICE_NAME,
        service_version=const.DOCKER_TAG,
        logger_provider=logger_provider,
        level=logging.INFO,
    )
    root_logger.addHandler(otel_handler)
    return logger_provider


def setup_opentelemetry() -> Tuple[Meter, Tracer]:
    """
    Initialize OpenTelemetry metrics, tracing, and logging exporters.

    Returns:
        tuple: (Meter, Tracer) - The configured meter and tracer instances
    """
    resource = _otel_resource()
    meter, _ = _setup_meter(resource)
    tracer, _ = _setup_tracer(resource)
    _setup_logging(resource)

    logger.warning(
        "OpenTelemetry metrics, tracing, and logging initialized: endpoint=%s, "
        "service=%s, version=%s, execution_id=%s",
        const.OTEL_ENDPOINT,
        const.IMAGE_NAME,
        const.DOCKER_TAG,
        EXECUTION_ID,
    )

    return meter, tracer


def _shutdown_inherited_otel() -> None:
    """Stop inherited parent export threads and allow fresh provider install after fork."""
    meter_provider = _metrics_internal.get_meter_provider()
    if isinstance(meter_provider, MeterProvider):
        meter_provider.shutdown()

    tracer_provider = _trace_api.get_tracer_provider()
    if isinstance(tracer_provider, TracerProvider):
        tracer_provider.shutdown()

    logger_provider = _logs_internal.get_logger_provider()
    if isinstance(logger_provider, LoggerProvider):
        logger_provider.shutdown()

    _metrics_internal._METER_PROVIDER_SET_ONCE._done = False
    _trace_api._TRACER_PROVIDER_SET_ONCE._done = False
    _logs_internal._LOGGER_PROVIDER_SET_ONCE._done = False


def reinit_otel_in_worker() -> None:
    """Re-create OTLP export in forked workers; inherited gRPC state is unsafe after fork."""
    global global_meter, global_tracer

    _shutdown_inherited_otel()
    resource = _otel_resource()
    global_meter, _ = _setup_meter(resource)
    global_tracer, _ = _setup_tracer(resource)
    _setup_logging(resource)

    logger.debug(
        "OpenTelemetry reinitialized in worker pid=%s execution_id=%s",
        os.getpid(),
        EXECUTION_ID,
    )


def lazy_instruments(factory: Callable[[Meter], Dict[str, Any]]) -> Dict[str, Any]:
    """Return instruments for the current process; recreate after fork/reinit."""
    cache = getattr(factory, "_cache", None)
    pid = os.getpid()
    if cache is None or cache[0] != pid:
        cache = (pid, factory(global_meter))
        factory._cache = cache
    return cache[1]


global_meter, global_tracer = setup_opentelemetry()


def trace_span(span_name: str = None, **attributes):
    """
    Decorator that creates an OpenTelemetry span for a function execution.
    The span includes timing information and can have attributes set during execution.

    Args:
        span_name: Name of the span (defaults to function name)
        **attributes: Additional attributes to set on the span

    Example usage:
    @trace_span("query_schedd_queue", schedd="crab3@vocms0195.cern.ch")
    def query_schedd_queue(starttime, schedd_ads):
        # ... function code ...
        with global_tracer.start_as_current_span("nested_operation") as span:
            span.set_attribute("jobs_count", count)
        return count
    """
    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            name = span_name or f"{const.OTEL_SERVICE_NAME}.{func.__module__}.{func.__name__}"
            parent_span_context = trace.get_current_span().get_span_context()
            is_root_span = not (
                parent_span_context and parent_span_context.is_valid
            )
            root_token = None
            with global_tracer.start_as_current_span(name) as span:
                if is_root_span:
                    root_token = _root_span_context.set(span.get_span_context())
                for key, value in attributes.items():
                    span.set_attribute(key, value)

                span.set_attribute("execution.id", get_execution_id())

                try:
                    result = func(*args, **kwargs)
                    span.set_status(Status(StatusCode.OK))
                    return result
                except Exception as e:
                    span.set_status(Status(StatusCode.ERROR, str(e)))
                    span.record_exception(e)
                    raise
                finally:
                    if root_token is not None:
                        _root_span_context.reset(root_token)

        return wrapper
    return decorator
