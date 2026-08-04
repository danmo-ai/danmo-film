"""Generation bridge — Make-shaped routes that talk to peer providers and return Film assets.

Film stages ``fas_*`` inputs to the provider for the job only, then ingests result
bytes into the Film asset store. No finals are uploaded back to Make.
"""

from __future__ import annotations

import json
from typing import Any

from fastapi import APIRouter, Depends, HTTPException, Request
from fastapi.responses import StreamingResponse

from backend.api.deps import get_film_asset_store, get_generation_provider
from backend.persistence.film_asset_store import FilmAssetStore
from backend.providers.base import GenerationProvider
from backend.providers.ingest import ingest_task_asset_ids, rewrite_body_asset_refs_for_provider

router = APIRouter(tags=["generation"])


def _submit(
    gen: GenerationProvider,
    store: FilmAssetStore,
    body: dict[str, Any],
    action: str,
) -> dict[str, Any]:
    try:
        staged = rewrite_body_asset_refs_for_provider(body, gen=gen, store=store)
        if action == "image":
            return gen.create_image(staged)
        if action == "video":
            return gen.create_video(staged)
        if action == "video_edit":
            return gen.edit_video(staged)
        raise RuntimeError(f"unknown action: {action}")
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.post("/api/images/generations")
async def images_generations(
    request: Request,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    body = await request.json()
    if not isinstance(body, dict):
        raise HTTPException(status_code=400, detail="JSON object required")
    return _submit(gen, store, body, "image")


@router.post("/api/videos/generations")
async def videos_generations(
    request: Request,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    body = await request.json()
    if not isinstance(body, dict):
        raise HTTPException(status_code=400, detail="JSON object required")
    return _submit(gen, store, body, "video")


@router.post("/api/videos/edits")
async def videos_edits(
    request: Request,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    body = await request.json()
    if not isinstance(body, dict):
        raise HTTPException(status_code=400, detail="JSON object required")
    return _submit(gen, store, body, "video_edit")


@router.get("/api/models")
def list_models(
    media: str | None = None,
    gen: GenerationProvider = Depends(get_generation_provider),
):
    try:
        items = gen.list_models(media=media)
        return {"items": items}
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("/api/registry")
def get_registry(gen: GenerationProvider = Depends(get_generation_provider)):
    try:
        return gen.get_registry()
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("/api/adapters")
def list_adapters(
    for_model: str | None = None,
    gen: GenerationProvider = Depends(get_generation_provider),
):
    try:
        return gen.list_adapters(for_model=for_model)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("/api/tasks/{task_id}")
def get_task(
    task_id: str,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    try:
        task = gen.get_task(task_id)
        return ingest_task_asset_ids(gen, store, task)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("/api/tasks/{task_id}/stream")
def stream_task(
    task_id: str,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    def event_iter():
        try:
            for event, data in gen.iter_task_sse(task_id):
                payload = data
                if event in ("result", "done"):
                    try:
                        obj = json.loads(data)
                    except json.JSONDecodeError:
                        obj = None
                    if isinstance(obj, dict) and obj.get("asset_ids"):
                        rewritten = ingest_task_asset_ids(gen, store, obj)
                        payload = json.dumps(rewritten, ensure_ascii=False)
                yield f"event: {event}\ndata: {payload}\n\n"
        except Exception as exc:
            err = json.dumps({"error": str(exc)}, ensure_ascii=False)
            yield f"event: error\ndata: {err}\n\n"

    return StreamingResponse(event_iter(), media_type="text/event-stream")
