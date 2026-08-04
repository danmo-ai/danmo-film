"""Resolve configured generation + LLM providers from env / settings."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Any

from backend.providers.base import GenerationProvider, LlmChatProvider
from backend.providers.cloud.stub import CloudGenerationProvider
from backend.providers.make import MakeGenerationProvider
from backend.providers.openai_llm import OpenAICompatibleLlmProvider


def _user_data_dir() -> Path:
    override = (os.environ.get("DANMO_FILM_USER_DATA_DIR") or os.environ.get("DANQING_USER_DATA_DIR") or "").strip()
    if override:
        return Path(override).expanduser()
    return Path.home() / ".danmo-film"


def load_settings() -> dict[str, Any]:
    path = _user_data_dir() / "config" / "settings.json"
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise RuntimeError(f"failed to read Film settings {path}: {exc}") from exc
    if not isinstance(data, dict):
        raise RuntimeError(f"Film settings must be a JSON object: {path}")
    return data


def resolve_generation_provider(settings: dict[str, Any] | None = None) -> GenerationProvider:
    cfg = settings if settings is not None else load_settings()
    providers = cfg.get("providers") if isinstance(cfg.get("providers"), dict) else {}
    gen = providers.get("generation") if isinstance(providers.get("generation"), dict) else {}
    kind = (
        os.environ.get("FILM_GENERATION_PROVIDER")
        or gen.get("provider")
        or "danmo-make"
    ).strip()
    if kind in ("danmo-make", "make", "danqing"):
        base = (
            os.environ.get("FILM_MAKE_BASE_URL")
            or gen.get("base_url")
            or "http://127.0.0.1:7800"
        ).strip()
        key = (os.environ.get("FILM_MAKE_API_KEY") or gen.get("api_key") or "").strip()
        return MakeGenerationProvider(base, api_key=key)
    if kind.startswith("cloud") or kind in ("runway", "kling", "minimax", "openai"):
        vendor = gen.get("vendor") or kind.replace("cloud:", "")
        return CloudGenerationProvider(vendor=str(vendor))
    raise RuntimeError(f"Unknown generation provider {kind!r}")


def resolve_llm_provider(settings: dict[str, Any] | None = None) -> LlmChatProvider:
    cfg = settings if settings is not None else load_settings()
    providers = cfg.get("providers") if isinstance(cfg.get("providers"), dict) else {}
    llm = providers.get("llm") if isinstance(providers.get("llm"), dict) else {}
    base = (
        os.environ.get("FILM_LLM_BASE_URL")
        or llm.get("base_url")
        or os.environ.get("FILM_MAKE_BASE_URL")
        or "http://127.0.0.1:7800"
    ).strip()
    key = (os.environ.get("FILM_LLM_API_KEY") or llm.get("api_key") or "").strip()
    model = (os.environ.get("FILM_LLM_MODEL") or llm.get("model") or "").strip()
    pid = (llm.get("provider") or "openai-compatible").strip()
    return OpenAICompatibleLlmProvider(
        base,
        api_key=key,
        default_model=model,
        provider_id=pid,
    )
