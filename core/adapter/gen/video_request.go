package gen

import (
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

// ChainModeToVideoMode maps Film long-video chain_mode → capability catalog token.
// Used by UI / capability checks; not read from HTTP body.mode.
func ChainModeToVideoMode(chainMode string) port.VideoMode {
	switch strings.ToLower(strings.TrimSpace(chainMode)) {
	case "reference_r2v":
		return port.VideoModeR2V
	case "reference_v2v", "v2v", "video_continue", "continuation":
		return port.VideoModeV2V
	case "first_last":
		return port.VideoModeFLF
	case "keyframe_only", "last_frame":
		return port.VideoModeI2V
	case "t2v", "text":
		return port.VideoModeT2V
	default:
		return ""
	}
}

// DeriveVideoMode maps filled media slots → capability catalog token (t2v|i2v|flf|r2v|v2v).
// Call after ValidateVideoRequest so exclusive groups are already enforced.
func DeriveVideoMode(req port.VideoRequest) port.VideoMode {
	hasFirst := strings.TrimSpace(req.FirstFrame) != ""
	hasLast := strings.TrimSpace(req.LastFrame) != ""
	hasRefs := len(filterNonEmptyURLs(req.ReferenceImages)) > 0
	hasVideos := strings.TrimSpace(req.SourceVideo) != "" || len(filterNonEmptyURLs(req.ReferenceVideos)) > 0
	switch {
	case hasVideos && !hasFirst && !hasLast:
		return port.VideoModeV2V
	case hasRefs && !hasFirst && !hasLast:
		return port.VideoModeR2V
	case hasFirst && hasLast:
		return port.VideoModeFLF
	case hasFirst:
		return port.VideoModeI2V
	default:
		return port.VideoModeT2V
	}
}

// ParseVideoRequest normalizes a Film / HTTP video body into port.VideoRequest.
// Staged asset ids (source_asset_id, reference_asset_ids, …) are preferred over aliases.
// Task kind is the filled role set — body.mode is ignored.
func ParseVideoRequest(body map[string]any) (port.VideoRequest, error) {
	if body == nil {
		return port.VideoRequest{}, fmt.Errorf("video body is required")
	}
	firstFrame := first(
		strVal(body["source_asset_id"]),
		strVal(body["start_asset_id"]),
		strVal(body["first_frame"]),
		strVal(body["first_frame_asset_id"]),
	)
	lastFrame := first(
		strVal(body["tail_asset_id"]),
		strVal(body["end_asset_id"]),
		strVal(body["last_frame"]),
		strVal(body["last_frame_asset_id"]),
		strVal(body["end_frame_asset_id"]),
	)
	refImages := bodyStringIDs(body["reference_asset_ids"])
	if len(refImages) == 0 {
		refImages = bodyStringIDs(body["reference_images"])
	}
	refVideos := bodyStringIDs(body["reference_video_asset_ids"])
	if len(refVideos) == 0 {
		refVideos = bodyStringIDs(body["reference_videos"])
	}
	sourceVideo := first(
		strVal(body["source_video_asset_id"]),
		strVal(body["video_asset_id"]),
		strVal(body["source_video"]),
	)
	if sourceVideo != "" {
		// Prefer explicit source as first reference video slot.
		out := make([]string, 0, 1+len(refVideos))
		out = append(out, sourceVideo)
		for _, v := range refVideos {
			if v != sourceVideo {
				out = append(out, v)
			}
		}
		refVideos = out
	}

	req := port.VideoRequest{
		Model:           strVal(body["model"]),
		Prompt:          strVal(body["prompt"]),
		NegativePrompt:  strVal(body["negative_prompt"]),
		DurationSec:     asFloat(body["duration"]),
		Size:            strVal(body["size"]),
		Resolution:      strVal(body["resolution"]),
		Ratio:           first(strVal(body["ratio"]), strVal(body["aspect_ratio"])),
		FPS:             asFloat(body["fps"]),
		NumFrames:       int(asFloat(body["num_frames"])),
		Seed:            int(asFloat(body["seed"])),
		FirstFrame:      firstFrame,
		LastFrame:       lastFrame,
		ReferenceImages: refImages,
		ReferenceVideos: refVideos,
		SourceVideo:     sourceVideo,
		Raw:             body,
	}
	if err := ValidateVideoRequest(req); err != nil {
		return port.VideoRequest{}, err
	}
	return req, nil
}

// ValidateVideoRequest enforces exclusive media groups (fail loud, no silent drops).
// Aligns with Make reject_video_roles: keyframes vs reference media vs source_video.
func ValidateVideoRequest(req port.VideoRequest) error {
	hasFirst := strings.TrimSpace(req.FirstFrame) != ""
	hasLast := strings.TrimSpace(req.LastFrame) != ""
	hasRefs := len(filterNonEmptyURLs(req.ReferenceImages)) > 0
	hasVideos := strings.TrimSpace(req.SourceVideo) != "" || len(filterNonEmptyURLs(req.ReferenceVideos)) > 0

	if hasLast && !hasFirst {
		return fmt.Errorf("last_frame requires first_frame / source_asset_id")
	}
	if (hasFirst || hasLast) && (hasRefs || hasVideos) {
		return fmt.Errorf("first/last frames cannot be combined with reference images or videos (mutually exclusive)")
	}
	return nil
}

// HasConditionedVideoMedia reports whether the request needs an edit/animate path
// (Make: /api/videos/edits) rather than pure text generations.
func HasConditionedVideoMedia(req port.VideoRequest) bool {
	return strings.TrimSpace(req.FirstFrame) != "" ||
		strings.TrimSpace(req.LastFrame) != "" ||
		len(filterNonEmptyURLs(req.ReferenceImages)) > 0 ||
		strings.TrimSpace(req.SourceVideo) != "" ||
		len(filterNonEmptyURLs(req.ReferenceVideos)) > 0
}

// NormalizeVideoBody parses and returns a body suitable for Make passthrough.
// Does not inject body.mode — Make derives task kind from media roles.
func NormalizeVideoBody(body map[string]any) (map[string]any, port.VideoRequest, error) {
	req, err := ParseVideoRequest(body)
	if err != nil {
		return nil, req, err
	}
	out := cloneMap(body)
	delete(out, "mode")
	delete(out, "film_protocol")
	delete(out, "film_modes")
	delete(out, "film_v2v_slot")
	return out, req, nil
}
