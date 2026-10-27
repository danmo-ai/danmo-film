package gen

import (
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

// ComposeModeToImageMode maps Film UI compose modes → port.ImageMode.
func ComposeModeToImageMode(compose string) port.ImageMode {
	switch strings.ToLower(strings.TrimSpace(compose)) {
	case "img2img", "i2i", "image2image", "reference":
		return port.ImageModeI2I
	case "edit", "instruct", "instruction":
		return port.ImageModeEdit
	case "inpaint":
		return port.ImageModeInpaint
	case "text2img", "t2i", "txt2img", "":
		return port.ImageModeT2I
	default:
		return port.ImageMode(compose)
	}
}

func imageModeFromBody(body map[string]any) port.ImageMode {
	if m := strings.ToLower(strings.TrimSpace(strVal(body["mode"]))); m != "" {
		switch m {
		case "t2i", "text2img", "txt2img":
			return port.ImageModeT2I
		case "i2i", "img2img", "image2image":
			return port.ImageModeI2I
		case "edit", "instruct":
			return port.ImageModeEdit
		case "inpaint":
			return port.ImageModeInpaint
		default:
			return port.ImageMode(m)
		}
	}
	if m := ComposeModeToImageMode(strVal(body["compose_mode"])); m != "" && strVal(body["compose_mode"]) != "" {
		return m
	}
	meta, _ := body["metadata"].(map[string]any)
	if meta != nil {
		if m := ComposeModeToImageMode(strVal(meta["compose_mode"])); strVal(meta["compose_mode"]) != "" {
			return m
		}
		if m := strings.ToLower(strings.TrimSpace(strVal(meta["image_mode"]))); m != "" {
			return ComposeModeToImageMode(m)
		}
	}
	return ""
}

func inferImageMode(source string, refs []string, mask string) port.ImageMode {
	if mask != "" && source != "" {
		return port.ImageModeInpaint
	}
	if len(refs) > 0 || source != "" {
		// Prefer i2i when refs present; single source without mask → i2i/edit ambiguous → i2i.
		return port.ImageModeI2I
	}
	return port.ImageModeT2I
}

// ParseImageRequest normalizes Film / HTTP image bodies into port.ImageRequest.
func ParseImageRequest(body map[string]any) (port.ImageRequest, error) {
	if body == nil {
		return port.ImageRequest{}, fmt.Errorf("image body is required")
	}
	source := first(
		strVal(body["source_asset_id"]),
		strVal(body["image_asset_id"]),
		strVal(body["init_asset_id"]),
	)
	mask := first(strVal(body["mask_asset_id"]), strVal(body["mask_id"]))
	refs := bodyStringIDs(body["reference_asset_ids"])
	if len(refs) == 0 {
		refs = bodyStringIDs(body["reference_images"])
	}
	// Single reference path also used as source for i2i.
	if source == "" && len(refs) == 1 {
		source = refs[0]
	}

	explicit := imageModeFromBody(body)
	mode := explicit
	if mode == "" {
		mode = inferImageMode(source, refs, mask)
	}
	if explicit != "" {
		switch mode {
		case port.ImageModeT2I:
			source, mask = "", ""
			refs = nil
		case port.ImageModeI2I:
			mask = ""
		case port.ImageModeEdit:
			mask = ""
			// edit keeps source; refs optional
		}
	}

	req := port.ImageRequest{
		Model:           strVal(body["model"]),
		Prompt:          strVal(body["prompt"]),
		NegativePrompt:  strVal(body["negative_prompt"]),
		Mode:            mode,
		Size:            strVal(body["size"]),
		Seed:            int(asFloat(body["seed"])),
		SourceID:        source,
		MaskID:          mask,
		ReferenceImages: refs,
		Raw:             body,
	}
	if err := ValidateImageRequest(req); err != nil {
		return port.ImageRequest{}, err
	}
	return req, nil
}

// ValidateImageRequest enforces per-mode required slots.
func ValidateImageRequest(req port.ImageRequest) error {
	hasSource := strings.TrimSpace(req.SourceID) != ""
	hasMask := strings.TrimSpace(req.MaskID) != ""
	hasRefs := len(filterNonEmptyURLs(req.ReferenceImages)) > 0
	switch req.Mode {
	case port.ImageModeT2I:
		if hasSource || hasMask || hasRefs {
			return fmt.Errorf("image mode t2i must not include source/reference/mask")
		}
		if strings.TrimSpace(req.Prompt) == "" {
			return fmt.Errorf("image mode t2i requires prompt")
		}
		return nil
	case port.ImageModeI2I:
		if !hasSource && !hasRefs {
			return fmt.Errorf("image mode i2i requires source_asset_id or reference_asset_ids")
		}
		return nil
	case port.ImageModeEdit:
		if !hasSource {
			return fmt.Errorf("image mode edit requires source_asset_id")
		}
		if strings.TrimSpace(req.Prompt) == "" {
			return fmt.Errorf("image mode edit requires prompt")
		}
		return nil
	case port.ImageModeInpaint:
		if !hasSource || !hasMask {
			return fmt.Errorf("image mode inpaint requires source_asset_id and mask_asset_id")
		}
		return nil
	default:
		return fmt.Errorf("unsupported image mode %q (want t2i|i2i|edit|inpaint)", req.Mode)
	}
}

// NormalizeImageBody sets body["mode"] after parse (Make / passthrough).
func NormalizeImageBody(body map[string]any) (map[string]any, port.ImageRequest, error) {
	req, err := ParseImageRequest(body)
	if err != nil {
		return nil, req, err
	}
	out := cloneMap(body)
	out["mode"] = string(req.Mode)
	return out, req, nil
}
