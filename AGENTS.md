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
| API entry | `cmd/danmo-film/main.go` (Go, Gin :7803) |
| Frontend | `frontend/` (Vite :5803) |
| Ports | `core/port/` (`LLMProvider`, `GenerationProvider`) |
| LLM adapters | `core/adapter/llm/` (OpenAI / Responses / Anthropic / mock) |
| Video adapters | `core/adapter/gen/` (Make, Volcengine Seedance, DashScope 万相, MiniMax Hailuo) |
| Film assets | `core/store/sqlite/` + `/api/assets` |
| Script-parse | `core/service/scriptparse/` |
| Settings | `~/.danmo-film/config/settings.json` + `/api/settings` |
| Workspace assets | `{workspace}/assets/` (`fas_*`) |
| Dev ports | API **7803**, UI **5803** |

## Architecture

```
Film UI / API
 ├─ /api/assets/*              → FilmAssetStore (SoR)
 ├─ /api/long-video/*          → projects + assemble (Film ffmpeg)
 ├─ /api/script-parse/*        → port.LLMProvider (OpenAI-compatible or Anthropic)
 └─ /api/images|videos|tasks/* → port.GenerationProvider peer
        ├─ stage fas_* inputs to provider for the job only
        ├─ submit create/edit
        └─ ingest result bytes → fas_* (never upload finals back)
```

Generation `providers.generation.provider`: `danmo-make` | `volcengine` | `dashscope` | `minimax`. One official async video API per cloud vendor. Fail loud — no silent cross-vendor fallback.

## Commands

```bash
make dev        # API :7803 + Vite :5803 (one-shot)
make stop
make backend    # API only
make health
make test
```

## Hard rules

1. Fail loud — no silent provider fallbacks.
2. Do not import Make engine internals; use HTTP providers.
3. Make / cloud = clip + LLM APIs only. Film owns assemble, edit timeline, and assets.
4. Provider `StageInput` is ephemeral job staging — not ownership transfer.
5. Product name: **Danmo Film** / `danmo-film-*`.
6. Domain/service code talks only to `core/port`; adapters live under `core/adapter`.
