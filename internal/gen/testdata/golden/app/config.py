"""Application settings read from environment variables and the .env file."""

from typing import Literal

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Runtime configuration of the app."""

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    APP_NAME: str = "myapp"
    DATABASE_URL: str = "sqlite:///./app.db"
    LOG_LEVEL: Literal["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"] = "INFO"


settings = Settings()
