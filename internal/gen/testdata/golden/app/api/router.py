"""Aggregates the routers of the app."""

from fastapi import APIRouter

from app.api.routers import health

# isort: off
# aicli:router-imports
# isort: on

router = APIRouter()
router.include_router(health.router)
# aicli:routers
