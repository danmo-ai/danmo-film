"""Danmo Make HTTP generation provider — peer API, same role as a cloud vendor.

Make supplies clip generation + ephemeral job-input staging. Film never treats
Make's asset store as the system of record for Film projects or finals.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any, Iterator

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

    def get_registry(self) -> dict[str, Any]:
        with self._client() as client:
            r = client.get("/api/registry")
            r.raise_for_status()
            data = r.json()
        if not isinstance(data, dict):
            raise RuntimeError(f"unexpected /api/registry payload from Make: {type(data).__name__}")
        return data

    def list_adapters(self, *, for_model: str | None = None) -> Any:
        params = {}
        if for_model:
            params["for_model"] = for_model
        with self._client() as client:
            r = client.get("/api/adapters", params=params)
            r.raise_for_status()
            return r.json()

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

    def download_result(self, provider_asset_id: str, dest: Path) -> Path:
        dest = Path(dest)
        dest.parent.mkdir(parents=True, exist_ok=True)
        with self._client() as client:
            r = client.get(f"/api/assets/{provider_asset_id}/file")
            if r.status_code >= 400:
                raise RuntimeError(f"Make result download failed ({r.status_code}): {r.text[:300]}")
            dest.write_bytes(r.content)
        return dest

    def stage_input_file(self, path: Path, *, mime_type: str = "application/octet-stream") -> dict[str, Any]:
        """Ephemeral Make upload so a job can reference the file — not Film ownership transfer."""
        path = Path(path)
        if not path.is_file():
            raise RuntimeError(f"stage path missing: {path}")
        with self._client() as client:
            with path.open("rb") as fh:
                r = client.post(
                    "/api/assets",
                    files={"file": (path.name, fh, mime_type)},
                )
            if r.status_code >= 400:
                raise RuntimeError(f"Make stage_input_file failed ({r.status_code}): {r.text[:300]}")
            return r.json()

    def iter_task_sse(self, task_id: str) -> Iterator[tuple[str, str]]:
        """Proxy Make task SSE as ``(event, data)`` pairs."""
        url = f"{self._base}/api/tasks/{task_id}/stream"
        with httpx.Client(headers=self._headers(), timeout=None) as client:
            with client.stream("GET", url) as resp:
                if resp.status_code >= 400:
                    raise RuntimeError(f"Make task stream failed ({resp.status_code})")
                event = "message"
                data_lines: list[str] = []
                for raw in resp.iter_lines():
                    line = raw.decode("utf-8") if isinstance(raw, (bytes, bytearray)) else str(raw)
                    if line.startswith(":"):
                        continue
                    if line.startswith("event:"):
                        event = line[6:].strip() or "message"
                        continue
                    if line.startswith("data:"):
                        data_lines.append(line[5:].lstrip())
                        continue
                    if line == "":
                        if data_lines:
                            yield event, "\n".join(data_lines)
                        event = "message"
                        data_lines = []
                if data_lines:
                    yield event, "\n".join(data_lines)
