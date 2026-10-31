"""OTLP/HTTP tracing shared by this service's HTTP and dependency instrumentation."""
import os

from opentelemetry import trace, propagate
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.trace.sampling import ALWAYS_OFF
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

_provider = None


def configure_traces(resource):
    global _provider
    if _provider is None:
        options = {"resource": resource}
        if os.getenv("OTEL_TRACES_EXPORTER") == "none":
            options["sampler"] = ALWAYS_OFF
        _provider = TracerProvider(**options)
        if os.getenv("OTEL_TESTING") != "1" and os.getenv("OTEL_TRACES_EXPORTER") != "none":
            _provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))
        trace.set_tracer_provider(_provider)
        propagate.set_global_textmap(TraceContextTextMapPropagator())
    return _provider
