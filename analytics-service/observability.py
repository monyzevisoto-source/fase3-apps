from tracing import configure_traces
import os
import logging

from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.instrumentation.logging.handler import LoggingHandler
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor

from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.botocore import BotocoreInstrumentor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import (
    InMemoryMetricReader,
    PeriodicExportingMetricReader,
)
from opentelemetry.sdk.resources import Resource, SERVICE_NAME


_provider = None
_logger_provider = None
_metric_reader = None


def configure_observability(app, service_name: str):
    global _metric_reader, _provider

    os.environ.setdefault("OTEL_SEMCONV_STABILITY_OPT_IN", "http")
    if _provider is None:
        resource = Resource(
            attributes={SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", service_name)}
        )
        if os.getenv("OTEL_TESTING") == "1":
            _metric_reader = InMemoryMetricReader()
        else:
            _metric_reader = PeriodicExportingMetricReader(OTLPMetricExporter())
        _provider = MeterProvider(resource=resource, metric_readers=[_metric_reader])
        metrics.set_meter_provider(_provider)

    configure_logs(Resource({SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", service_name)}))

    tracer_provider = configure_traces(Resource.create({SERVICE_NAME: os.getenv("OTEL_SERVICE_NAME", service_name)}))
    if not BotocoreInstrumentor().is_instrumented_by_opentelemetry:
        BotocoreInstrumentor().instrument(tracer_provider=tracer_provider)
    FlaskInstrumentor().instrument_app(app, tracer_provider=tracer_provider)
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
