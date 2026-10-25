# Danmo Film

Long-form video workbench for the Danmo suite: script → storyboard → timeline → segment generation → assemble.

The API is **Go**. The UI is Vue. **Providers are peers.** Danmo Make and cloud vendors (火山引擎 / 阿里云 / MiniMax) supply LLM / image / short-clip APIs. Video jobs route by `provider_id/model_id` like LLM. Film downloads results, stores them in its own asset library (`fas_*`), then merges and manages media locally. Finals are **not** uploaded back to Make.

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
    "generation_default_model": "volcengine/doubao-seedance-1-0-pro-250528",
    "generation_profiles": [
      {
        "id": "volcengine",
        "label": "火山引擎",
        "vendor": "volcengine",
        "base_url": "https://ark.cn-beijing.volces.com",
        "api_key": "",
        "enabled": true,
        "models": [{ "id": "doubao-seedance-1-0-pro-250528", "enabled": true }]
      }
    ]
  }
}
```

**Video** presets (`generation_profiles`): `danmo-make` | `volcengine` | `dashscope` | `minimax`.  
**Image** presets (`image_profiles`): same four plus `openai` (Images API).  
Same preset can be added multiple times (rename each profile for Plan vs API). Create / keyframe pickers use `profile_id/model_id`. Saving in the UI applies immediately (no restart).

Use each vendor’s **open-platform** API Key + native Base URL (Settings defaults). Subscription / Token Plan keys often need a second profile. See **Video providers** in [`AGENTS.md`](AGENTS.md).

## Layout

- `cmd/danmo-film/` — API process
- `core/port/` — `LLMProvider`, `GenerationProvider`, stores
- `core/adapter/llm/` — Chat Completions / Responses / Anthropic
- `core/adapter/gen/` — Make + cloud video adapters (`gen.Resolve` factory)
- `core/store/sqlite/` — Film asset SoR + projects
- `core/service/scriptparse/` — 4-pass script-parse pipeline
- `frontend/` — Film UI (long-video studio)
