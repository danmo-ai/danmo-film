"""Film-owned asset store — system of record for images / clips / finals.

Providers (Make, cloud) only supply bytes; they are not Film's gallery.
"""

from __future__ import annotations

import json
import shutil
import sqlite3
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Optional


def _now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def is_film_asset_id(asset_id: str) -> bool:
    return str(asset_id or "").startswith("fas_")


class FilmAssetStore:
    def __init__(self, db_path: Path, files_root: Path) -> None:
        self._db_path = Path(db_path)
        self._root = Path(files_root)
        self._root.mkdir(parents=True, exist_ok=True)
        self._db_path.parent.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()
        self._local = threading.local()
        self._init_db()

    @property
    def files_root(self) -> Path:
        return self._root

    def _conn(self) -> sqlite3.Connection:
        if not hasattr(self._local, "conn") or self._local.conn is None:
            c = sqlite3.connect(str(self._db_path), check_same_thread=False)
            c.row_factory = sqlite3.Row
            c.execute("PRAGMA journal_mode=WAL")
            self._local.conn = c
        return self._local.conn

    def _init_db(self) -> None:
        with self._lock:
            conn = self._conn()
            conn.executescript(
                """
                CREATE TABLE IF NOT EXISTS film_assets (
                    id TEXT PRIMARY KEY,
                    kind TEXT NOT NULL,
                    mime_type TEXT NOT NULL,
                    file_path TEXT NOT NULL,
                    thumbnail_path TEXT,
                    source_provider TEXT,
                    provider_ref TEXT,
                    source_task_id TEXT,
                    metadata TEXT,
                    created_at TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_film_assets_kind ON film_assets(kind);
                CREATE INDEX IF NOT EXISTS idx_film_assets_created ON film_assets(created_at);
                CREATE INDEX IF NOT EXISTS idx_film_assets_provider_ref
                    ON film_assets(source_provider, provider_ref);
                """
            )
            conn.commit()

    def create_from_file(
        self,
        src_path: Path,
        *,
        kind: str,
        mime_type: str,
        source_provider: str = "",
        provider_ref: str = "",
        source_task_id: str = "",
        metadata: Optional[dict[str, Any]] = None,
    ) -> dict[str, Any]:
        src = Path(src_path)
        if not src.is_file():
            raise RuntimeError(f"Film asset source missing: {src}")
        aid = "fas_" + uuid.uuid4().hex[:24]
        ext = src.suffix or ".bin"
        dest = self._root / f"{aid}{ext}"
        shutil.copy2(src, dest)
        thumb: Optional[Path] = None
        if kind == "image" and mime_type.startswith("image/"):
            try:
                from PIL import Image

                with Image.open(dest) as im:
                    thumb = self._root / f"{aid}.thumb.webp"
                    im = im.convert("RGB")
                    im.thumbnail((512, 512))
                    im.save(thumb, format="WEBP", quality=80)
            except Exception:
                thumb = None
        created = _now()
        meta = dict(metadata or {})
        with self._lock:
            conn = self._conn()
            conn.execute(
                """
                INSERT INTO film_assets (
                    id, kind, mime_type, file_path, thumbnail_path,
                    source_provider, provider_ref, source_task_id, metadata, created_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    aid,
                    kind,
                    mime_type,
                    str(dest),
                    str(thumb) if thumb else None,
                    source_provider or None,
                    provider_ref or None,
                    source_task_id or None,
                    json.dumps(meta, ensure_ascii=False),
                    created,
                ),
            )
            conn.commit()
        return self.get(aid)  # type: ignore[return-value]

    def get(self, asset_id: str) -> Optional[dict[str, Any]]:
        with self._conn() as conn:
            row = conn.execute(
                "SELECT * FROM film_assets WHERE id = ?",
                (asset_id,),
            ).fetchone()
        if not row:
            return None
        return self._row_to_dict(row)

    def resolve_path(self, asset_id: str) -> Path:
        row = self.get(asset_id)
        if not row:
            raise RuntimeError(f"Film asset not found: {asset_id}")
        path = Path(str(row["file_path"]))
        if not path.is_file():
            raise RuntimeError(f"Film asset file missing on disk: {asset_id} → {path}")
        return path

    def list_assets(
        self,
        *,
        kind: str | None = None,
        limit: int = 100,
        offset: int = 0,
    ) -> list[dict[str, Any]]:
        lim = max(1, min(int(limit), 500))
        off = max(0, int(offset))
        sql = "SELECT * FROM film_assets"
        args: list[Any] = []
        if kind:
            sql += " WHERE kind = ?"
            args.append(kind)
        sql += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
        args.extend([lim, off])
        with self._conn() as conn:
            rows = conn.execute(sql, tuple(args)).fetchall()
        return [self._row_to_dict(r) for r in rows]

    def find_by_provider_ref(self, provider_id: str, provider_ref: str) -> Optional[dict[str, Any]]:
        with self._conn() as conn:
            row = conn.execute(
                """
                SELECT * FROM film_assets
                WHERE source_provider = ? AND provider_ref = ?
                ORDER BY created_at DESC LIMIT 1
                """,
                (provider_id, provider_ref),
            ).fetchone()
        return self._row_to_dict(row) if row else None

    def delete(self, asset_id: str) -> bool:
        row = self.get(asset_id)
        if not row:
            return False
        with self._lock:
            conn = self._conn()
            conn.execute("DELETE FROM film_assets WHERE id = ?", (asset_id,))
            conn.commit()
        for key in ("file_path", "thumbnail_path"):
            p = row.get(key)
            if p:
                try:
                    Path(str(p)).unlink(missing_ok=True)
                except OSError:
                    pass
        return True

    def _row_to_dict(self, row: sqlite3.Row) -> dict[str, Any]:
        meta_raw = row["metadata"]
        try:
            meta = json.loads(meta_raw) if meta_raw else {}
        except json.JSONDecodeError:
            meta = {}
        return {
            "id": row["id"],
            "kind": row["kind"],
            "mime_type": row["mime_type"],
            "file_path": row["file_path"],
            "thumbnail_path": row["thumbnail_path"],
            "source_provider": row["source_provider"],
            "provider_ref": row["provider_ref"],
            "source_task_id": row["source_task_id"],
            "metadata": meta,
            "created_at": row["created_at"],
            "thumbnail_url": f"/api/assets/{row['id']}/thumbnail",
            "file_url": f"/api/assets/{row['id']}/file",
        }
