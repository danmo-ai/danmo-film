# AGENTS.md — Danmo Film

Operational guide for contributors and coding agents.

## What this is

**Danmo Film** — long-form video workbench (script parse, storyboard, project timeline). Generation executes on **Danmo Make** or cloud vendors through `backend/providers/`.

| Product | Repo | Role |
|---------|------|------|
| Make | `danmo-ai/danmo-make` | Local image / video / audio studio (MLX + CUDA) |
| Film | `danmo-ai/danmo-film` | Long-form narrative workbench |
| Work | `danmo-ai/danmo-work` | Long-horizon AI work agent |
| Inbox | `danmo-ai/danmo-inbox` | AI email stream organization |

## Quick reference

| Item | Location |
|------|----------|
| API entry | `backend/main.py` |
| Frontend | `frontend/` (Vite :5803) |
| Providers | `backend/providers/` |
| Film domain | `backend/film/` |
| Script-parse LLM | `backend/llm/` |
| User data | `~/.danmo-film/` |
| Dev ports | API **7803**, UI **5803** |

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
3. Generation contracts for `/api/videos/long-generations` stay on Make; Film is a client.
4. Product name: **Danmo Film** / `danmo-film-*`.
