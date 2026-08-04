"""Danmo Film — long-form video workbench API."""

from __future__ import annotations

import os
from pathlib import Path

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles

from backend.api.routes import long_video_projects, providers, script_parse
from backend.runtime_paths import user_data_dir, workspace_dir


def create_app() -> FastAPI:
    app = FastAPI(title="Danmo Film", version="0.1.0")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/api/system/health")
    def health():
        return {
            "ok": True,
            "product": "danmo-film",
            "user_data_dir": str(user_data_dir()),
            "workspace_dir": str(workspace_dir()),
        }

    app.include_router(long_video_projects.router)
    app.include_router(script_parse.router)
    app.include_router(providers.router)

    dist = Path(__file__).resolve().parents[1] / "out" / "frontend" / "dist"
    if dist.is_dir():
        app.mount("/assets", StaticFiles(directory=dist / "assets"), name="assets")

        @app.get("/{full_path:path}")
        def spa(full_path: str = ""):
            index = dist / "index.html"
            if index.is_file():
                return FileResponse(index)
            return {"detail": "frontend not built"}

    return app


app = create_app()


def main() -> None:
    import uvicorn

    host = os.environ.get("DQ_HTTP_HOST") or os.environ.get("DANMO_FILM_HOST") or "127.0.0.1"
    port = int(os.environ.get("DQ_BACKEND_PORT") or os.environ.get("DANMO_FILM_PORT") or "7803")
    uvicorn.run("backend.main:app", host=host, port=port, reload=os.environ.get("FILM_RELOAD") == "1")


if __name__ == "__main__":
    main()
