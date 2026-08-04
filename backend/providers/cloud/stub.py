"""Cloud generation provider stubs — fail loud until a vendor is configured."""

from __future__ import annotations

from typing import Any

from backend.providers.base import GenerationProvider


class CloudGenerationProvider(GenerationProvider):
    """Placeholder for vendor APIs (Runway / Kling / MiniMax / etc.)."""

    provider_id = "cloud"

    def __init__(self, vendor: str = "", **_kwargs: Any) -> None:
        self._vendor = (vendor or "").strip()
        self.provider_id = f"cloud:{self._vendor}" if self._vendor else "cloud"

    def _unsupported(self, action: str) -> None:
        vendor = self._vendor or "(unset)"
        raise RuntimeError(
            f"Cloud generation provider {vendor!r} does not implement {action} yet. "
            "Configure provider=danmo-make or implement backend/providers/cloud/<vendor>.py."
        )

    def health(self) -> dict[str, Any]:
        return {"ok": False, "provider": self.provider_id, "vendor": self._vendor, "configured": False}

    def list_models(self, *, media: str | None = None) -> list[dict[str, Any]]:
        self._unsupported("list_models")
        return []

    def create_image(self, body: dict[str, Any]) -> dict[str, Any]:
        self._unsupported("create_image")
        return {}

    def create_video(self, body: dict[str, Any]) -> dict[str, Any]:
        self._unsupported("create_video")
        return {}

    def edit_video(self, body: dict[str, Any]) -> dict[str, Any]:
        self._unsupported("edit_video")
        return {}

    def get_task(self, task_id: str) -> dict[str, Any]:
        self._unsupported("get_task")
        return {}
