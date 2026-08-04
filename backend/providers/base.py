"""Generation / LLM provider contracts for Danmo Film."""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any


class GenerationProvider(ABC):
    """Submit image/video jobs and poll tasks (Make local API or cloud vendor)."""

    provider_id: str

    @abstractmethod
    def health(self) -> dict[str, Any]:
        ...

    @abstractmethod
    def list_models(self, *, media: str | None = None) -> list[dict[str, Any]]:
        ...

    @abstractmethod
    def create_image(self, body: dict[str, Any]) -> dict[str, Any]:
        ...

    @abstractmethod
    def create_video(self, body: dict[str, Any]) -> dict[str, Any]:
        ...

    @abstractmethod
    def edit_video(self, body: dict[str, Any]) -> dict[str, Any]:
        ...

    @abstractmethod
    def get_task(self, task_id: str) -> dict[str, Any]:
        ...

    def download_asset_file(self, asset_id: str, dest: Path) -> Path:
        raise RuntimeError(f"{self.provider_id} does not support download_asset_file")

    def upload_asset_file(self, path: Path, *, mime_type: str = "video/mp4") -> dict[str, Any]:
        raise RuntimeError(f"{self.provider_id} does not support upload_asset_file")


class LlmChatProvider(ABC):
    """OpenAI-compatible chat completions used by script-parse."""

    provider_id: str

    @abstractmethod
    def is_available(self) -> bool:
        ...

    @abstractmethod
    def chat(
        self,
        *,
        messages: list[dict[str, Any]],
        model: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.35,
    ) -> str:
        ...
