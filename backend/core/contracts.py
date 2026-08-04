"""Film workbench DTOs (script-parse / storyboard / project activity)."""

from __future__ import annotations

import secrets
from typing import Any, Literal, Union

from pydantic import BaseModel, Field


ChatMessageContent = Union[str, list[dict[str, Any]]]


class ChatMessage(BaseModel):
    role: Literal["system", "user", "assistant"]
    content: ChatMessageContent


def new_parse_run_id() -> str:
    return "prun_" + secrets.token_hex(12)


class LongVideoPlanDTO(BaseModel):
    target_duration_sec: float
    initial_duration_sec: float
    segment_extend_sec: float
    reference_duration_sec: float
    extend_pass_count: int
    total_segments: int
    segment_durations_sec: list[float]
    narrative_budget: str


class ScriptParseDecomposeRequest(BaseModel):
    script_text: str
    title: str = ""
    locale: str = ""
    long_video_project_id: str = ""
    model: str = ""


class ScriptParseExpandRequest(BaseModel):
    script_artifact: dict[str, Any]
    locale: str = ""
    target_duration_sec: float = 60.0
    segment_duration_sec: float = 5.0
    max_clip_sec: float = 10.0
    long_video_project_id: str = ""
    model: str = ""
    beat_indices: list[int] = Field(default_factory=list)


class ScriptParseExpandBeatRequest(BaseModel):
    script_artifact: dict[str, Any]
    beat_index: int = Field(ge=0)
    existing_shots: list[dict[str, Any]] = Field(default_factory=list)
    locale: str = ""
    target_duration_sec: float = 60.0
    segment_duration_sec: float = 5.0
    max_clip_sec: float = 10.0
    long_video_project_id: str = ""
    model: str = ""


class LongVideoChapterParsePhaseDTO(BaseModel):
    phase: str
    message: str = ""


class LongVideoChapterSceneDTO(BaseModel):
    order: int
    title: str = ""
    beat: str = ""


class LongVideoParseQualityIssueDTO(BaseModel):
    code: str
    message: str
    severity: Literal["warning", "critical"] = "warning"
    shot_index: int | None = None
    beat_index: int | None = None


class LongVideoCharacterLookDTO(BaseModel):
    id: str
    label: str = "默认"
    body: str = ""


class LongVideoCharacterDTO(BaseModel):
    id: str
    name: str
    looks: list[LongVideoCharacterLookDTO] = Field(default_factory=list)
    default_look_id: str = ""


class LongVideoSceneLookDTO(BaseModel):
    id: str
    label: str = "默认"
    body: str = ""


class LongVideoSceneDTO(BaseModel):
    id: str
    name: str
    looks: list[LongVideoSceneLookDTO] = Field(default_factory=list)
    default_look_id: str = ""
    spatial_layout_json: dict = Field(default_factory=dict)
    grounding_panorama_asset_id: str = ""
    grounding_depth_asset_id: str = ""


class LongVideoShotCastLookDTO(BaseModel):
    character_id: str
    look_id: str


class LongVideoShotSceneLookDTO(BaseModel):
    scene_id: str
    look_id: str


class LongVideoStoryboardShotDTO(BaseModel):
    id: str = ""
    order: int = 0
    visual_prompt: str = ""
    motion_prompt: str = ""
    video_prompt: str = ""
    start_visual_prompt: str = ""
    end_visual_prompt: str = ""
    anchor_visual_prompt: str = ""
    segment_role: Literal[
        "establishing",
        "pre_anchor",
        "face_anchor",
        "post_anchor",
        "keyframe",
        "tail_continuation",
    ] = "keyframe"
    start_frame_mode: Literal["keyframe", "prev_segment_tail", "anchor_link"] = "keyframe"
    segment_group_id: str = ""
    segment_group_index: int = 0
    face_anchor_shot_id: str = ""
    flf_mode: Literal["none", "first_last", "continuation"] = "none"
    end_frame_sync_anchor: bool = False
    chain_mode: Literal["keyframe_only", "last_frame", "first_last", "reference_r2v"] | None = None
    scene_prompt: str = ""
    cast_looks: list[LongVideoShotCastLookDTO] = Field(default_factory=list)
    scene_look: LongVideoShotSceneLookDTO | None = None
    duration_sec: float | None = None
    first_frame_visibility: Literal["invisible", "silhouette", "partial", "full_face"] = "full_face"
    end_visibility: Literal["invisible", "silhouette", "partial", "full_face"] = "full_face"
    characters_on_screen: list[str] = Field(default_factory=list)
    clip_start_state: str = ""
    clip_end_state: str = ""
    first_frame_requirement: str = ""
    camera_zone_id: str = ""
    first_frame_strategy: Literal[
        "direct_reuse_portrait",
        "scene_composite",
        "reuse_prev_tail",
        "img2img_light",
        "t2i_from_grounding",
        "causal_generate",
    ] = "t2i_from_grounding"
    location: str = ""
    shot_size: str = ""
    narrative_beat_index: int | None = None
    shot_intent: str = ""
    narrative_role: str = ""
    camera_movement: str = ""
    lighting_key: str = ""
    is_establishing_empty: bool = False


