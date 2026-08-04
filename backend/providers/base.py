"""Generation / LLM provider contracts for Danmo Film.

Providers are peers (Make local API, cloud vendors): submit jobs, poll tasks,
download result bytes. Film owns assets — providers are not Film's gallery.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from pathlib import Path
from typing import Any, Iterator


class GenerationProvider(ABC):
    """Submit image/video jobs and poll tasks (Make local API or cloud vendor)."""

    provider_id: str

    @abstractmethod
    def health(self) -> dict[str, Any]:
        ...

    @abstractmethod
    def list_models(self, *, media: str | None = None) -> list[dict[str, Any]]:
        ...

    def get_registry(self) -> dict[str, Any]:
        raise RuntimeError(f"{self.provider_id} does not expose a model registry")

    def list_adapters(self, *, for_model: str | None = None) -> Any:
        raise RuntimeError(f"{self.provider_id} does not list adapters")

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

    def download_result(self, provider_asset_id: str, dest: Path) -> Path:
        """Download a provider-native result asset onto ``dest``."""
        raise RuntimeError(f"{self.provider_id} does not support download_result")

    def stage_input_file(self, path: Path, *, mime_type: str = "application/octet-stream") -> dict[str, Any]:
        """Ephemeral upload of a Film-owned file as a job input (not Film SoR)."""
        raise RuntimeError(f"{self.provider_id} does not support stage_input_file")

    def iter_task_sse(self, task_id: str) -> Iterator[tuple[str, str]]:
        """Yield ``(event_name, data_json_str)`` for task SSE. Optional."""
        raise RuntimeError(f"{self.provider_id} does not support iter_task_sse")


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
