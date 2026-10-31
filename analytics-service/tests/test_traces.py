import json

from botocore.stub import Stubber
from opentelemetry import trace
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter
from opentelemetry.trace import StatusCode
from test_analytics import load_app
from tracing import configure_traces


def test_sqs_consumer_continues_go_trace_and_traces_dynamodb(monkeypatch):
    module = load_app(monkeypatch)
    exporter = InMemorySpanExporter()
    configure_traces(Resource.create({})).add_span_processor(SimpleSpanProcessor(exporter))
    # Use real botocore with stubbed transport to verify dependency instrumentation.
    import botocore.session
    client = botocore.session.get_session().create_client(
        "dynamodb", region_name="us-east-1", aws_access_key_id="test", aws_secret_access_key="test"
    )
    stub = Stubber(client)
    stub.add_response("put_item", {})
    monkeypatch.setattr(module, "dynamodb_client", client)
    message = {
        "MessageId": "test", "ReceiptHandle": "receipt",
        "MessageAttributes": {"traceparent": {"StringValue": "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01"}},
        "Body": json.dumps({"user_id": "private-user", "flag_name": "flag", "result": True, "timestamp": "2026-01-01T00:00:00Z"}),
    }
    with stub:
        module.process_message(message)
        stub.assert_no_pending_responses()
    spans = exporter.get_finished_spans()
    consumer = next(span for span in spans if span.kind.name == "CONSUMER")
    assert consumer.context.trace_id == int("0123456789abcdef0123456789abcdef", 16)
    assert consumer.parent.span_id == int("0123456789abcdef", 16)
    dependency = next(span for span in spans if "PutItem" in span.name)
    assert dependency.parent.span_id == consumer.context.span_id
    assert "private-user" not in repr([dict(span.attributes) for span in spans])
    assert module.sqs_client.deleted


def test_invalid_sqs_message_marks_consumer_error_without_ack(monkeypatch):
    module = load_app(monkeypatch)
    exporter = InMemorySpanExporter()
    configure_traces(Resource.create({})).add_span_processor(SimpleSpanProcessor(exporter))
    module.process_message({"MessageId": "bad", "Body": "invalid-json", "ReceiptHandle": "receipt"})
    span = next(span for span in exporter.get_finished_spans() if span.kind.name == "CONSUMER")
    assert span.status.status_code == StatusCode.ERROR
    assert not module.sqs_client.deleted
