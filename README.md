# Danmo Film

Long-form video workbench for the Danmo suite: script → storyboard → timeline → segment generation → assemble.

**Providers are peers.** Danmo Make and cloud vendors supply LLM / image / short-clip APIs the same way. Film downloads results, stores them in its own asset library (`fas_*`), then merges and manages media locally. Finals are **not** uploaded back to Make.

## Dev ports

| | Backend | Frontend |
|--|---------|----------|
| Make | 7800 | 5800 |
| Film | **7803** | **5803** |

## Quick start

```bash
python3.11 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
make install-frontend

# Terminal A — Film API
make backend

# Terminal B — Film UI
make frontend
```

Optional: point generation / LLM at Make (or any OpenAI-compatible base):

```bash
export FILM_MAKE_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_MODEL=<your-local-llm-id>
```

Settings file: `~/.danmo-film/config/settings.json`.

## Layout

- `backend/film/` — domain helpers (timing, parse quality, stitch)
- `backend/llm/` — script-parse / storyboard LLM pipeline
- `backend/providers/` — Make + cloud generation / LLM adapters (peers)
- `backend/persistence/film_asset_store.py` — Film asset SoR
- `frontend/` — Film UI (long-video studio)
