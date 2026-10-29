from test_analytics import load_app


def test_health_metrics_identify_analytics_service(monkeypatch):
    module = load_app(monkeypatch)
    assert module.app.test_client().get("/health").status_code == 200

    data = module.telemetry_reader.get_metrics_data()
    metrics = [metric for rm in data.resource_metrics for sm in rm.scope_metrics for metric in sm.metrics]
    points = [point for metric in metrics for point in metric.data.data_points]
    attributes = [dict(point.attributes or {}) for point in points]

    assert any(metric.name == "http.server.request.duration" for metric in metrics)
    assert any(item.get("http.response.status_code") == 200 for item in attributes)
    assert data.resource_metrics[0].resource.attributes["service.name"] == "analytics-service"
