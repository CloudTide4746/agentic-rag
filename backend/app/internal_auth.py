"""Authentication for the private Agent API used by the Go backend."""

from secrets import compare_digest

from fastapi import Header, HTTPException, status

from .config import settings


def require_internal_token(x_agent_token: str | None = Header(default=None)) -> None:
    """Reject anonymous callers before any Agent or database work is performed."""
    expected = settings.internal_api_token
    if not expected or not x_agent_token or not compare_digest(x_agent_token, expected):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="内部服务凭据无效")
