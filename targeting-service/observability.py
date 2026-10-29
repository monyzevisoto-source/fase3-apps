import os

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


_provider = None
_metric_reader = None
_instrumented_requests = False


def configure_observability(app, service_name: str):
    global _metric_reader, _provider, _instrumented_requests

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

    FlaskInstrumentor().instrument_app(app)
    if not _instrumented_requests:
        RequestsInstrumentor().instrument()
        _instrumented_requests = True
    return _provider, _metric_reader
