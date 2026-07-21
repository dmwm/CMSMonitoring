#!/usr/bin/env python
"""
OpenTelemetry setup for sending metrics to CERN monitoring endpoint.
"""

import base64
import logging
import contextvars
import opentelemetry
import sys
from typing import Optional, Tuple
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.metrics import Meter
from opentelemetry import trace, propagate
from opentelemetry.trace import format_trace_id
from opentelemetry.context import Context
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

import src.constants as const

# Context variable to store additional metric attributes during function execution
_metric_attributes = contextvars.ContextVar('metric_attributes', default={})

# Global logger provider/handler references
_logger_provider = None
_otel_handler = None
_worker_span = None
_worker_scope = None
_worker_traceparent = None
_worker_missing_traceparent_warned = False


class CustomLoggingHandler(LoggingHandler):
    """Custom LoggingHandler that adds service.name to log record attributes"""
    
    def __init__(self, service_name: str, service_version: str, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._service_name = service_name
        self._service_version = service_version
    
    def _get_attributes(self, record: logging.LogRecord):
        """Override to add service.name and service.version to attributes"""
        # Get base attributes from parent
        attributes = super()._get_attributes(record)
        
        # Add service.name and service.version to log attributes (highest precedence)
        attributes["service.name"] = self._service_name
        attributes["service.version"] = self._service_version

        # Always derive execution.id from the active trace context when available.
        span_context = trace.get_current_span().get_span_context()
        if span_context and span_context.is_valid:
            attributes["execution.id"] = format_trace_id(span_context.trace_id)
        
        return attributes


def set_up_logging(log_level: int = logging.INFO) -> logging.Logger:
    """Configure global logging and attach it to OpenTelemetry"""
    
    # Resource.create() automatically merges with default resource and OTELResourceDetector
    # Ensure values are not None to prevent service.name from being None
    logging_resource = Resource.create({
        "service.name": const.IMAGE_NAME or "spider-worker",
        "service.version": const.DOCKER_TAG or "unknown",
    })
    logger_provider = LoggerProvider(resource=logging_resource)
    set_logger_provider(logger_provider)
    
    creds = f"{const.OTEL_USERNAME}:{const.OTEL_PASSWORD}".encode("utf-8")
    token = base64.b64encode(creds).decode("utf-8")
    otel_headers = {"authorization": f"Basic {token}"}
    channel_options=(
        ("grpc.keepalive_time_ms", 20000),
        ("grpc.keepalive_timeout_ms", 10000),
        ("grpc.keepalive_permit_without_calls", True),
        ("grpc.http2.max_pings_without_data", 0),
        ("grpc.http2.min_time_between_pings_ms", 10000),
    )
    
    exporter = OTLPLogExporter(
        endpoint=const.OTEL_ENDPOINT,
        insecure=True,
        headers=otel_headers,
        channel_options=channel_options,
    )
    
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(exporter)
    )
    
    logger = logging.getLogger()
    logger.setLevel(log_level)

    # Ensure we don't duplicate handlers if logging is initialized multiple times.
    logger.handlers.clear()

    # Stream handler for stdout
    stream_handler = logging.StreamHandler(sys.stdout)
    stream_handler.setFormatter(logging.Formatter('%(asctime)s : %(name)s:%(levelname)s - %(message)s'))
    
    # OpenTelemetry handler - adds service.name directly to log attributes
    service_name = const.IMAGE_NAME or "spider-worker"
    service_version = const.DOCKER_TAG or "unknown"
    otel_handler = CustomLoggingHandler(
        service_name=service_name,
        service_version=service_version,
        logger_provider=logger_provider,
        level=log_level
    )
    
    # Store globals for later updates
    global _logger_provider, _otel_handler
    _logger_provider = logger_provider
    _otel_handler = otel_handler
    
    logger.addHandler(stream_handler)
    logger.addHandler(otel_handler)

    # Set StompAMQ logger to ERROR level to suppress WARNING messages
    logging.getLogger("StompAMQ").setLevel(logging.ERROR)
    logging.getLogger("opensearch").setLevel(logging.ERROR)
    
    if log_level <= logging.INFO:
        logging.getLogger("stomp.py").setLevel(log_level + 10)
    
    return logger


def create_parent_context_from_trace_headers(
    traceparent: Optional[str], tracestate: Optional[str] = None
) -> Optional[Context]:
    """Build a parent context from W3C trace headers."""
    if not traceparent:
        return None

    carrier = {"traceparent": traceparent}
    if tracestate:
        carrier["tracestate"] = tracestate

    parent_context = propagate.extract(carrier)
    span_context = trace.get_current_span(parent_context).get_span_context()
    if not span_context.is_valid:
        return None
    return parent_context


def _execution_id_from_parent_context(parent_context: Context) -> Optional[str]:
    """Return trace-id string from extracted parent context."""
    span_context = trace.get_current_span(parent_context).get_span_context()
    if not span_context.is_valid:
        return None
    return format_trace_id(span_context.trace_id)


def start_worker_span(
    parent_context: Context, execution_id: Optional[str] = None
) -> Tuple[trace.Span, object]:
    """
    Start a long-lived worker span attached to the supplied parent context.
    Returns (span, scope) where scope must be exited by the caller.
    """
    lifetime_span = global_tracer.start_span(
        name="spider_worker",
        context=parent_context,
    )
    if execution_id:
        lifetime_span.set_attribute("execution.id", execution_id)
    lifetime_span.set_attribute("worker.service", const.OTEL_SERVICE_NAME)

    scope = trace.use_span(lifetime_span, end_on_exit=False)
    scope.__enter__()
    return lifetime_span, scope


