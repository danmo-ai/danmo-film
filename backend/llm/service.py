"""Film LLMService — script-parse over a remote OpenAI-compatible chat provider."""

from __future__ import annotations

import json
import queue
import threading
from typing import Any, Callable

from backend.providers.base import LlmChatProvider


class ChatMessage:
    def __init__(self, role: str, content: str) -> None:
        self.role = role
        self.content = content


class LLMService:
    def __init__(self, llm: LlmChatProvider) -> None:
        self._llm = llm

    def is_available(self) -> bool:
        return self._llm.is_available()

    def _script_parse_runtime(
        self,
        model_id: str | None,
    ) -> tuple[str, bool, Callable[[str], str], Callable[[int], int], Callable[..., str]]:
        from backend.llm.prompt_sanitize import sanitize_structured_llm_response

        analyze_model_id = (model_id or "").strip()
        think_active = False
        think_apply = lambda text: text

        def chat_fn(*, messages: list[Any], max_tokens: int) -> str:
            payload = []
            for m in messages:
                if hasattr(m, "role") and hasattr(m, "content"):
                    payload.append({"role": m.role, "content": m.content})
                elif isinstance(m, dict):
                    payload.append({"role": m["role"], "content": m["content"]})
                else:
                    raise RuntimeError(f"unsupported chat message type: {type(m).__name__}")
            raw = self._llm.chat(
                messages=payload,
                model=analyze_model_id,
                max_tokens=max(256, int(max_tokens)),
                temperature=0.35,
            )
            return sanitize_structured_llm_response(raw, think_enabled=think_active)

        token_budget = lambda b: max(256, int(b))
        return analyze_model_id, think_active, think_apply, token_budget, chat_fn

    def script_parse_decompose(
        self,
        request: Any,
        *,
        on_progress: Callable[[str, str], None] | None = None,
        activity_recorder: Any | None = None,
    ) -> Any:
        from backend.core.contracts import LongVideoChapterParsePhaseDTO
        from backend.llm.script_parse.pipeline import run_decompose
        from backend.llm.script_parse.response import decompose_to_response
        from backend.llm.storyboard import normalize_storyboard_locale

        locale = normalize_storyboard_locale(getattr(request, "locale", None))
        parse_phases: list[LongVideoChapterParsePhaseDTO] = []
        project_id = str(getattr(request, "long_video_project_id", "") or "").strip()

        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_started(chapter_title=request.title, target_duration_sec=0.0)

        def report(phase: str, message: str = "") -> None:
            parse_phases.append(LongVideoChapterParsePhaseDTO(phase=phase, message=message))
            if activity_recorder is not None and activity_recorder.active:
                activity_recorder.record_phase(phase, message)
            if on_progress:
                on_progress(phase, message)

        _, _, think_apply, token_budget, chat_fn = self._script_parse_runtime(getattr(request, "model", None))
        try:
            result = run_decompose(
                script_text=request.script_text,
                title=request.title,
                locale=locale,
                chat_fn=chat_fn,
                think_apply=think_apply,
                token_budget=token_budget,
                on_progress=report,
            )
        except ValueError as exc:
            raise RuntimeError(str(exc)) from exc

        response = decompose_to_response(
            result,
            parse_phases=parse_phases,
            parse_run_id=activity_recorder.parse_run_id if activity_recorder and activity_recorder.active else "",
            project_id=project_id,
            locale=locale,
        )
        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_completed(response)
        return response

    def script_parse_expand(
        self,
        request: Any,
        *,
        on_progress: Callable[[str, str], None] | None = None,
        activity_recorder: Any | None = None,
    ) -> Any:
        from backend.core.contracts import LongVideoChapterParsePhaseDTO
        from backend.llm.script_parse.pipeline import run_expand
        from backend.llm.script_parse.response import expand_to_response, script_dict_to_artifact
        from backend.llm.storyboard import normalize_storyboard_locale

        locale = normalize_storyboard_locale(getattr(request, "locale", None))
        parse_phases: list[LongVideoChapterParsePhaseDTO] = []
        project_id = str(getattr(request, "long_video_project_id", "") or "").strip()

        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_started(
                chapter_title="",
                target_duration_sec=float(getattr(request, "target_duration_sec", 60.0) or 60.0),
            )

        def report(phase: str, message: str = "") -> None:
            parse_phases.append(LongVideoChapterParsePhaseDTO(phase=phase, message=message))
            if activity_recorder is not None and activity_recorder.active:
                activity_recorder.record_phase(phase, message)
            if on_progress:
                on_progress(phase, message)

        _, _, think_apply, token_budget, chat_fn = self._script_parse_runtime(getattr(request, "model", None))
        try:
            artifact = script_dict_to_artifact(request.script_artifact)
        except Exception as exc:
            raise RuntimeError(f"invalid script_artifact: {exc}") from exc
        try:
            result = run_expand(
                script=artifact,
                target_duration_sec=float(getattr(request, "target_duration_sec", 60.0) or 60.0),
                segment_duration_sec=float(getattr(request, "segment_duration_sec", 5.0) or 5.0),
                max_clip_sec=float(getattr(request, "max_clip_sec", 10.0) or 10.0),
                locale=locale,
                chat_fn=chat_fn,
                think_apply=think_apply,
                token_budget=token_budget,
                on_progress=report,
                beat_indices=list(request.beat_indices or []),
            )
        except ValueError as exc:
            raise RuntimeError(str(exc)) from exc

        report("done", "done")
        response = expand_to_response(
            result,
            artifact=artifact,
            parse_phases=parse_phases,
            parse_run_id=activity_recorder.parse_run_id if activity_recorder and activity_recorder.active else "",
            project_id=project_id,
        )
        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_completed(response)
        return response

    def script_parse_expand_beat(
        self,
        request: Any,
        *,
        on_progress: Callable[[str, str], None] | None = None,
        activity_recorder: Any | None = None,
    ) -> Any:
        from backend.core.contracts import LongVideoChapterParsePhaseDTO
        from backend.llm.script_parse.pipeline import run_expand_beat
        from backend.llm.script_parse.response import expand_to_response, script_dict_to_artifact
        from backend.llm.storyboard import normalize_storyboard_locale

        locale = normalize_storyboard_locale(getattr(request, "locale", None))
        parse_phases: list[LongVideoChapterParsePhaseDTO] = []
        project_id = str(getattr(request, "long_video_project_id", "") or "").strip()

        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_started(
                chapter_title="",
                target_duration_sec=float(getattr(request, "target_duration_sec", 60.0) or 60.0),
            )

        def report(phase: str, message: str = "") -> None:
            parse_phases.append(LongVideoChapterParsePhaseDTO(phase=phase, message=message))
            if activity_recorder is not None and activity_recorder.active:
                activity_recorder.record_phase(phase, message)
            if on_progress:
                on_progress(phase, message)

        _, _, think_apply, token_budget, chat_fn = self._script_parse_runtime(getattr(request, "model", None))
        try:
            artifact = script_dict_to_artifact(request.script_artifact)
        except Exception as exc:
            raise RuntimeError(f"invalid script_artifact: {exc}") from exc
        try:
            result = run_expand_beat(
                script=artifact,
                beat_index=int(request.beat_index),
                target_duration_sec=float(getattr(request, "target_duration_sec", 60.0) or 60.0),
                segment_duration_sec=float(getattr(request, "segment_duration_sec", 5.0) or 5.0),
                max_clip_sec=float(getattr(request, "max_clip_sec", 10.0) or 10.0),
                locale=locale,
                chat_fn=chat_fn,
                think_apply=think_apply,
                token_budget=token_budget,
                on_progress=report,
                existing_shots=list(request.existing_shots or []),
            )
        except ValueError as exc:
            raise RuntimeError(str(exc)) from exc

        report("done", "done")
        response = expand_to_response(
            result,
            artifact=artifact,
            parse_phases=parse_phases,
            parse_run_id=activity_recorder.parse_run_id if activity_recorder and activity_recorder.active else "",
            project_id=project_id,
        )
        if activity_recorder is not None and activity_recorder.active:
            activity_recorder.record_completed(response)
        return response

    def _script_parse_stream(self, worker: Callable[[Callable[[str, str], None]], Any]):
        event_queue: queue.Queue = queue.Queue()
        holder: dict[str, object] = {}

        def on_progress(phase: str, message: str) -> None:
            event_queue.put({"event": "progress", "phase": phase, "message": message})

        def run_worker() -> None:
            try:
                holder["result"] = worker(on_progress)
            except Exception as exc:
                holder["error"] = exc
            finally:
                event_queue.put(None)

        threading.Thread(target=run_worker, daemon=True).start()
        while True:
            item = event_queue.get()
            if item is None:
                break
            yield f"data: {json.dumps(item, ensure_ascii=False)}\n\n"

        err = holder.get("error")
        if err is not None:
            yield f"data: {json.dumps({'event': 'error', 'detail': str(err)}, ensure_ascii=False)}\n\n"
            return
        result = holder.get("result")
        if result is None:
            yield f"data: {json.dumps({'event': 'error', 'detail': 'script parse returned no result'}, ensure_ascii=False)}\n\n"
            return
        payload = {"event": "result", "data": result.model_dump(mode="json")}
        yield f"data: {json.dumps(payload, ensure_ascii=False)}\n\n"

    def script_parse_decompose_stream(self, request: Any, *, activity_recorder: Any | None = None):
        def worker(on_progress):
            try:
                return self.script_parse_decompose(request, on_progress=on_progress, activity_recorder=activity_recorder)
            except Exception as exc:
                if activity_recorder is not None and activity_recorder.active:
                    activity_recorder.record_failed(str(exc))
                raise

        yield from self._script_parse_stream(worker)

    def script_parse_expand_stream(self, request: Any, *, activity_recorder: Any | None = None):
        def worker(on_progress):
            try:
                return self.script_parse_expand(request, on_progress=on_progress, activity_recorder=activity_recorder)
            except Exception as exc:
                if activity_recorder is not None and activity_recorder.active:
                    activity_recorder.record_failed(str(exc))
                raise

        yield from self._script_parse_stream(worker)

    def script_parse_expand_beat_stream(self, request: Any, *, activity_recorder: Any | None = None):
        def worker(on_progress):
            try:
                return self.script_parse_expand_beat(request, on_progress=on_progress, activity_recorder=activity_recorder)
            except Exception as exc:
                if activity_recorder is not None and activity_recorder.active:
                    activity_recorder.record_failed(str(exc))
                raise

        yield from self._script_parse_stream(worker)
