#!/usr/bin/env python
"""
OpenTelemetry setup for CMS Monitoring Python cronjobs.

Sends metrics, traces, and logs to the in-cluster OpenTelemetry Collector
(opentelemetry-collector.opentelemetry.svc.cluster.local:4317 by default).

Enable by setting OTEL_ENABLED=true and OTEL_SERVICE_NAME in the workload env.
Child images built on cmsmon-py can import:

    from helpers.otel_setup import global_meter, global_tracer, trace_span, setup_opentelemetry
"""

from __future__ import annotations

import atexit
import base64
import functools
import logging
import os
import sys
import uuid
from typing import Callable, Optional, Tuple, TypeVar

import opentelemetry
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
from opentelemetry.trace import Status, StatusCode, Tracer

logger = logging.getLogger(__name__)

F = TypeVar("F", bound=Callable)

DEFAULT_COLLECTOR_ENDPOINT = (
    "opentelemetry-collector.opentelemetry.svc.cluster.local:4317"
)

EXECUTION_ID = str(uuid.uuid4())

_otel_initialized = False

_STOMP_LOGGERS = ("stomp.py", "StompAMQ", "StompyListener", "stomp")

DEFAULT_LOG_FORMAT = "%(asctime)s [%(levelname)s] %(message)s"

_logging_configured = False


def configure_stomp_log_levels() -> None:
    """Set stomp loggers to WARNING. Safe to call again after StompAMQ init."""
    for logger_name in _STOMP_LOGGERS:
        logging.getLogger(logger_name).setLevel(logging.WARNING)


def _has_stdout_handler(root_logger: logging.Logger) -> bool:
    return any(
        isinstance(handler, logging.StreamHandler)
        and getattr(handler, "stream", None) is sys.stdout
        for handler in root_logger.handlers
    )


def _retarget_stderr_to_stdout(root_logger: logging.Logger, log_level: int) -> bool:
    """basicConfig() adds a stderr handler; point it at stdout for kubectl logs."""
    for handler in root_logger.handlers:
        if (
            isinstance(handler, logging.StreamHandler)
            and getattr(handler, "stream", None) is sys.stderr
        ):
            handler.stream = sys.stdout
            if handler.formatter is None:
                handler.setFormatter(logging.Formatter(DEFAULT_LOG_FORMAT))
            handler.setLevel(log_level)
            return True
    return False


def configure_logging() -> None:
    """Attach a stdout handler to the root logger (idempotent).

    Child scripts should import helpers.otel_setup before calling logging.getLogger()
    and should not call logging.basicConfig() — stdout is configured here, and OTLP
    export is added when OTEL_ENABLED and OTEL_LOGS_ENABLED are true.
    """
    global _logging_configured

    configure_stomp_log_levels()

    if _logging_configured:
        return

    config = otel_config()
    log_level = getattr(logging, config["log_level"], logging.INFO)
    root_logger = logging.getLogger()

    if not _has_stdout_handler(root_logger):
        if not _retarget_stderr_to_stdout(root_logger, log_level):
            stdout_handler = logging.StreamHandler(sys.stdout)
            stdout_handler.setFormatter(logging.Formatter(DEFAULT_LOG_FORMAT))
            stdout_handler.setLevel(log_level)
            root_logger.addHandler(stdout_handler)

    root_logger.setLevel(log_level)
    _logging_configured = True


def _env_bool(name: str, default: bool = False) -> bool:
    value = os.getenv(name)
    if value is None:
        return default
    return value.lower() in {"1", "true", "yes", "on"}


def otel_config() -> dict:
    """Return OpenTelemetry settings from environment variables."""
    return {
        "enabled": _env_bool("OTEL_ENABLED", default=False),
        "endpoint": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", DEFAULT_COLLECTOR_ENDPOINT),
        "metric_export_interval_ms": int(
            os.getenv("OTEL_METRIC_EXPORT_INTERVAL", "60000")
        ),
        "service_name": os.getenv("OTEL_SERVICE_NAME", "cmsmon-py"),
        "service_version": os.getenv(
            "OTEL_SERVICE_VERSION",
            os.getenv("CMSMON_TAG", "unknown"),
        ),
        "username": os.getenv("OPENTELEMETRY_USERNAME"),
        "password": os.getenv("OPENTELEMETRY_PASSWORD"),
        "metrics_enabled": _env_bool("OTEL_METRICS_ENABLED", default=True),
        "traces_enabled": _env_bool("OTEL_TRACES_ENABLED", default=True),
        "logs_enabled": _env_bool("OTEL_LOGS_ENABLED", default=True),
        "log_level": os.getenv("OTEL_LOG_LEVEL", "INFO").upper(),
        "export_timeout_sec": int(os.getenv("OTEL_EXPORTER_OTLP_TIMEOUT", "10")),
        "insecure": _env_bool("OTEL_EXPORTER_OTLP_INSECURE", default=True),
    }


def _grpc_channel_options() -> tuple:
    return (
        ("grpc.keepalive_time_ms", 20000),
        ("grpc.keepalive_timeout_ms", 10000),
        ("grpc.keepalive_permit_without_calls", False),
        ("grpc.http2.min_time_between_pings_ms", 30000),
    )


