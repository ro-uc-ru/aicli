"""Global error handlers: every error response is an ErrorModel."""

import logging
from http import HTTPStatus

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

from app.schemas.errors import ErrorModel

logger = logging.getLogger(__name__)

MESSAGES = {
    404: "Not Found",
    405: "Method Not Allowed",
    422: "Unprocessable Entity",
    500: "Internal Server Error",
}


def message_for(status: int) -> str:
    """Return the fixed message of a status code."""
    if status in MESSAGES:
        return MESSAGES[status]
    try:
        return HTTPStatus(status).phrase
    except ValueError:
        return "Error"


def error_response(status: int) -> JSONResponse:
    """Build the JSON error response of a status code."""
    body = ErrorModel(status=status, message=message_for(status))
    return JSONResponse(status_code=status, content=body.model_dump())


async def http_exception_handler(
    request: Request, exc: StarletteHTTPException
) -> JSONResponse:
    """Handle HTTP errors such as 404 and 405."""
    return error_response(exc.status_code)


async def validation_exception_handler(
    request: Request, exc: RequestValidationError
) -> JSONResponse:
    """Handle request validation errors."""
    return error_response(422)


async def unhandled_exception_handler(request: Request, exc: Exception) -> JSONResponse:
    """Handle unhandled exceptions, logging only the exception type name."""
    logger.error("Unhandled exception: %s", type(exc).__name__)
    return error_response(500)


def register_error_handlers(app: FastAPI) -> None:
    """Register the error handlers on the app."""
    app.add_exception_handler(StarletteHTTPException, http_exception_handler)
    app.add_exception_handler(RequestValidationError, validation_exception_handler)
    app.add_exception_handler(Exception, unhandled_exception_handler)
