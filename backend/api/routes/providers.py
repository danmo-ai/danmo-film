"""Provider status + generation proxy routes."""

from __future__ import annotations

from typing import Any

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field

from backend.api.deps import get_generation_provider, get_llm_chat_provider
from backend.providers.base import GenerationProvider, LlmChatProvider

router = APIRouter(prefix="/api/providers", tags=["providers"])


class ProxyBody(BaseModel):
    body: dict[str, Any] = Field(default_factory=dict)


@router.get("/status")
def providers_status(
    gen: GenerationProvider = Depends(get_generation_provider),
    llm: LlmChatProvider = Depends(get_llm_chat_provider),
):
    gen_health: dict[str, Any]
    try:
        gen_health = gen.health()
    except Exception as exc:
        gen_health = {"ok": False, "error": str(exc)}
    return {
        "generation": {"provider_id": gen.provider_id, "health": gen_health},
        "llm": {"provider_id": llm.provider_id, "available": llm.is_available()},
    }


@router.get("/models")
def list_models(
    media: str | None = None,
    gen: GenerationProvider = Depends(get_generation_provider),
):
    try:
        return {"items": gen.list_models(media=media)}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.post("/proxy/images/generations")
def proxy_image_gen(req: ProxyBody, gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.create_image(req.body)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.post("/proxy/videos/generations")
def proxy_video_gen(req: ProxyBody, gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.create_video(req.body)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.post("/proxy/videos/edits")
def proxy_video_edit(req: ProxyBody, gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.edit_video(req.body)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.post("/proxy/videos/long-generations")
def proxy_video_long(req: ProxyBody, gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.long_generate_video(req.body)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("/proxy/tasks/{task_id}")
def proxy_task(task_id: str, gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.get_task(task_id)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
