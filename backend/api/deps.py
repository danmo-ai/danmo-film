"""FastAPI dependency providers for Danmo Film."""

from __future__ import annotations

from functools import lru_cache

from backend.llm.service import LLMService
from backend.persistence.long_video_activity_store import LongVideoActivityStore
from backend.persistence.long_video_project_store import LongVideoProjectStore
from backend.providers.base import GenerationProvider, LlmChatProvider
from backend.providers.resolve import resolve_generation_provider, resolve_llm_provider
from backend.runtime_paths import film_db_path


@lru_cache(maxsize=1)
def get_long_video_project_store() -> LongVideoProjectStore:
    return LongVideoProjectStore(film_db_path())


@lru_cache(maxsize=1)
def get_long_video_activity_store() -> LongVideoActivityStore:
    return LongVideoActivityStore(film_db_path())


@lru_cache(maxsize=1)
def get_llm_chat_provider() -> LlmChatProvider:
    return resolve_llm_provider()


@lru_cache(maxsize=1)
def get_generation_provider() -> GenerationProvider:
    return resolve_generation_provider()


@lru_cache(maxsize=1)
def get_llm_service() -> LLMService:
    return LLMService(get_llm_chat_provider())
