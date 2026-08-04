"""Danmo Make HTTP generation provider."""

from __future__ import annotations

from pathlib import Path
from typing import Any

import httpx

from backend.providers.base import GenerationProvider


class MakeGenerationProvider(GenerationProvider):
    provider_id = "danmo-make"

    def __init__(self, base_url: str, *, api_key: str = "", timeout: float = 120.0) -> None:
        self._base = base_url.rstrip("/")
        self._api_key = (api_key or "").strip()
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        headers = {"Accept": "application/json"}
        if self._api_key:
            headers["Authorization"] = f"Bearer {self._api_key}"
            headers["X-API-Key"] = self._api_key
        return headers

    def _client(self) -> httpx.Client:
        return httpx.Client(base_url=self._base, headers=self._headers(), timeout=self._timeout)

    def health(self) -> dict[str, Any]:
        with self._client() as client:
            r = client.get("/api/system/health")
            r.raise_for_status()
            return r.json()

    def list_models(self, *, media: str | None = None) -> list[dict[str, Any]]:
        params = {}
        if media:
            params["media"] = media
        with self._client() as client:
            r = client.get("/api/models", params=params)
            r.raise_for_status()
            data = r.json()
        if isinstance(data, list):
            return data
        if isinstance(data, dict) and "items" in data:
            return list(data["items"])
        if isinstance(data, dict) and "models" in data:
            return list(data["models"])
        raise RuntimeError(f"unexpected /api/models payload from Make: {type(data).__name__}")

    def create_image(self, body: dict[str, Any]) -> dict[str, Any]:
        with self._client() as client:
            r = client.post("/api/images/generations", json=body)
            r.raise_for_status()
            return r.json()

    def create_video(self, body: dict[str, Any]) -> dict[str, Any]:
        with self._client() as client:
            r = client.post("/api/videos/generations", json=body)
            r.raise_for_status()
            return r.json()

    def edit_video(self, body: dict[str, Any]) -> dict[str, Any]:
        with self._client() as client:
            r = client.post("/api/videos/edits", json=body)
            r.raise_for_status()
            return r.json()

    def get_task(self, task_id: str) -> dict[str, Any]:
        with self._client() as client:
            r = client.get(f"/api/tasks/{task_id}")
            r.raise_for_status()
            return r.json()

    def download_asset_file(self, asset_id: str, dest: Path) -> Path:
        from pathlib import Path as _Path

        dest = _Path(dest)
        dest.parent.mkdir(parents=True, exist_ok=True)
        with self._client() as client:
            r = client.get(f"/api/assets/{asset_id}/file")
            if r.status_code >= 400:
                raise RuntimeError(f"Make asset download failed ({r.status_code}): {r.text[:300]}")
            dest.write_bytes(r.content)
        return dest

    def upload_asset_file(self, path: Path, *, mime_type: str = "video/mp4") -> dict[str, Any]:
        from pathlib import Path as _Path

        path = _Path(path)
        if not path.is_file():
            raise RuntimeError(f"upload path missing: {path}")
        with self._client() as client:
            with path.open("rb") as fh:
                r = client.post(
                    "/api/assets",
                    files={"file": (path.name, fh, mime_type)},
                )
            if r.status_code >= 400:
                raise RuntimeError(f"Make asset upload failed ({r.status_code}): {r.text[:300]}")
            return r.json()
