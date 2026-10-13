"""Error response schema."""

from pydantic import BaseModel


class ErrorModel(BaseModel):
    """Body of every error response."""

    status: int
    message: str
