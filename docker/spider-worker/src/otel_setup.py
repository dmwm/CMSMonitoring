#!/usr/bin/env python
"""
OpenTelemetry setup for sending metrics to CERN monitoring endpoint.
"""

import base64
import logging
import contextvars
import opentelemetry
import sys
from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
from opentelemetry._logs import set_logger_provider
from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.metrics import Meter

from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.resources import Resource

import constants as const

# Context variable to store additional metric attributes during function execution
_metric_attributes = contextvars.ContextVar('metric_attributes', default={})

# Global logger provider for updating execution_id
_logger_provider = None
_otel_handler = None


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


def update_execution_id(query_execution_id: str) -> None:
    """
    Update the execution_id in OpenTelemetry logging to link worker logs 
    back to the query cronjob that triggered them.
    
    Args:
        query_execution_id: Execution ID from the query cronjob (extracted from NATS message headers)
    """
    global _logger_provider, _otel_handler
    
    if not query_execution_id:
        return
    
    # Update the constant so it's used in future resource creation
    const.EXECUTION_ID = query_execution_id
    
    # Create new resource with updated execution_id
    # Resource.create() automatically merges with default resource and OTELResourceDetector
    # Ensure values are not None to prevent service.name from being None
    logging_resource = Resource.create({
        "service.name": "spider-worker",
        "service.version": "test",
        "execution.id": query_execution_id,
    })
    
    # Create new logger provider with updated resource
    new_logger_provider = LoggerProvider(resource=logging_resource)
    set_logger_provider(new_logger_provider)
    
    # Get existing exporter configuration
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
        endpoint=f"{const.OTEL_ENDPOINT}/v1/logs",
        insecure=True,
        headers=otel_headers,
        channel_options=channel_options,
    )
    
    new_logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(exporter)
    )
    
    # Remove old handler and add new one
    logger = logging.getLogger()
    if _otel_handler:
        logger.removeHandler(_otel_handler)
    
    # Use custom handler that adds service.name to log attributes (highest precedence)
    service_name = const.IMAGE_NAME or "spider-worker"
    service_version = const.DOCKER_TAG or "unknown"
    new_otel_handler = CustomLoggingHandler(
        service_name=service_name,
        service_version=service_version,
        logger_provider=new_logger_provider,
        level=logging.INFO
    )
    logger.addHandler(new_otel_handler)
    
    # Update globals
    _logger_provider = new_logger_provider
    _otel_handler = new_otel_handler
    
    global_logger.info(
        "Updated execution_id to query cronjob execution_id: %s", query_execution_id
    )


def set_up_logging(log_level: int = logging.INFO) -> logging.Logger:
    """Configure global logging and attach it to OpenTelemetry"""
    
    # Resource.create() automatically merges with default resource and OTELResourceDetector
    # Ensure values are not None to prevent service.name from being None
    logging_resource = Resource.create({
        "service.name": const.IMAGE_NAME or "spider-worker",
        "service.version": const.DOCKER_TAG or "unknown",
        "execution.id": const.EXECUTION_ID,
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

    # The same file is imported as both 'otel_setup'
    # (from src.queues) and 'src.otel_setup' (from main), so set_up_logging runs
    # twice and would add duplicate handlers.
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


def setup_opentelemetry() -> Meter:
    """
    Initialize OpenTelemetry metrics exporter to send metrics to CERN monitoring endpoint.
    
    Returns:
        Meter: The configured meter instance for creating metrics
    """
    # OTLP endpoint configuration
    # For gRPC, endpoint should be host:port without protocol scheme
    otlp_endpoint = const.OTEL_ENDPOINT

    
    # Service name and version
    service_name = const.IMAGE_NAME
    service_version = const.OTEL_SERVICE_NAME
    
    
    # Create resource with service information and run ID
    # Resource.create() automatically merges with default resource and OTELResourceDetector
    # Ensure values are not None to prevent service.name from being None
    resource = Resource.create({
        "service.name": service_name or "spider-worker",
        "service.version": service_version or "unknown",
        "execution.id": const.EXECUTION_ID,
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
    logger = set_up_logging()
    logger.warning(
        f"OpenTelemetry metrics initialized: endpoint={otlp_endpoint}, "
        f"service={service_name}, version={service_version}, execution_id={const.EXECUTION_ID}"
    )
    
    return meter, logger


global_meter, global_logger = setup_opentelemetry()
