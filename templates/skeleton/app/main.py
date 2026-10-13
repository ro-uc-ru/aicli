"""FastAPI application entry point."""

from fastapi import FastAPI

from app.api.router import router
from app.config import settings
from app.errors import register_error_handlers
from app.logging import configure_logging

configure_logging(settings.LOG_LEVEL)

app = FastAPI(title=settings.APP_NAME)
register_error_handlers(app)
app.include_router(router)
