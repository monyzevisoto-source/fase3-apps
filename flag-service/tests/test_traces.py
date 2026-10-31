from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter
from test_flags import load_app
from tracing import configure_traces
from opentelemetry.sdk.resources import Resource


def test_http_trace_continues_incoming_context(monkeypatch):
    module = load_app(monkeypatch)
    exporter = InMemorySpanExporter()
    provider = configure_traces(Resource.create({}))
    provider.add_span_processor(SimpleSpanProcessor(exporter))
    response = module.app.test_client().get(
        "/flags", headers={"traceparent": "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01"}
    )
    assert response.status_code == 401
    spans = exporter.get_finished_spans()
    assert spans
    server = next(span for span in spans if span.kind.name == "SERVER")
    assert server.context.trace_id == int("0123456789abcdef0123456789abcdef", 16)
    assert server.parent.span_id == int("0123456789abcdef", 16)
    assert "authorization" not in repr(server.attributes).lower()
