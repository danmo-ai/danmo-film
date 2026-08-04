# Danmo Film

Long-form video workbench for the Danmo suite: script → storyboard → timeline → segment generation.

Danmo **Make** remains the local generation engine (image / video / audio). Film orchestrates narrative production and calls:

- **Danmo Make API** (default `http://127.0.0.1:7800`) for generation / assets / tasks
- **Cloud vendor APIs** via `backend/providers/cloud/` (stubs today — fail loud until implemented)
- **Assemble** — `POST /api/long-video/assemble` stitches Make segment assets (no Make long-video API)

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

Optional: point LLM/script-parse at Make’s OpenAI-compatible chat:

```bash
export FILM_MAKE_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_MODEL=<your-local-llm-id>
```

Settings file: `~/.danmo-film/config/settings.json`.

## Layout

- `backend/film/` — domain helpers (timing, parse quality, activity)
- `backend/llm/` — script-parse / storyboard LLM pipeline
- `backend/providers/` — Make + cloud generation / LLM adapters
- `frontend/` — Film UI (long-video studio)
