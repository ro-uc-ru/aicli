"""Tests of the settings and the logging setup."""

import logging

import pytest
from pydantic import ValidationError

from app.config import Settings, settings
from app.logging import configure_logging
from app.main import app


@pytest.fixture(autouse=True)
def clean_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Remove the app settings from the environment of each test."""
    for name in ("APP_NAME", "DATABASE_URL", "LOG_LEVEL"):
        monkeypatch.delenv(name, raising=False)


def test_default_database_url() -> None:
    """DATABASE_URL defaults to a local SQLite file."""
    assert Settings(_env_file=None).DATABASE_URL == "sqlite:///./app.db"


def test_app_name_is_the_app_title() -> None:
    """APP_NAME is the title of the FastAPI app."""
    assert app.openapi()["info"]["title"] == settings.APP_NAME


def test_invalid_log_level_fails() -> None:
    """An invalid LOG_LEVEL fails validation naming the setting."""
    with pytest.raises(ValidationError, match="LOG_LEVEL"):
        Settings(_env_file=None, LOG_LEVEL="LOUD")


def test_logging_is_configured() -> None:
    """configure_logging sets the level of the root logger."""
    root = logging.getLogger()
    previous = root.level
    try:
        configure_logging("WARNING")
        assert root.level == logging.WARNING
    finally:
        root.setLevel(previous)
