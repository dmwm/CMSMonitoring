#!/usr/bin/env python
"""
OpenTelemetry setup for sending metrics to CERN monitoring endpoint.
"""

import base64
import logging
import uuid
import functools
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
from typing import Tuple

import constants as const

logger = logging.getLogger(__name__)

# Global execution ID for this cronjob run
EXECUTION_ID = str(uuid.uuid4())


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
        
        return attributes

# Generate a unique execution ID for this cronjob run - shared across all processes
# This will be included in NATS message headers to link worker pod logs back to this run



def setup_opentelemetry() -> Tuple[Meter, Tracer]:
    """
    Initialize OpenTelemetry metrics and tracing exporters to send data to CERN monitoring endpoint.
    
    Returns:
        tuple: (Meter, Tracer) - The configured meter and tracer instances
    """
    
    # Create resource with service information and run ID
    resource = Resource.create({
        "service.name": const.OTEL_SERVICE_NAME,
        "service.version": const.DOCKER_TAG,
        "execution.id": EXECUTION_ID,
    })
    
    channel_options=(
        ("grpc.keepalive_time_ms", 20000),
        ("grpc.keepalive_timeout_ms", 10000),
        ("grpc.keepalive_permit_without_calls", False),
        ("grpc.http2.min_time_between_pings_ms", 30000),
    )
    
    creds = f"{const.OTEL_USERNAME}:{const.OTEL_PASSWORD}".encode("utf-8")
    token = base64.b64encode(creds).decode("utf-8")
    otel_headers = {"authorization": f"Basic {token}"}

    # Create OTLP metric exporter
    metric_exporter = OTLPMetricExporter(
        endpoint=f"{const.OTEL_ENDPOINT}/v1/metrics",
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
    
    # Create OTLP trace exporter
    trace_exporter = OTLPSpanExporter(
        endpoint=f"{const.OTEL_ENDPOINT}/v1/traces",
        headers=otel_headers,
        channel_options=channel_options,
        insecure=True,
    )
    
    # Create and set global tracer provider
    trace_provider = TracerProvider(resource=resource)
    trace_provider.add_span_processor(BatchSpanProcessor(trace_exporter))
    opentelemetry.trace.set_tracer_provider(trace_provider)
    
    # Get tracer for creating spans
    tracer = opentelemetry.trace.get_tracer(__name__)
    
    # Create and set global logger provider for OpenTelemetry logging
    logger_provider = LoggerProvider(resource=resource)
    set_logger_provider(logger_provider)
    
    # Create OTLP log exporter
    log_exporter = OTLPLogExporter(
        endpoint=f"{const.OTEL_ENDPOINT}/v1/logs",
        headers=otel_headers,
        channel_options=channel_options,
        insecure=True,
    )
    
    # Add log record processor
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(log_exporter)
    )
    
    # Get root logger and add OpenTelemetry handler
    # Use custom handler that adds service.name to log attributes (highest precedence)
    root_logger = logging.getLogger()
    otel_handler = CustomLoggingHandler(
        service_name=const.OTEL_SERVICE_NAME,
        service_version=const.DOCKER_TAG,
        logger_provider=logger_provider,
        level=logging.INFO
    )
    root_logger.addHandler(otel_handler)
    
    logger.warning(
        f"OpenTelemetry metrics, tracing, and logging initialized: endpoint={const.OTEL_ENDPOINT}, "
        f"service={const.IMAGE_NAME}, version={const.DOCKER_TAG}, execution_id={EXECUTION_ID}"
    )
    
    return meter, tracer


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
            # Use provided span_name or default to function name
            name = span_name or f"{const.OTEL_SERVICE_NAME}.{func.__module__}.{func.__name__}"
            # Start a new span
            with global_tracer.start_as_current_span(name) as span:
                # Set initial attributes
                for key, value in attributes.items():
                    span.set_attribute(key, value)
                
                # Set span kind to INTERNAL (default for internal operations)
                span.set_attribute("execution.id", EXECUTION_ID)
                
                try:
                    # Execute the function
                    result = func(*args, **kwargs)
                    # Mark span as successful
                    span.set_status(Status(StatusCode.OK))
                    return result
                except Exception as e:
                    # Mark span as error
                    span.set_status(Status(StatusCode.ERROR, str(e)))
                    span.record_exception(e)
                    raise
        
        return wrapper
    return decorator
