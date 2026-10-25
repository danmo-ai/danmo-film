# Danmo Film

Long-form video workbench for the Danmo suite: script → storyboard → timeline → segment generation → assemble.

The API is **Go**. The UI is Vue. **Providers are peers.** Danmo Make and cloud vendors (Volcengine Seedance, DashScope 万相, MiniMax Hailuo) supply LLM / image / short-clip APIs. Film downloads results, stores them in its own asset library (`fas_*`), then merges and manages media locally. Finals are **not** uploaded back to Make.

## Dev ports

| | Backend | Frontend |
|--|---------|----------|
| Make | 7800 | 5800 |
| Film | **7803** | **5803** |

## Quick start

```bash
make dev    # Go API :7803 + Vite :5803
make stop
```

Or split:

```bash
make backend          # API only
make frontend-dev     # Vite only (after npm install)
```

Optional: point generation / LLM at Make (or any OpenAI-compatible base):

```bash
export FILM_MAKE_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_BASE_URL=http://127.0.0.1:7800
export FILM_LLM_MODEL=<your-local-llm-id>
export FILM_GENERATION_PROVIDER=danmo-make   # or volcengine | dashscope | minimax
```

Settings file: `~/.danmo-film/config/settings.json` (also editable in UI **Settings**).

```json
{
  "providers": {
    "llm": { "protocol": "openai", "base_url": "http://127.0.0.1:7800", "model": "" },
    "generation": { "provider": "danmo-make", "base_url": "http://127.0.0.1:7800" },
    "volcengine": { "api_key": "", "model": "doubao-seedance-1-0-pro-250528" },
    "dashscope": { "api_key": "", "model": "wan2.2-t2v-plus" },
    "minimax": { "api_key": "", "model": "MiniMax-Hailuo-02" }
  }
}
```

`generation.provider`: `danmo-make` | `volcengine` | `dashscope` | `minimax`. Saving in the UI applies immediately (no restart).

## Layout

- `cmd/danmo-film/` — API process
- `core/port/` — `LLMProvider`, `GenerationProvider`, stores
- `core/adapter/llm/` — Chat Completions / Responses / Anthropic
- `core/adapter/gen/` — Make + cloud video adapters
- `core/store/sqlite/` — Film asset SoR + projects
- `core/service/scriptparse/` — 4-pass script-parse pipeline
- `frontend/` — Film UI (long-video studio)
