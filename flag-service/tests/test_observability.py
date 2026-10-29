import requests
from test_flags import load_app


def _metric_points(metric_reader):
    data = metric_reader.get_metrics_data()
    return [metric for scope in data.resource_metrics for sm in scope.scope_metrics for metric in sm.metrics]


def _attributes(point):
    return dict(point.attributes or {})


def test_http_metrics_include_service_name_route_and_status(monkeypatch):
    module = load_app(monkeypatch)
    client = module.app.test_client()

    assert client.get("/health").status_code == 200
    assert client.get("/flags").status_code == 401

    metrics = _metric_points(module.telemetry_reader)
    assert metrics
    assert any(metric.name == "http.server.request.duration" for metric in metrics)

    data_points = [point for metric in metrics for point in metric.data.data_points]
    assert data_points
    assert any(
        _attributes(point).get("http.response.status_code") == 200
        for point in data_points
    )
    assert any(
        _attributes(point).get("http.response.status_code") == 401
        for point in data_points
    )
    assert any(
        _attributes(point).get("http.route") == "/health"
        for point in data_points
    )


def test_http_metrics_include_server_error_status(monkeypatch):
    module = load_app(monkeypatch)
    monkeypatch.setattr(
        module.requests,
        "get",
        lambda *args, **kwargs: type("Response", (), {"status_code": 200})(),
    )
    monkeypatch.setattr(
        module.pool,
        "getconn",
        lambda: (_ for _ in ()).throw(RuntimeError("database unavailable")),
    )

    response = module.app.test_client().get(
        "/flags", headers={"Authorization": "Bearer unit-key"}
    )

    assert response.status_code == 500
    data_points = [
        point
        for metric in _metric_points(module.telemetry_reader)
        for point in metric.data.data_points
    ]
    assert any(
        _attributes(point).get("http.response.status_code") == 500
        for point in data_points
    )

    resource_attributes = dict(
        module.telemetry_reader.get_metrics_data().resource_metrics[0].resource.attributes
    )
    assert resource_attributes["service.name"] == "flag-service"


def test_http_metrics_do_not_capture_sensitive_attributes(monkeypatch):
    module = load_app(monkeypatch)
    monkeypatch.setattr(
        module.requests,
        "get",
        lambda *args, **kwargs: type("Response", (), {"status_code": 200})(),
    )
    response = module.app.test_client().get(
        "/flags/user@example.com?token=secret-token",
        headers={
            "Authorization": "Bearer secret-token",
            "X-Request-ID": "request-id-123",
        },
    )

    assert response.status_code == 404
    metrics = _metric_points(module.telemetry_reader)
    forbidden = ("authorization", "secret-token", "request-id", "user", "email")
    serialized = repr([_attributes(point) for metric in metrics for point in metric.data.data_points]).lower()
    assert not any(value in serialized for value in forbidden)


def test_requests_client_instrumentation_preserves_authorization_header(monkeypatch):
    received = {}

    def fake_send(self, request, **_kwargs):
        received["authorization"] = request.headers.get("Authorization")
        response = requests.Response()
        response.status_code = 200
        response.url = request.url
        response.request = request
        return response

    monkeypatch.setattr(requests.sessions.Session, "send", fake_send)
    module = load_app(monkeypatch)
    response = module.requests.get(
        "http://auth.test/validate",
        headers={"Authorization": "Bearer secret-token"},
        timeout=3,
    )

    assert response.status_code == 200
    assert received["authorization"] == "Bearer secret-token"
