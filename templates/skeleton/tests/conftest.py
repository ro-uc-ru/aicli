"""Shared pytest fixtures."""

import pytest
from fastapi.testclient import TestClient

from app.main import app


@pytest.fixture
def client() -> TestClient:
    """Test client that returns 500 responses instead of raising."""
    return TestClient(app, raise_server_exceptions=False)
