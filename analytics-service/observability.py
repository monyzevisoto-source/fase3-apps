import os

from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import (
    InMemoryMetricReader,
    PeriodicExportingMetricReader,
)
from opentelemetry.sdk.resources import Resource, SERVICE_NAME


_provider = None
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

    FlaskInstrumentor().instrument_app(app)
    return _provider, _metric_reader
