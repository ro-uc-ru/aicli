"""Health check schema."""

from pydantic import BaseModel


class HealthModel(BaseModel):
    """Body of the health check response."""

    status: int