def initialize_worker_trace(
    traceparent: Optional[str], tracestate: Optional[str]
) -> None:
    """
    Initialize the worker lifetime span once from remote trace headers.
    On failure, logs a warning.
    """
    global _worker_span, _worker_scope
    global _worker_traceparent, _worker_missing_traceparent_warned

    if _worker_span is not None:
        if traceparent and traceparent != _worker_traceparent:
            # TODO: Rotate the worker lifetime span when a new execution traceparent arrives.
            global_logger.warning(
                "Received different root traceparent after worker trace initialization; ignoring. "
                "current=%s new=%s",
                _worker_traceparent,
                traceparent,
            )
        return

    if not traceparent:
        if not _worker_missing_traceparent_warned:
            global_logger.warning(
                "Failed to initialize worker lifetime trace: missing traceparent header"
            )
            _worker_missing_traceparent_warned = True
        return

    parent_context = create_parent_context_from_trace_headers(traceparent, tracestate)
    if parent_context is None:
        global_logger.warning(
            "Failed to initialize worker lifetime trace: invalid traceparent=%s",
            traceparent,
        )
        return

    execution_id = _execution_id_from_parent_context(parent_context)
    _worker_span, _worker_scope = start_worker_span(parent_context, execution_id)
    _worker_traceparent = traceparent
    global_logger.info("Initialized worker lifetime span from traceparent header")


def finalize_worker_trace(
    worker_seconds: float,
    total_jobs: int,
    total_sent_amq: int,
    total_sent_os: int,
) -> None:
    """End and detach the lifetime span if it was initialized."""
    global _worker_span, _worker_scope

    if _worker_span is not None:
        _worker_span.set_attribute(
            "worker.lifetime.seconds", worker_seconds
        )
        _worker_span.set_attribute("worker.total_jobs", total_jobs)
        _worker_span.set_attribute("worker.total_sent_amq", total_sent_amq)
        _worker_span.set_attribute("worker.total_sent_os", total_sent_os)
        _worker_span.end()
        _worker_span = None

    if _worker_scope is not None:
        _worker_scope.__exit__(None, None, None)
        _worker_scope = None


def setup_opentelemetry() -> Tuple[Meter, logging.Logger, trace.Tracer]:
    """
    Initialize OpenTelemetry metrics + tracing exporters and logging.
    
    Returns:
        tuple: (Meter, Logger, Tracer)
    """
    # OTLP endpoint configuration
    # For gRPC, endpoint should be host:port without protocol scheme
    otlp_endpoint = const.OTEL_ENDPOINT

    
    # Use a fixed service name so worker spans are visually distinct in Grafana.
    service_name = const.OTEL_SERVICE_NAME
    service_version = const.DOCKER_TAG
    
    
    # Create resource with service information and run ID
    # Resource.create() automatically merges with default resource and OTELResourceDetector
    # Ensure values are not None to prevent service.name from being None
    resource = Resource.create({
        "service.name": service_name,
        "service.version": service_version or "unknown",
    })
    
    channel_options=(
        ("grpc.keepalive_time_ms", 20000),
        ("grpc.keepalive_timeout_ms", 10000),
        ("grpc.keepalive_permit_without_calls", True),
        ("grpc.http2.max_pings_without_data", 0),
        ("grpc.http2.min_time_between_pings_ms", 10000),
    )
    
    creds = f"{const.OTEL_USERNAME}:{const.OTEL_PASSWORD}".encode("utf-8")
    token = base64.b64encode(creds).decode("utf-8")
    otel_headers = {"authorization": f"Basic {token}"}

    # Create OTLP metric exporter
    metric_exporter = OTLPMetricExporter(
        endpoint=otlp_endpoint,
        headers=otel_headers,
        channel_options=channel_options,
        insecure=True,
    )
    
    # Create metric reader with periodic export
    metric_reader = PeriodicExportingMetricReader(
        exporter=metric_exporter,
        export_interval_millis=int(const.OTEL_METRIC_EXPORT_INTERVAL),  # Default 60s
    )
    
    # Create and set global meter provider
    provider = MeterProvider(
        resource=resource,
        metric_readers=[metric_reader],
    )
    opentelemetry.metrics.set_meter_provider(provider)
    
    # Get meter for creating metrics
    meter = opentelemetry.metrics.get_meter(__name__)

    # Create OTLP trace exporter/provider so worker can emit spans.
    trace_exporter = OTLPSpanExporter(
        endpoint=otlp_endpoint,
        headers=otel_headers,
        channel_options=channel_options,
        insecure=True,
    )
    trace_provider = TracerProvider(resource=resource)
    trace_provider.add_span_processor(BatchSpanProcessor(trace_exporter))
    trace.set_tracer_provider(trace_provider)
    tracer = trace.get_tracer(__name__)

    logger = set_up_logging()
    logger.warning(
        f"OpenTelemetry metrics and tracing initialized: endpoint={otlp_endpoint}, "
        f"service={service_name}, version={service_version}"
    )
    
    return meter, logger, tracer


global_meter, global_logger, global_tracer = setup_opentelemetry()
