"""REST API for long-video workbench projects."""

from __future__ import annotations

from pathlib import Path
from typing import Any, Optional

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from backend.api.deps import (
    get_film_asset_store,
    get_generation_provider,
    get_long_video_activity_store,
    get_long_video_project_store,
)
from backend.core.contracts import LongVideoProjectActivityDTO, LongVideoProjectActivityListResponse
from backend.persistence.film_asset_store import FilmAssetStore
from backend.persistence.long_video_activity_store import LongVideoActivityStore
from backend.persistence.long_video_project_store import LongVideoProjectStore
from backend.providers.base import GenerationProvider

router = APIRouter(prefix="/api/long-video", tags=["long-video"])


class LongVideoProjectCreateRequest(BaseModel):
    title: str = ""
    state: Optional[dict[str, Any]] = None


class LongVideoProjectUpdateRequest(BaseModel):
    title: Optional[str] = None
    state: Optional[dict[str, Any]] = None


class SceneGroundingDepthRequest(BaseModel):
    source_asset_id: str
    width: int = 1024
    height: int = 1024


@router.get("/projects")
async def list_long_video_projects(
    limit: int = 100,
    store: LongVideoProjectStore = Depends(get_long_video_project_store),
):
    return {"items": store.list_projects(limit=limit)}


@router.post("/projects", status_code=201)
async def create_long_video_project(
    body: LongVideoProjectCreateRequest,
    store: LongVideoProjectStore = Depends(get_long_video_project_store),
):
    return store.create_project(title=body.title, state=body.state)


@router.get("/projects/{project_id}")
async def get_long_video_project(
    project_id: str,
    store: LongVideoProjectStore = Depends(get_long_video_project_store),
):
    row = store.get_project(project_id)
    if not row:
        raise HTTPException(status_code=404, detail="long video project not found")
    return row


@router.put("/projects/{project_id}")
async def update_long_video_project(
    project_id: str,
    body: LongVideoProjectUpdateRequest,
    store: LongVideoProjectStore = Depends(get_long_video_project_store),
):
    row = store.update_project(project_id, title=body.title, state=body.state)
    if not row:
        raise HTTPException(status_code=404, detail="long video project not found")
    return row


@router.delete("/projects/{project_id}", status_code=204)
async def delete_long_video_project(
    project_id: str,
    store: LongVideoProjectStore = Depends(get_long_video_project_store),
):
    if not store.delete_project(project_id):
        raise HTTPException(status_code=404, detail="long video project not found")


@router.get("/projects/{project_id}/activity")
async def list_long_video_project_activity(
    project_id: str,
    limit: int = 200,
    offset: int = 0,
    category: str | None = None,
    phase: str | None = None,
    event_type: str | None = None,
    parse_run_id: str | None = None,
    task_id: str | None = None,
    shot_id: str | None = None,
    project_store: LongVideoProjectStore = Depends(get_long_video_project_store),
    activity_store: LongVideoActivityStore = Depends(get_long_video_activity_store),
):
    if not project_store.get_project(project_id):
        raise HTTPException(status_code=404, detail="long video project not found")
    items = activity_store.list_events(
        project_id,
        limit=limit,
        offset=offset,
        category=category,
        phase=phase,
        event_type=event_type,
        parse_run_id=parse_run_id,
        task_id=task_id,
        shot_id=shot_id,
    )
    total = activity_store.count_events(
        project_id,
        category=category,
        event_type=event_type,
        parse_run_id=parse_run_id,
    )
    return LongVideoProjectActivityListResponse(
        items=[LongVideoProjectActivityDTO(**row) for row in items],
        total=total,
    )


@router.get("/projects/{project_id}/activity/parse-runs/{parse_run_id}")
async def get_long_video_parse_run(
    project_id: str,
    parse_run_id: str,
    project_store: LongVideoProjectStore = Depends(get_long_video_project_store),
    activity_store: LongVideoActivityStore = Depends(get_long_video_activity_store),
):
    if not project_store.get_project(project_id):
        raise HTTPException(status_code=404, detail="long video project not found")
    row = activity_store.get_parse_run(project_id, parse_run_id)
    if not row:
        raise HTTPException(status_code=404, detail="parse run not found")
    return row


@router.post("/scene-grounding/depth-from-asset")
async def scene_grounding_depth_from_asset(
    body: SceneGroundingDepthRequest,
):
    """Depth-pro is a provider capability — Film does not host it yet."""
    _ = body
    raise HTTPException(
        status_code=501,
        detail=(
            "Scene-grounding depth encoding is not hosted in Danmo Film yet. "
            "Run depth on a generation provider and ingest the result into Film assets."
        ),
    )


class AssembleRequest(BaseModel):
    segment_asset_ids: list[str]
    overlap_frames: int = 0
    fps: float = 16.0
    title: str = ""


@router.post("/assemble")
async def assemble_segments(
    body: AssembleRequest,
    gen: GenerationProvider = Depends(get_generation_provider),
    store: FilmAssetStore = Depends(get_film_asset_store),
):
    """Stitch segment videos into one Film-owned final asset (no provider upload-back)."""
    import tempfile
    from datetime import datetime

    from backend.film.stitch import stitch_segment_paths
    from backend.providers.ingest import materialize_segment_path

    ids = [str(x).strip() for x in (body.segment_asset_ids or []) if str(x).strip()]
    if not ids:
        raise HTTPException(status_code=400, detail="segment_asset_ids is required")
    try:
        with tempfile.TemporaryDirectory(prefix="film_assemble_") as tmp:
            tmp_path = Path(tmp)
            paths: list[Path] = []
            for i, aid in enumerate(ids):
                dest = tmp_path / f"seg_{i:03d}.mp4"
                materialize_segment_path(aid, dest=dest, gen=gen, store=store)
                paths.append(dest)
            out = tmp_path / f"assembled_{datetime.now().strftime('%Y%m%d_%H%M%S')}.mp4"
            stitch_segment_paths(
                paths,
                output=out,
                overlap_frames=int(body.overlap_frames or 0),
                fps=float(body.fps or 16),
            )
            row = store.create_from_file(
                out,
                kind="video",
                mime_type="video/mp4",
                source_provider="film-assemble",
                metadata={
                    "title": body.title,
                    "segment_count": len(ids),
                    "segment_asset_ids": ids,
                },
            )
    except Exception as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
    return {
        "asset_id": str(row["id"]),
        "title": body.title,
        "segment_count": len(ids),
    }

