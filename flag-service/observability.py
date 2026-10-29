import os
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

    FlaskInstrumentor().instrument_app(app)
    if not _instrumented_requests:
        RequestsInstrumentor().instrument()
        _instrumented_requests = True

    return _provider, _metric_reader
