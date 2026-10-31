import logging
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

from opentelemetry.proto.collector.logs.v1.logs_service_pb2 import ExportLogsServiceRequest
from opentelemetry.sdk.resources import Resource
from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter

import observability


def test_logging_is_sent_as_otlp_protobuf_without_duplicate_handlers(monkeypatch):
    requests = []

    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            body = self.rfile.read(int(self.headers["Content-Length"]))
            requests.append((self.path, ExportLogsServiceRequest.FromString(body)))
            self.send_response(200)
            self.send_header("Content-Type", "application/x-protobuf")
            self.end_headers()

        def log_message(self, *args):
            pass

    server = HTTPServer(("127.0.0.1", 0), Receiver)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    root = logging.getLogger()
    original_handlers = list(root.handlers)
    monkeypatch.setattr(observability, "_logger_provider", None)
    monkeypatch.setenv("OTEL_LOGS_EXPORTER", "otlp")
    monkeypatch.setenv("OTEL_EXPORTER_OTLP_ENDPOINT", f"http://127.0.0.1:{server.server_port}")
    provider = None
    try:
        provider = observability.configure_logs(
            Resource.create({"service.name": "logs-integration"}), OTLPLogExporter()
        )
        assert observability.configure_logs(Resource.create({})) is provider
        assert len(root.handlers) == len(original_handlers) + 1
        logger = logging.getLogger("application.integration")
        logger.warning("otlp-logs-test")
        logging.getLogger("opentelemetry.exporter.test").warning("must-not-reexport")
        assert provider.force_flush()
        assert requests and all(path == "/v1/logs" for path, _ in requests)
        resources = [rl for _, req in requests for rl in req.resource_logs]
        records = [record for rl in resources for scope in rl.scope_logs for record in scope.log_records]
        assert [record.body.string_value for record in records] == ["otlp-logs-test"]
        assert records[0].severity_text == "WARN"
        assert any(a.key == "service.name" and a.value.string_value == "logs-integration"
                   for rl in resources for a in rl.resource.attributes)
    finally:
        if provider is not None:
            provider.shutdown()
        for handler in root.handlers[:]:
            if handler not in original_handlers:
                root.removeHandler(handler)
                handler.close()
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
