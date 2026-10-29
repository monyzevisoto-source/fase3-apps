import os

import pytest
from opentelemetry import metrics
from opentelemetry.exporter.otlp.proto.http.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.sdk.resources import Resource, SERVICE_NAME


@pytest.mark.otel_integration
def test_otlp_metrics_exporter_reaches_configured_collector():
    endpoint = os.getenv("OTEL_TEST_OTLP_METRICS_ENDPOINT")
    if not endpoint:
        pytest.skip("OTEL_TEST_OTLP_METRICS_ENDPOINT não configurado")

    reader = PeriodicExportingMetricReader(
        OTLPMetricExporter(endpoint=endpoint), export_interval_millis=50
    )
    provider = MeterProvider(
        resource=Resource.create({SERVICE_NAME: "flag-service"}),
        metric_readers=[reader],
    )
    meter = provider.get_meter("flag-service.integration-test")
    meter.create_counter("flag_service_integration_requests").add(1)

    assert provider.force_flush()
    provider.shutdown()
