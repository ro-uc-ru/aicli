"""Tests of the error responses."""

from fastapi.testclient import TestClient

from app.main import app


@app.get("/__boom")
def boom() -> None:
    """Route that always fails, to exercise the unhandled exception handler."""
    raise RuntimeError("secret detail")


def test_unknown_route_returns_404_error_model(client: TestClient) -> None:
    """An unknown route returns a 404 ErrorModel."""
    response = client.get("/does-not-exist")
    assert response.status_code == 404
    assert response.json() == {"status": 404, "message": "Not Found"}


def test_unhandled_exception_returns_500_error_model(client: TestClient) -> None:
    """An unhandled exception returns a 500 ErrorModel without details."""
    response = client.get("/__boom")
    assert response.status_code == 500
    assert response.json() == {"status": 500, "message": "Internal Server Error"}
    assert "secret detail" not in response.text
