"""HTTP helpers shared by Film routes."""

from __future__ import annotations

from fastapi import HTTPException, Request


def resolve_locale(request: Request, explicit: str = "") -> str:
    if (explicit or "").strip():
        return explicit.strip()
    al = (request.headers.get("accept-language") or "").lower()
    if al.startswith("zh"):
        return "zh"
    return "en"


def http_error_from_task(exc: BaseException) -> HTTPException:
    msg = str(exc) or exc.__class__.__name__
    status = 400 if isinstance(exc, ValueError) else 500
    if "not installed" in msg.lower() or "not configured" in msg.lower():
        status = 503
    return HTTPException(status_code=status, detail=msg)
