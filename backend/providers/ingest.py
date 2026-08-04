"""Ingest provider results into Film assets; stage Film assets as ephemeral provider job inputs."""

from __future__ import annotations

import mimetypes
import tempfile
from pathlib import Path
from typing import Any

from backend.persistence.film_asset_store import FilmAssetStore, is_film_asset_id
from backend.providers.base import GenerationProvider

_SINGLE_ASSET_KEYS = (
    "source_asset_id",
    "control_asset_id",
    "mask_asset_id",
    "start_asset_id",
    "end_asset_id",
    "tail_asset_id",
)
_LIST_ASSET_KEYS = ("reference_asset_ids",)


def guess_kind(mime_type: str, *, fallback: str = "video") -> str:
    mt = (mime_type or "").lower()
    if mt.startswith("image/"):
        return "image"
    if mt.startswith("audio/"):
        return "audio"
    if mt.startswith("video/"):
        return "video"
    return fallback


def ingest_provider_asset(
    gen: GenerationProvider,
    store: FilmAssetStore,
    provider_asset_id: str,
    *,
    kind: str = "video",
    mime_type: str = "",
    source_task_id: str = "",
    metadata: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Download bytes from a generation provider and store as a Film asset."""
    ref = str(provider_asset_id or "").strip()
    if not ref:
        raise RuntimeError("provider_asset_id is required")
    if is_film_asset_id(ref):
        existing = store.get(ref)
        if not existing:
            raise RuntimeError(f"Film asset not found: {ref}")
        return existing
    cached = store.find_by_provider_ref(gen.provider_id, ref)
    if cached:
        return cached
    suffix = ".mp4" if kind == "video" else ".png" if kind == "image" else ".bin"
    with tempfile.TemporaryDirectory(prefix="film_ingest_") as tmp:
        dest = Path(tmp) / f"provider_{ref}{suffix}"
        gen.download_result(ref, dest)
        mt = mime_type or mimetypes.guess_type(str(dest))[0] or (
            "video/mp4" if kind == "video" else "image/png" if kind == "image" else "application/octet-stream"
        )
        return store.create_from_file(
            dest,
            kind=kind,
            mime_type=mt,
            source_provider=gen.provider_id,
            provider_ref=ref,
            source_task_id=source_task_id,
            metadata=metadata,
        )


def ingest_task_asset_ids(
    gen: GenerationProvider,
    store: FilmAssetStore,
    task: dict[str, Any],
    *,
    default_kind: str = "video",
) -> dict[str, Any]:
    """Rewrite provider ``asset_ids`` on a completed task payload to Film ``fas_*`` ids."""
    out = dict(task)
    status = str(out.get("status") or "").lower()
    raw_ids = out.get("asset_ids") or []
    if not isinstance(raw_ids, list) or not raw_ids:
        return out
    # SSE ``result`` often has asset_ids without status; skip only clearly incomplete tasks.
    if status in ("queued", "pending", "running", "cancelled", "canceled", "failed", "error"):
        return out
    kind = default_kind
    task_kind = str(out.get("kind") or out.get("task_kind") or "").lower()
    if "image" in task_kind:
        kind = "image"
    elif "audio" in task_kind:
        kind = "audio"
    film_ids: list[str] = []
    tid = str(out.get("id") or out.get("task_id") or "")
    for rid in raw_ids:
        aid = str(rid or "").strip()
        if not aid:
            continue
        row = ingest_provider_asset(gen, store, aid, kind=kind, source_task_id=tid)
        film_ids.append(str(row["id"]))
    out["asset_ids"] = film_ids
    out["provider_asset_ids"] = [str(x) for x in raw_ids if str(x).strip()]
    return out


def stage_film_asset_to_provider(
    gen: GenerationProvider,
    store: FilmAssetStore,
    film_asset_id: str,
) -> str:
    """Upload a Film asset as an ephemeral provider job input; return provider-native id."""
    path = store.resolve_path(film_asset_id)
    row = store.get(film_asset_id)
    mime = (row or {}).get("mime_type") or mimetypes.guess_type(str(path))[0] or "application/octet-stream"
    staged = gen.stage_input_file(path, mime_type=str(mime))
    pid = staged.get("id") or staged.get("asset_id")
    if not pid:
        raise RuntimeError(f"provider stage_input_file returned no id: {staged!r}")
    return str(pid)


def rewrite_body_asset_refs_for_provider(
    body: dict[str, Any],
    *,
    gen: GenerationProvider,
    store: FilmAssetStore,
) -> dict[str, Any]:
    """Replace Film ``fas_*`` refs in a generation request with provider-native ids."""
    out = dict(body)
    for key in _SINGLE_ASSET_KEYS:
        val = out.get(key)
        if isinstance(val, str) and is_film_asset_id(val):
            out[key] = stage_film_asset_to_provider(gen, store, val)
    for key in _LIST_ASSET_KEYS:
        val = out.get(key)
        if not isinstance(val, list):
            continue
        rewritten: list[str] = []
        for item in val:
            s = str(item or "").strip()
            if not s:
                continue
            if is_film_asset_id(s):
                rewritten.append(stage_film_asset_to_provider(gen, store, s))
            else:
                rewritten.append(s)
        out[key] = rewritten
    return out


def materialize_segment_path(
    asset_id: str,
    *,
    dest: Path,
    gen: GenerationProvider,
    store: FilmAssetStore,
) -> Path:
    """Resolve a Film or provider asset id onto ``dest`` for assemble."""
    aid = str(asset_id or "").strip()
    if not aid:
        raise RuntimeError("empty segment asset id")
    if is_film_asset_id(aid):
        src = store.resolve_path(aid)
        dest.parent.mkdir(parents=True, exist_ok=True)
        import shutil

        shutil.copy2(src, dest)
        return dest
    # Legacy / direct provider ref — download bytes for stitch (Film remains SoR for finals).
    gen.download_result(aid, dest)
    return dest
