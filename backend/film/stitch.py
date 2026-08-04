"""Concatenate segment MP4s with ffmpeg (Film-owned assemble; Make has no long-video API)."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
from pathlib import Path


def require_ffmpeg() -> str:
    exe = shutil.which("ffmpeg")
    if not exe:
        raise RuntimeError("ffmpeg not found on PATH (required to assemble long-video segments)")
    return exe


def stitch_segment_paths(
    segment_paths: list[Path],
    *,
    output: Path,
    overlap_frames: int = 0,
    fps: float = 16.0,
) -> Path:
    """Stitch ordered segment videos. ``overlap_frames`` reserved; concat demuxer for v1."""
    _ = overlap_frames, fps
    if not segment_paths:
        raise RuntimeError("assemble: no segment videos")
    out = Path(output)
    out.parent.mkdir(parents=True, exist_ok=True)
    if len(segment_paths) == 1:
        shutil.copy2(segment_paths[0], out)
        return out

    ffmpeg = require_ffmpeg()
    with tempfile.TemporaryDirectory(prefix="film_stitch_") as tmp:
        list_path = Path(tmp) / "concat.txt"
        lines = []
        for p in segment_paths:
            if not Path(p).is_file():
                raise RuntimeError(f"assemble: missing segment file {p}")
            # ffmpeg concat demuxer requires escaped single quotes
            path = str(Path(p).resolve()).replace("'", "'\\''")
            lines.append(f"file '{path}'")
        list_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
        cmd = [
            ffmpeg,
            "-y",
            "-hide_banner",
            "-loglevel",
            "error",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            str(list_path),
            "-c",
            "copy",
            str(out),
        ]
        proc = subprocess.run(cmd, capture_output=True, text=True)
        if proc.returncode != 0:
            raise RuntimeError(f"ffmpeg stitch failed: {(proc.stderr or proc.stdout or '')[:800]}")
    if not out.is_file():
        raise RuntimeError(f"assemble: output missing after ffmpeg: {out}")
    return out
