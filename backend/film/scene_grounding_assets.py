"""Scene grounding asset helpers — depth encoding stays on Danmo Make (MLX/CUDA)."""

from __future__ import annotations

from pathlib import Path
from typing import Any


def depth_asset_from_source_image(
    source_path: Path,
    *,
    asset_store: Any,
    depth_bundle_root: Path,
    width: int = 1024,
    height: int = 1024,
    source_asset_id: str = "",
    group_id: str | None = None,
    on_log: Any = None,
) -> str:
    raise RuntimeError(
        "Scene-grounding depth encoding runs on Danmo Make (depth-pro MLX/CUDA), not in Danmo Film. "
        "Upload/select a depth asset produced by Make, or call Make tooling until Film proxies this."
    )
