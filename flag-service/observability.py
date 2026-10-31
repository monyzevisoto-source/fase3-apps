import os
import logging

from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.instrumentation.logging.handler import LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from typing import Optional

from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import (
    InMemoryMetricReader,
    MetricReader,
    PeriodicExportingMetricReader,
)
from opentelemetry.sdk.resources import Resource, SERVICE_NAME


SERVICE_NAME_VALUE = "flag-service"
_provider = None
_logger_provider = None
_metric_reader = None
_instrumented_requests = False


def configure_observability(app, metric_reader: Optional[MetricReader] = None):
    """Configure HTTP metrics and return the configured metric reader.

    A reader can be injected by tests so they can inspect metrics without a
    running Collector. Production uses the standard OTLP environment settings.
    """
    global _metric_reader, _provider, _instrumented_requests

    os.environ.setdefault("OTEL_SEMCONV_STABILITY_OPT_IN", "http")

    if _provider is None:
        resource = Resource.create(
            attributes={
                SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", SERVICE_NAME_VALUE)
            }
        )

        if metric_reader is None:
            if os.getenv("OTEL_TESTING") == "1":
                metric_reader = InMemoryMetricReader()
            else:
                metric_reader = PeriodicExportingMetricReader(OTLPMetricExporter())

        _metric_reader = metric_reader
        _provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
        metrics.set_meter_provider(_provider)

    configure_logs(Resource({SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", SERVICE_NAME_VALUE)}))

    FlaskInstrumentor().instrument_app(app)
    if not _instrumented_requests:
        RequestsInstrumentor().instrument()
        _instrumented_requests = True

    return _provider, _metric_reader


def configure_logs(resource, exporter=None):
    """Bridge application logging to OTLP, keeping existing console handlers."""
    global _logger_provider
    if _logger_provider is not None:
        return _logger_provider
    if os.getenv("OTEL_TESTING") == "1" and exporter is None:
        return None
    if os.getenv("OTEL_LOGS_EXPORTER", "otlp") == "none":
        return None

    provider = LoggerProvider(resource=resource)
    provider.add_log_record_processor(
        BatchLogRecordProcessor(exporter if exporter is not None else OTLPLogExporter())
    )
    handler = LoggingHandler(level=logging.NOTSET, logger_provider=provider)
    # Exporter diagnostics must stay on the console to avoid recursive exports.
    handler.addFilter(lambda record: not record.name.startswith("opentelemetry"))
    logging.getLogger().addHandler(handler)
    _logger_provider = provider
    return provider
