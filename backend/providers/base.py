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
    def long_generate_video(self, body: dict[str, Any]) -> dict[str, Any]:
        ...

    @abstractmethod
    def get_task(self, task_id: str) -> dict[str, Any]:
        ...


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
