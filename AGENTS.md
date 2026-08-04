# AGENTS.md — Danmo Film

Operational guide for contributors and coding agents.

## What this is

**Danmo Film** — long-form video workbench (script parse, storyboard, project timeline, assemble, asset management).

**Danmo Make** (and cloud vendors) are **peer generation providers** for LLM / image / short video — the same role as a cloud API. Film downloads result bytes, stores them locally, merges/edits, and owns the gallery. There is **no asset回传** to Make.

| Product | Repo | Role |
|---------|------|------|
| Make | `danmo-ai/danmo-make` | Local image / video / audio generation API |
| Film | `danmo-ai/danmo-film` | Narrative workbench + Film asset SoR |
| Work | `danmo-ai/danmo-work` | Long-horizon AI work agent |
| Inbox | `danmo-ai/danmo-inbox` | AI email stream organization |

## Quick reference

| Item | Location |
|------|----------|
| API entry | `backend/main.py` |
| Frontend | `frontend/` (Vite :5803) |
| Providers | `backend/providers/` (Make / cloud peers) |
| Film assets | `backend/persistence/film_asset_store.py` + `/api/assets` |
| Film domain | `backend/film/` |
| Script-parse LLM | `backend/llm/` |
| User data | `~/.danmo-film/` |
| Workspace assets | `{workspace}/assets/` (`fas_*`) |
| Dev ports | API **7803**, UI **5803** |

## Architecture

```
Film UI / API
  ├─ /api/assets/*              → FilmAssetStore (SoR)
  ├─ /api/long-video/*          → projects + assemble (Film ffmpeg)
  ├─ /api/script-parse/*        → LlmChatProvider (Make or any OpenAI-compatible)
  └─ /api/images|videos|tasks/* → GenerationProvider peer
        ├─ stage fas_* inputs to provider for the job only
        ├─ submit create/edit
        └─ ingest result bytes → fas_* (never upload finals back)
```

## Commands

```bash
make install
make install-frontend
make backend    # :7803
make frontend   # :5803
make health
```

## Hard rules

1. Fail loud — no silent provider fallbacks.
2. Do not import Make `backend/engine/families/*` into Film hot paths; use HTTP providers.
3. Make / cloud = clip + LLM APIs only. Film owns assemble, edit timeline, and assets.
4. Provider `stage_input_file` is ephemeral job staging — not ownership transfer.
5. Product name: **Danmo Film** / `danmo-film-*`.