class ScriptParseDecomposeResponse(BaseModel):
    chapter_title: str = ""
    synopsis: str
    mood: str = ""
    character_anchor: str = ""
    style_anchor: str = ""
    characters: list[LongVideoCharacterDTO] = Field(default_factory=list)
    scenes: list[LongVideoSceneDTO] = Field(default_factory=list)
    scene_beats: list[LongVideoChapterSceneDTO] = Field(default_factory=list)
    scene_count: int = 0
    script_artifact: dict[str, Any] = Field(default_factory=dict)
    llm_calls: int = 0
    parse_phases: list[LongVideoChapterParsePhaseDTO] = Field(default_factory=list)
    parse_run_id: str = ""
    long_video_project_id: str = ""


class ScriptParseExpandResponse(BaseModel):
    chapter_title: str = ""
    synopsis: str = ""
    mood: str = ""
    character_anchor: str = ""
    style_anchor: str = ""
    characters: list[LongVideoCharacterDTO] = Field(default_factory=list)
    scenes: list[LongVideoSceneDTO] = Field(default_factory=list)
    scene_beats: list[LongVideoChapterSceneDTO] = Field(default_factory=list)
    scene_count: int = 0
    shots: list[LongVideoStoryboardShotDTO] = Field(default_factory=list)
    script_artifact: dict[str, Any] = Field(default_factory=dict)
    llm_calls: int = 0
    parse_phases: list[LongVideoChapterParsePhaseDTO] = Field(default_factory=list)
    quality_warnings: list[str] = Field(default_factory=list)
    quality_issues: list[LongVideoParseQualityIssueDTO] = Field(default_factory=list)
    parse_run_id: str = ""
    long_video_project_id: str = ""


class LongVideoChapterAnalyzeRequest(BaseModel):
    chapter_text: str
    chapter_title: str = ""
    locale: str = ""
    target_duration_sec: float = 60.0
    segment_duration_sec: float = 5.0
    max_clip_sec: float = 10.0
    long_video_project_id: str = ""
    model: str = ""


class LongVideoChapterAnalyzeResponse(BaseModel):
    chapter_title: str
    synopsis: str
    mood: str = ""
    character_anchor: str
    style_anchor: str = ""
    characters: list[LongVideoCharacterDTO] = Field(default_factory=list)
    scenes: list[LongVideoSceneDTO] = Field(default_factory=list)
    scene_beats: list[LongVideoChapterSceneDTO] = Field(default_factory=list)
    scene_count: int = 0
    shots: list[LongVideoStoryboardShotDTO] = Field(default_factory=list)
    parse_phases: list[LongVideoChapterParsePhaseDTO] = Field(default_factory=list)
    quality_warnings: list[str] = Field(default_factory=list)
    quality_issues: list[LongVideoParseQualityIssueDTO] = Field(default_factory=list)
    llm_calls: int = 0
    parse_run_id: str = ""
    long_video_project_id: str = ""


class LongVideoProjectActivityDTO(BaseModel):
    id: str
    project_id: str
    category: str
    event_type: str
    phase: str = ""
    status: str = ""
    summary: str = ""
    task_id: str | None = None
    parse_run_id: str | None = None
    shot_id: str = ""
    detail: dict[str, Any] = Field(default_factory=dict)
    created_at: str


class LongVideoProjectActivityListResponse(BaseModel):
    items: list[LongVideoProjectActivityDTO] = Field(default_factory=list)
    total: int = 0


class LongVideoStoryboardRequest(BaseModel):
    prompt: str
    target_duration_sec: float = 60.0
    initial_duration_sec: float = 8.0
    segment_extend_sec: float = 8.0
    segment_duration_sec: float = 5.0
    reference_duration_sec: float = 3.0
    style_positive: str = ""
    locale: str = ""
    use_shot_plan: bool = True
    source_mode: Literal["brief", "chapter"] = "brief"
    scene_beats: list[str] = Field(default_factory=list)
    prebuilt_character_anchor: str = ""
    prebuilt_style_anchor: str = ""
    prebuilt_scenes: list[LongVideoSceneDTO] = Field(default_factory=list)


class LongVideoStoryboardResponse(BaseModel):
    character_anchor: str
    opening_prompt: str
    segment_prompts: list[str]
    segment_count: int
    plan: LongVideoPlanDTO
    beat_sheet: list[str]
    llm_calls: int
    shots: list[LongVideoStoryboardShotDTO] = Field(default_factory=list)
    characters: list[LongVideoCharacterDTO] = Field(default_factory=list)
    scenes: list[LongVideoSceneDTO] = Field(default_factory=list)
    style_anchor: str = ""
