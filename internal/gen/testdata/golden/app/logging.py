"""Standard-library logging setup."""

import logging

LOG_FORMAT = "%(asctime)s %(levelname)s %(name)s: %(message)s"


def configure_logging(level: str) -> None:
    """Configure the root logger with the fixed format at the given level."""
    logging.basicConfig(level=level, format=LOG_FORMAT, force=True)