def _grpc_endpoint(endpoint: str) -> str:
    """Return host:port for OTLP/gRPC exporters.

    Some configs append /v1/{signal} to OTEL_EXPORTER_OTLP_ENDPOINT.
    gRPC exporters expect host:port only; the SDK adds signal routing internally.
    """
    endpoint = endpoint.rstrip("/")
    for suffix in ("/v1/logs", "/v1/traces", "/v1/metrics"):
        if endpoint.endswith(suffix):
            endpoint = endpoint[: -len(suffix)]
    return endpoint.rstrip("/")


def _build_otlp_headers(config: dict) -> Optional[dict]:
    username = config["username"]
    password = config["password"]
    if not username or not password:
        return None
    creds = f"{username}:{password}".encode("utf-8")
    token = base64.b64encode(creds).decode("utf-8")
    return {"authorization": f"Basic {token}"}


def setup_opentelemetry() -> Tuple[Meter, Tracer]:
    """
    Initialize OpenTelemetry exporters for metrics, traces, and logs.

    Returns no-op meter and tracer instances when OTEL_ENABLED is false.
    """
    global _otel_initialized

    config = otel_config()
    meter = opentelemetry.metrics.get_meter(__name__)
    tracer = opentelemetry.trace.get_tracer(__name__)

    configure_logging()

    if not config["enabled"]:
        return meter, tracer

    if _otel_initialized:
        return meter, tracer

    resource = Resource.create(
        {
            "service.name": config["service_name"],
            "service.version": config["service_version"],
            "execution.id": EXECUTION_ID,
        }
    )

    grpc_endpoint = _grpc_endpoint(config["endpoint"])
    exporter_kwargs = {
        "headers": _build_otlp_headers(config),
        "channel_options": _grpc_channel_options(),
        "insecure": config["insecure"],
        "timeout": config["export_timeout_sec"],
    }

    if config["metrics_enabled"]:
        metric_exporter = OTLPMetricExporter(
            endpoint=grpc_endpoint,
            **exporter_kwargs,
        )
        metric_reader = PeriodicExportingMetricReader(
            exporter=metric_exporter,
            export_interval_millis=config["metric_export_interval_ms"],
        )
        provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
        opentelemetry.metrics.set_meter_provider(provider)
        meter = opentelemetry.metrics.get_meter(__name__)

    if config["traces_enabled"]:
        trace_exporter = OTLPSpanExporter(
            endpoint=grpc_endpoint,
            **exporter_kwargs,
        )
        trace_provider = TracerProvider(resource=resource)
        trace_provider.add_span_processor(BatchSpanProcessor(trace_exporter))
        opentelemetry.trace.set_tracer_provider(trace_provider)
        tracer = opentelemetry.trace.get_tracer(__name__)

    if config["logs_enabled"]:
        logger_provider = LoggerProvider(resource=resource)
        set_logger_provider(logger_provider)

        log_exporter = OTLPLogExporter(
            endpoint=grpc_endpoint,
            **exporter_kwargs,
        )
        logger_provider.add_log_record_processor(BatchLogRecordProcessor(log_exporter))

        log_level = getattr(logging, config["log_level"], logging.INFO)
        root_logger = logging.getLogger()
        if not any(isinstance(h, LoggingHandler) for h in root_logger.handlers):
            root_logger.addHandler(
                LoggingHandler(logger_provider=logger_provider, level=log_level)
            )

        atexit.register(_shutdown_on_exit)

    _otel_initialized = True
    logger.warning(
        "OpenTelemetry initialized: endpoint=%s, service=%s, version=%s, execution_id=%s",
        grpc_endpoint,
        config["service_name"],
        config["service_version"],
        EXECUTION_ID,
    )

    return meter, tracer


def shutdown_opentelemetry(timeout_millis: int = 30000) -> None:
    """Flush and shut down OpenTelemetry providers (call before short-lived jobs exit)."""
    if not _otel_initialized:
        return

    config = otel_config()
    if config["logs_enabled"]:
        from opentelemetry._logs import get_logger_provider

        provider = get_logger_provider()
        if provider is not None:
            provider.force_flush(timeout_millis=timeout_millis)
            provider.shutdown()

    if config["metrics_enabled"]:
        from opentelemetry.metrics import get_meter_provider

        provider = get_meter_provider()
        if provider is not None:
            provider.force_flush(timeout_millis=timeout_millis)
            provider.shutdown()

    if config["traces_enabled"]:
        from opentelemetry.trace import get_tracer_provider

        provider = get_tracer_provider()
        if provider is not None:
            provider.force_flush(timeout_millis=timeout_millis)
            provider.shutdown()


def _shutdown_on_exit() -> None:
    shutdown_opentelemetry()


def trace_span(span_name: Optional[str] = None, **attributes: object) -> Callable[[F], F]:
    """
    Decorator that creates an OpenTelemetry span around a function call.

    When OTEL is disabled, the wrapped function runs unchanged.
    """

    def decorator(func: F) -> F:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            if not otel_config()["enabled"]:
                return func(*args, **kwargs)

            config = otel_config()
            name = span_name or f"{config['service_name']}.{func.__module__}.{func.__name__}"

            with global_tracer.start_as_current_span(name) as span:
                for key, value in attributes.items():
                    span.set_attribute(key, value)
                try:
                    result = func(*args, **kwargs)
                    span.set_status(Status(StatusCode.OK))
                    return result
                except Exception as exc:
                    span.set_status(Status(StatusCode.ERROR, str(exc)))
                    span.record_exception(exc)
                    raise

        return wrapper  # type: ignore[return-value]

    return decorator


global_meter, global_tracer = setup_opentelemetry()
