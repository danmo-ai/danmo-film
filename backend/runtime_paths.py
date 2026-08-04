"""Filesystem layout for Danmo Film control plane + workspace DB."""

from __future__ import annotations

import os
from pathlib import Path


def user_data_dir() -> Path:
    override = (os.environ.get("DANMO_FILM_USER_DATA_DIR") or "").strip()
    if override:
        return Path(override).expanduser().resolve()
    return (Path.home() / ".danmo-film").resolve()


def workspace_dir() -> Path:
    override = (os.environ.get("DANMO_FILM_WORKSPACE") or "").strip()
    if override:
        return Path(override).expanduser().resolve()
    # Dev default: repo-local db/
    repo = Path(__file__).resolve().parents[1]
    return (repo / "db").resolve()


def film_db_path() -> Path:
    path = workspace_dir() / "film.db"
    path.parent.mkdir(parents=True, exist_ok=True)
    return path


def film_assets_root() -> Path:
    path = workspace_dir() / "assets"
    path.mkdir(parents=True, exist_ok=True)
    return path
