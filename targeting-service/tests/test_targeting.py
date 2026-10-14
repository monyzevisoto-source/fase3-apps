import importlib.util
import pathlib
import sys

import psycopg2.pool


class FakeCursor:
    def __init__(self, rows=None):
        self.rows = rows or []
        self.rowcount = len(self.rows)
        self.executed = []

    def execute(self, query, params=()):
        self.executed.append((query, params))

    def fetchone(self):
        return self.rows[0] if self.rows else None

    def fetchall(self):
        return self.rows

    def close(self):
        pass


class FakeConnection:
    def __init__(self, cursor):
        self.cursor_instance = cursor

    def cursor(self, **kwargs):
        return self.cursor_instance

    def commit(self):
        pass

    def rollback(self):
        pass


class FakePool:
    def __init__(self, cursor):
        self.connection = FakeConnection(cursor)

    def getconn(self):
        return self.connection

    def putconn(self, connection):
        pass


def load_app(monkeypatch, cursor=None):
    monkeypatch.setenv("DATABASE_URL", "postgresql://unit-test")
    monkeypatch.setenv("AUTH_SERVICE_URL", "http://auth")
    monkeypatch.setattr(
        psycopg2.pool,
        "SimpleConnectionPool",
        lambda *args, **kwargs: FakePool(cursor or FakeCursor()),
    )
    path = pathlib.Path(__file__).parents[1] / "app.py"
    spec = importlib.util.spec_from_file_location("targeting_service_app", path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_health_endpoint(monkeypatch):
    module = load_app(monkeypatch)
    response = module.app.test_client().get("/health")

    assert response.status_code == 200
    assert response.get_json() == {"status": "ok"}


def test_create_rule_validates_required_fields_before_database(monkeypatch):
    module = load_app(monkeypatch)
    monkeypatch.setattr(module.requests, "get", lambda *args, **kwargs: type("R", (), {"status_code": 200})())

    response = module.app.test_client().post(
        "/rules", json={"flag_name": "checkout"}, headers={"Authorization": "Bearer unit-key"}
    )

    assert response.status_code == 400
    assert "rules" in response.get_json()["error"]


def test_get_rule_returns_not_found(monkeypatch):
    module = load_app(monkeypatch, FakeCursor())
    monkeypatch.setattr(module.requests, "get", lambda *args, **kwargs: type("R", (), {"status_code": 200})())

    response = module.app.test_client().get(
        "/rules/checkout", headers={"Authorization": "Bearer unit-key"}
    )

    assert response.status_code == 404
    assert response.get_json() == {"error": "Regra não encontrada"}
