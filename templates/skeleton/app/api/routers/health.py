"""Health check route."""

from fastapi import APIRouter

from app.schemas.health import HealthModel

router = APIRouter()


@router.get("/health", response_model=HealthModel)
def health() -> HealthModel:
    """Report that the app is up."""
    return HealthModel(status=200)
