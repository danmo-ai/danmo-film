"""OpenAI-compatible chat provider (Make /v1 or cloud vendors)."""

from __future__ import annotations

from typing import Any

import httpx

from backend.providers.base import LlmChatProvider


class OpenAICompatibleLlmProvider(LlmChatProvider):
    """POST {base}/v1/chat/completions — works with Danmo Make and many cloud APIs."""

    def __init__(
        self,
        base_url: str,
        *,
        api_key: str = "",
        default_model: str = "",
        provider_id: str = "openai-compatible",
        timeout: float = 180.0,
    ) -> None:
        self.provider_id = provider_id
        self._base = base_url.rstrip("/")
        self._api_key = (api_key or "").strip()
        self._default_model = (default_model or "").strip()
        self._timeout = timeout

    def is_available(self) -> bool:
        return bool(self._base)

    def chat(
        self,
        *,
        messages: list[dict[str, Any]],
        model: str = "",
        max_tokens: int = 4096,
        temperature: float = 0.35,
    ) -> str:
        if not self.is_available():
            raise RuntimeError("LLM provider is not configured (set FILM_LLM_BASE_URL)")
        mid = (model or self._default_model or "").strip()
        if not mid:
            raise RuntimeError("LLM model id is required (request.model or FILM_LLM_MODEL)")
        headers = {"Accept": "application/json", "Content-Type": "application/json"}
        if self._api_key:
            headers["Authorization"] = f"Bearer {self._api_key}"
        payload = {
            "model": mid,
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "stream": False,
        }
        url = f"{self._base}/v1/chat/completions"
        with httpx.Client(timeout=self._timeout) as client:
            r = client.post(url, headers=headers, json=payload)
            if r.status_code >= 400:
                raise RuntimeError(f"LLM chat failed ({r.status_code}): {r.text[:500]}")
            data = r.json()
        try:
            content = data["choices"][0]["message"]["content"]
        except (KeyError, IndexError, TypeError) as exc:
            raise RuntimeError(f"unexpected LLM response shape: {data!r}") from exc
        if not isinstance(content, str):
            raise RuntimeError("LLM response content is not a string")
        return content
