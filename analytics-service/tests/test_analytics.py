import importlib.util
import json
import pathlib
import sys

import boto3


class FakeClient:
    def __init__(self):
        self.put_items = []
        self.deleted = []

    def put_item(self, **kwargs):
        self.put_items.append(kwargs)

    def delete_message(self, **kwargs):
        self.deleted.append(kwargs)


class FakeSession:
    def __init__(self, **kwargs):
        self.sqs = FakeClient()
        self.dynamodb = FakeClient()

    def client(self, name):
        return self.sqs if name == "sqs" else self.dynamodb


class FakeThread:
    def __init__(self, *args, **kwargs):
        pass

    def start(self):
        pass


def load_app(monkeypatch):
    monkeypatch.setenv("AWS_REGION", "us-east-1")
    monkeypatch.setenv("AWS_SQS_URL", "https://sqs/unit-test")
    monkeypatch.setenv("AWS_DYNAMODB_TABLE", "analytics")
    monkeypatch.setattr(boto3, "Session", FakeSession)
    monkeypatch.setattr("threading.Thread", FakeThread)
    path = pathlib.Path(__file__).parents[1] / "app.py"
    spec = importlib.util.spec_from_file_location("analytics_service_app", path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_health_endpoint(monkeypatch):
    module = load_app(monkeypatch)
    response = module.app.test_client().get("/health")

    assert response.status_code == 200
    assert response.get_json() == {"status": "ok"}


def test_process_message_persists_and_deletes_valid_event(monkeypatch):
    module = load_app(monkeypatch)
    message = {
        "MessageId": "message-1",
        "ReceiptHandle": "receipt-1",
        "Body": json.dumps(
            {"user_id": "user-1", "flag_name": "checkout", "result": True, "timestamp": "2026-01-01T00:00:00Z"}
        ),
    }

    module.process_message(message)

    assert len(module.dynamodb_client.put_items) == 1
    assert module.dynamodb_client.put_items[0]["Item"]["flag_name"] == {"S": "checkout"}
    assert module.sqs_client.deleted == [{"QueueUrl": "https://sqs/unit-test", "ReceiptHandle": "receipt-1"}]


def test_process_message_keeps_invalid_json_in_queue(monkeypatch):
    module = load_app(monkeypatch)
    module.process_message({"MessageId": "bad", "ReceiptHandle": "receipt", "Body": "not-json"})

    assert module.dynamodb_client.put_items == []
    assert module.sqs_client.deleted == []
