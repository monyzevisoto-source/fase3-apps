from test_targeting import load_app


def test_http_metrics_identify_service_status_and_normalized_route(monkeypatch):
    module = load_app(monkeypatch)
    client = module.app.test_client()

    assert client.get("/health").status_code == 200
    assert client.get("/rules/customer@example.com?token=secret").status_code == 401

    data = module.telemetry_reader.get_metrics_data()
    metrics = [metric for rm in data.resource_metrics for sm in rm.scope_metrics for metric in sm.metrics]
    points = [point for metric in metrics for point in metric.data.data_points]
    attributes = [dict(point.attributes or {}) for point in points]

    assert any(metric.name == "http.server.request.duration" for metric in metrics)
    assert any(item.get("http.response.status_code") == 200 for item in attributes)
    assert any(item.get("http.response.status_code") == 401 for item in attributes)
    assert any(item.get("http.route") == "/health" for item in attributes)
    assert data.resource_metrics[0].resource.attributes["service.name"] == "targeting-service"
    assert "customer@example.com" not in repr(attributes)
    assert "secret" not in repr(attributes)


def test_http_metrics_capture_internal_server_error(monkeypatch):
    module = load_app(monkeypatch)
    monkeypatch.setattr(module.requests, "get", lambda *args, **kwargs: type("R", (), {"status_code": 200})())
    monkeypatch.setattr(module.pool, "getconn", lambda: (_ for _ in ()).throw(RuntimeError("db down")))

    response = module.app.test_client().get("/rules/checkout", headers={"Authorization": "Bearer key"})
    assert response.status_code == 500

    data = module.telemetry_reader.get_metrics_data()
    points = [point for rm in data.resource_metrics for sm in rm.scope_metrics for metric in sm.metrics for point in metric.data.data_points]
    assert any(dict(point.attributes or {}).get("http.response.status_code") == 500 for point in points)
