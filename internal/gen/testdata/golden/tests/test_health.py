"""Tests of the health check route."""

from fastapi.testclient import TestClient


def test_health(client: TestClient) -> None:
    """GET /health returns 200 and a HealthModel body."""
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": 200}
