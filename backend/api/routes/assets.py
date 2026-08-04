"""Film-owned asset API — local SoR for images, clips, and assembled finals."""

from __future__ import annotations

import mimetypes
from pathlib import Path
from typing import Any

from fastapi import APIRouter, Depends, File, Form, HTTPException, UploadFile
from fastapi.responses import FileResponse
from pydantic import BaseModel, Field

from backend.api.deps import get_film_asset_store, get_generation_provider
from backend.persistence.film_asset_store import FilmAssetStore, is_film_asset_id
from backend.providers.base import GenerationProvider
from backend.providers.ingest import guess_kind, ingest_provider_asset

router = APIRouter(prefix="/api/assets", tags=["assets"])


class IngestFromProviderBody(BaseModel):
    provider_asset_id: str
    kind: str = "video"
    mime_type: str = ""
    source_task_id: str = ""
    metadata: dict[str, Any] = Field(default_factory=dict)


@router.post("")
async def upload_asset(
    file: UploadFile = File(...),
    kind: str = Form(""),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    raw_name = file.filename or "upload.bin"
    suffix = Path(raw_name).suffix or ""
    mime = file.content_type or mimetypes.guess_type(raw_name)[0] or "application/octet-stream"
    resolved_kind = (kind or "").strip() or guess_kind(mime, fallback="image")
    import tempfile

    with tempfile.TemporaryDirectory(prefix="film_upload_") as tmp:
        dest = Path(tmp) / f"upload{suffix or '.bin'}"
        dest.write_bytes(await file.read())
        row = store.create_from_file(
            dest,
            kind=resolved_kind,
            mime_type=mime,
            source_provider="film-upload",
            metadata={"filename": raw_name},
        )
    return row


@router.post("/from-provider")
def ingest_from_provider(
    body: IngestFromProviderBody,
    store: FilmAssetStore = Depends(get_film_asset_store),
    gen: GenerationProvider = Depends(get_generation_provider),
):
    try:
        return ingest_provider_asset(
            gen,
            store,
            body.provider_asset_id,
            kind=body.kind,
            mime_type=body.mime_type,
            source_task_id=body.source_task_id,
            metadata=body.metadata or None,
        )
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc


@router.get("")
def list_assets(
    kind: str | None = None,
    limit: int = 100,
    offset: int = 0,
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    items = store.list_assets(kind=kind, limit=limit, offset=offset)
    return {"items": items, "total": len(items)}


@router.get("/groups")
def list_asset_groups():
    """Film v1 has no Make-style asset groups — empty list (fail soft for gallery UI)."""
    return {"items": []}


@router.get("/{asset_id}")
def get_asset(asset_id: str, store: FilmAssetStore = Depends(get_film_asset_store)):
    row = store.get(asset_id)
    if not row:
        raise HTTPException(status_code=404, detail=f"asset not found: {asset_id}")
    return row


@router.get("/{asset_id}/file")
def get_asset_file(asset_id: str, store: FilmAssetStore = Depends(get_film_asset_store)):
    if not is_film_asset_id(asset_id):
        raise HTTPException(
            status_code=404,
            detail=(
                f"Not a Film asset id ({asset_id!r}). "
                "Ingest provider results via POST /api/assets/from-provider first."
            ),
        )
    try:
        path = store.resolve_path(asset_id)
    except RuntimeError as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc
    row = store.get(asset_id)
    mime = (row or {}).get("mime_type") or "application/octet-stream"
    return FileResponse(path, media_type=str(mime), filename=path.name)


@router.get("/{asset_id}/thumbnail")
def get_asset_thumbnail(asset_id: str, store: FilmAssetStore = Depends(get_film_asset_store)):
    row = store.get(asset_id)
    if not row:
        raise HTTPException(status_code=404, detail=f"asset not found: {asset_id}")
    thumb = row.get("thumbnail_path")
    if thumb and Path(str(thumb)).is_file():
        return FileResponse(str(thumb), media_type="image/webp")
    # Fall back to full file for images; videos have no thumb yet.
    if row.get("kind") == "image":
        return get_asset_file(asset_id, store)
    raise HTTPException(status_code=404, detail="thumbnail not available")


@router.delete("/{asset_id}")
def delete_asset(asset_id: str, store: FilmAssetStore = Depends(get_film_asset_store)):
    if not store.delete(asset_id):
        raise HTTPException(status_code=404, detail=f"asset not found: {asset_id}")
    return {"ok": True}
