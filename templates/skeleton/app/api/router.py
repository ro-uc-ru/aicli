"""Aggregates the routers of the app."""

from fastapi import APIRouter

from app.api.routers import health

router = APIRouter()
router.include_router(health.router)
