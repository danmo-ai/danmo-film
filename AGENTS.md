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

**Video** presets: `danmo-make` | `volcengine` | `dashscope` | `minimax` → `providers.generation_profiles`.  
**Image** presets: `danmo-make` | `volcengine` | `dashscope` | `minimax` | `openai` → `providers.image_profiles`.  
Both are **multi-instance profiles** (same shape as LLM): `id` / `label` / `vendor` / credentials / models. Same vendor may be added twice (Plan vs API) and distinguished by label. Create page uses `profile_id/model_id`; `gen.Router` routes `CreateVideo` via video profiles and `CreateImage` via image profiles. Fail loud — no silent cross-vendor fallback. **Model lists are live refresh only** (no local curated catalogs): Make `GET /api/models?installed=true&media=`; DashScope Token Plan `GET …/compatible-mode/v1/models` (full subscription set; no type filter unless rows carry media); open-platform may use `GET /api/v1/models`; Volcengine / MiniMax / OpenAI `GET …/models`. Untyped rows are shown as-is — user enables what they need.

## Video providers

**Unified port + per-vendor adapters.** Domain code depends only on `port.GenerationProvider` ([`core/port/generation.go`](core/port/generation.go)). Each cloud vendor implements that interface under [`core/adapter/gen/`](core/adapter/gen/); [`gen.Resolve`](core/adapter/gen/resolve.go) returns a **Router** that dispatches by `profile_id/model_id` (same shape as LLM).

```
Film UI / API  →  port.GenerationProvider  →  gen.Resolve → Router
                                              ├─ video profiles → Make / Volcengine / DashScope / MiniMax
                                              └─ image profiles → Make / Volcengine / DashScope / MiniMax / OpenAI Images
```

| Layer | Role |
|-------|------|
| Port | `ID`, `Health`, `ListModels`, `CreateImage` / `CreateVideo` / `EditVideo`, `GetTask`, `IterTaskSSE`, `StageInput`, `DownloadResult` |
| Factory | `Resolve(settings)` → `gen.Router` (or single vendor if `FILM_GENERATION_PROVIDER` set); unknown → error |
| Router | Video vs image profile maps; `profile_id/model_id`; task ids `profile_id:native_id` (multi-account safe) |
| Adapter | Vendor native async (or sync-image) HTTP API; `ListModels` from vendor refresh (no curated catalogs); stage inputs; poll / download |

**Not in repo yet:** Kling / 可灵. To add: new `kling.go` implementing `GenerationProvider`, a `Resolve` case, settings block + UI — same pattern; no new abstraction layer. Prefer the vendor’s open-platform pay-as-you-go async API unless a subscription channel speaks that same protocol.

### Token plan vs pay-as-you-go

Vendors often sell **both** a subscription / Token Plan **and** metered API (sometimes plus prepaid video point packs). Film does **not** branch on billing type: each vendor config is only `base_url` + `api_key` + `model`. The adapter always speaks the **native async video** protocol; the cloud account decides which billing lane that Key+URL pair hits.

| Situation | Implication for Film |
|-----------|----------------------|
| Same native video API, different metering (e.g. MiniMax video pack vs balance) | Same adapter; correct Key / balance / pack on the vendor side |
| Subscription uses a **different** Base URL + Key + protocol face (e.g. 阿里 Token Plan `sk-sp-` + token-plan host; 方舟 Coding Plan URL ≠ Seedance data plane) | **Do not mix.** Current adapters target open-platform async endpoints (`dashscope.aliyuncs.com`, `ark.cn-beijing.volces.com`, `api.minimax.chat`). Plan keys/URLs usually will not work as a drop-in `base_url` swap |
| Wrong Key on right URL (or vice versa) | 401/403, or silent pay-as-you-go charges while believing you are on a plan |

**Configure Film with the open-platform (pay-as-you-go) Key and the Base URL that matches that vendor’s native video task API** (Settings defaults). A separate “plan adapter” is out of scope unless that plan endpoint is protocol-identical to the native async API.

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
