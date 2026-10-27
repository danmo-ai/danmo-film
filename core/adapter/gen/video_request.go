package gen

import (
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

// ChainModeToVideoMode maps Film long-video chain_mode → port.VideoMode.
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

func videoModeFromBody(body map[string]any) port.VideoMode {
	if m := port.VideoMode(strings.ToLower(strings.TrimSpace(strVal(body["mode"])))); m != "" {
		return m
	}
	if m := ChainModeToVideoMode(strVal(body["chain_mode"])); m != "" {
		return m
	}
	meta, _ := body["metadata"].(map[string]any)
	if meta != nil {
		if m := ChainModeToVideoMode(first(strVal(meta["long_video_chain_mode"]), strVal(meta["chain_mode"]))); m != "" {
			return m
		}
		if m := port.VideoMode(strings.ToLower(strings.TrimSpace(strVal(meta["video_mode"])))); m != "" {
			return m
		}
	}
	return ""
}

func inferVideoMode(firstFrame, lastFrame string, refImages, refVideos []string, sourceVideo string) port.VideoMode {
	hasVideo := sourceVideo != "" || len(refVideos) > 0
	hasRefs := len(refImages) > 0
	hasFirst := firstFrame != ""
	hasLast := lastFrame != ""
	switch {
	case hasVideo && !hasFirst && !hasLast:
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

	explicit := videoModeFromBody(body)
	mode := explicit
	if mode == "" {
		mode = inferVideoMode(firstFrame, lastFrame, refImages, refVideos, sourceVideo)
	}
	// Explicit mode selects the legal media group. Film may still attach a keyframe
	// source alongside reference_r2v; drop the conflicting slots rather than 400.
	if explicit != "" {
		switch mode {
		case port.VideoModeR2V, port.VideoModeV2V:
			firstFrame, lastFrame = "", ""
		case port.VideoModeI2V:
			lastFrame = ""
			refImages, refVideos, sourceVideo = nil, nil, ""
		case port.VideoModeFLF:
			refImages, refVideos, sourceVideo = nil, nil, ""
		case port.VideoModeT2V:
			firstFrame, lastFrame = "", ""
			refImages, refVideos, sourceVideo = nil, nil, ""
		}
	}

	req := port.VideoRequest{
		Model:           strVal(body["model"]),
		Prompt:          strVal(body["prompt"]),
		NegativePrompt:  strVal(body["negative_prompt"]),
		Mode:            mode,
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

// ValidateVideoRequest enforces exclusive media groups per mode (fail loud).
func ValidateVideoRequest(req port.VideoRequest) error {
	hasFirst := strings.TrimSpace(req.FirstFrame) != ""
	hasLast := strings.TrimSpace(req.LastFrame) != ""
	hasRefs := len(filterNonEmptyURLs(req.ReferenceImages)) > 0
	hasVideos := strings.TrimSpace(req.SourceVideo) != "" || len(filterNonEmptyURLs(req.ReferenceVideos)) > 0

	switch req.Mode {
	case port.VideoModeT2V:
		if hasFirst || hasLast || hasRefs || hasVideos {
			return fmt.Errorf("video mode t2v must not include frame/reference media")
		}
		return nil
	case port.VideoModeI2V:
		if !hasFirst {
			return fmt.Errorf("video mode i2v requires source_asset_id/first_frame")
		}
		if hasRefs || hasVideos {
			return fmt.Errorf("video mode i2v cannot mix reference images/videos with first_frame")
		}
		return nil
	case port.VideoModeFLF:
		if !hasFirst || !hasLast {
			return fmt.Errorf("video mode flf requires first_frame and last_frame")
		}
		if hasRefs || hasVideos {
			return fmt.Errorf("video mode flf cannot mix reference images/videos with first/last frames")
		}
		return nil
	case port.VideoModeR2V:
		if !hasRefs {
			return fmt.Errorf("video mode r2v requires reference_asset_ids")
		}
		if hasFirst || hasLast {
			return fmt.Errorf("video mode r2v cannot mix first/last frames with reference images (mutually exclusive)")
		}
		return nil
	case port.VideoModeV2V:
		if !hasVideos {
			return fmt.Errorf("video mode v2v requires source_video_asset_id or reference_video_asset_ids")
		}
		if hasFirst || hasLast {
			return fmt.Errorf("video mode v2v cannot mix first/last frames with reference video (mutually exclusive)")
		}
		return nil
	default:
		return fmt.Errorf("unsupported video mode %q (want t2v|i2v|flf|r2v|v2v)", req.Mode)
	}
}

// NormalizeVideoBody ensures body["mode"] is set after ParseVideoRequest (for Make passthrough).
func NormalizeVideoBody(body map[string]any) (map[string]any, port.VideoRequest, error) {
	req, err := ParseVideoRequest(body)
	if err != nil {
		return nil, req, err
	}
	out := cloneMap(body)
	out["mode"] = string(req.Mode)
	return out, req, nil
}
