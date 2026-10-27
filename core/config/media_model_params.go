package config

import (
	"fmt"
	"strings"
)

// Wire protocols for video adapters (settings-driven; no vendor version heuristics in adapters).
const (
	VideoProtocolDashMedia     = "dashscope_media"   // wan2.7+ / wan3 input.media
	VideoProtocolDashImgURL    = "dashscope_img_url" // wan2.6- input.img_url
	VideoProtocolMiniMaxH3     = "minimax_h3_content"
	VideoProtocolVolcContent   = "volcengine_content"
	VideoProtocolMakeNative    = "danmo_make"
)

// VideoModelParams is per-model video capability (providers.video_model_params).
type VideoModelParams struct {
	Modes              []string `json:"modes,omitempty"` // t2v|i2v|flf|r2v|v2v
	Protocol           string   `json:"protocol,omitempty"`
	// V2VSlot selects vendor media type for v2v: reference_video (default) | first_clip (wan2.7).
	V2VSlot            string  `json:"v2v_slot,omitempty"`
	MaxReferenceImages int     `json:"max_reference_images,omitempty"`
	MaxReferenceVideos int     `json:"max_reference_videos,omitempty"`
	MaxDurationSec     float64 `json:"max_duration_sec,omitempty"`
}

// ImageModelParams is per-model image capability (providers.image_model_params).
type ImageModelParams struct {
	Modes              []string `json:"modes,omitempty"` // t2i|i2i|edit|inpaint
	Protocol           string   `json:"protocol,omitempty"`
	MaxReferenceImages int      `json:"max_reference_images,omitempty"`
}

// VideoModelParamsMap returns providers.video_model_params keyed by bare model id.
func VideoModelParamsMap(s Settings) map[string]VideoModelParams {
	return parseVideoModelParamsMap(s.Map("providers")["video_model_params"])
}

// ImageModelParamsMap returns providers.image_model_params keyed by bare model id.
func ImageModelParamsMap(s Settings) map[string]ImageModelParams {
	return parseImageModelParamsMap(s.Map("providers")["image_model_params"])
}

func parseVideoModelParamsMap(raw any) map[string]VideoModelParams {
	out := map[string]VideoModelParams{}
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return out
	}
	for id, v := range m {
		id = strings.TrimSpace(id)
		row, ok := v.(map[string]any)
		if !ok || id == "" {
			continue
		}
		p := VideoModelParams{
			Modes:              stringSliceAny(row["modes"]),
			Protocol:           strings.TrimSpace(strAny(row["protocol"])),
			V2VSlot:            strings.TrimSpace(strAny(row["v2v_slot"])),
			MaxReferenceImages: intAny(row["max_reference_images"]),
			MaxReferenceVideos: intAny(row["max_reference_videos"]),
		}
		if f, ok := asFloat(row["max_duration_sec"]); ok {
			p.MaxDurationSec = f
		}
		out[id] = p
	}
	return out
}

func parseImageModelParamsMap(raw any) map[string]ImageModelParams {
	out := map[string]ImageModelParams{}
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return out
	}
	for id, v := range m {
		id = strings.TrimSpace(id)
		row, ok := v.(map[string]any)
		if !ok || id == "" {
			continue
		}
		out[id] = ImageModelParams{
			Modes:              stringSliceAny(row["modes"]),
			Protocol:           strings.TrimSpace(strAny(row["protocol"])),
			MaxReferenceImages: intAny(row["max_reference_images"]),
		}
	}
	return out
}

func stringSliceAny(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, raw := range t {
			s := strings.TrimSpace(fmt.Sprint(raw))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// VideoModelParamsToAny serializes the catalog for settings JSON.
func VideoModelParamsToAny(m map[string]VideoModelParams) map[string]any {
	out := make(map[string]any, len(m))
	for id, p := range m {
		row := map[string]any{}
		if len(p.Modes) > 0 {
			modes := make([]any, len(p.Modes))
			for i, s := range p.Modes {
				modes[i] = s
			}
			row["modes"] = modes
		}
		if p.Protocol != "" {
			row["protocol"] = p.Protocol
		}
		if p.V2VSlot != "" {
			row["v2v_slot"] = p.V2VSlot
		}
		if p.MaxReferenceImages > 0 {
			row["max_reference_images"] = p.MaxReferenceImages
		}
		if p.MaxReferenceVideos > 0 {
			row["max_reference_videos"] = p.MaxReferenceVideos
		}
		if p.MaxDurationSec > 0 {
			row["max_duration_sec"] = p.MaxDurationSec
		}
		if len(row) == 0 {
			continue
		}
		out[id] = row
	}
	return out
}

// ImageModelParamsToAny serializes image capability catalog.
func ImageModelParamsToAny(m map[string]ImageModelParams) map[string]any {
	out := make(map[string]any, len(m))
	for id, p := range m {
		row := map[string]any{}
		if len(p.Modes) > 0 {
			modes := make([]any, len(p.Modes))
			for i, s := range p.Modes {
				modes[i] = s
			}
			row["modes"] = modes
		}
		if p.Protocol != "" {
			row["protocol"] = p.Protocol
		}
		if p.MaxReferenceImages > 0 {
			row["max_reference_images"] = p.MaxReferenceImages
		}
		if len(row) == 0 {
			continue
		}
		out[id] = row
	}
	return out
}

// ResolveVideoModelParams looks up bare model id (also accepts profile/model refs).
func ResolveVideoModelParams(s Settings, modelRef string) VideoModelParams {
	bare := bareModelID(modelRef)
	if bare == "" {
		return VideoModelParams{}
	}
	return VideoModelParamsMap(s)[bare]
}

// ResolveImageModelParams looks up bare model id.
func ResolveImageModelParams(s Settings, modelRef string) ImageModelParams {
	bare := bareModelID(modelRef)
	if bare == "" {
		return ImageModelParams{}
	}
	return ImageModelParamsMap(s)[bare]
}

func bareModelID(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 && i+1 < len(ref) {
		return strings.TrimSpace(ref[i+1:])
	}
	return ref
}

// SupportsVideoMode reports whether params.Modes includes mode (empty Modes = unknown).
func (p VideoModelParams) SupportsVideoMode(mode string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" || len(p.Modes) == 0 {
		return len(p.Modes) == 0 // unknown → caller may allow vendor fail-loud
	}
	for _, m := range p.Modes {
		if strings.EqualFold(strings.TrimSpace(m), mode) {
			return true
		}
	}
	return false
}

// AssertSupportsVideoMode fails when modes are configured and mode is absent.
func (p VideoModelParams) AssertSupportsVideoMode(mode string) error {
	if len(p.Modes) == 0 {
		return nil
	}
	if p.SupportsVideoMode(mode) {
		return nil
	}
	return fmt.Errorf("model does not support video mode %q (supports: %s)", mode, strings.Join(p.Modes, ", "))
}

// AssertSupportsImageMode fails when modes are configured and mode is absent.
func (p ImageModelParams) AssertSupportsImageMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if len(p.Modes) == 0 {
		return nil
	}
	for _, m := range p.Modes {
		mm := strings.ToLower(strings.TrimSpace(m))
		if mm == mode || (mode == "i2i" && (mm == "img2img" || mm == "i2i")) {
			return nil
		}
	}
	return fmt.Errorf("model does not support image mode %q (supports: %s)", mode, strings.Join(p.Modes, ", "))
}

// HasConfiguredVideoParams is true when the model has an explicit catalog row.
func HasConfiguredVideoParams(s Settings, modelRef string) bool {
	bare := bareModelID(modelRef)
	_, ok := VideoModelParamsMap(s)[bare]
	return ok
}

// ApplyVideoParamsToModelRow merges catalog capabilities into a registry/list row.
func ApplyVideoParamsToModelRow(row map[string]any, p VideoModelParams) map[string]any {
	if len(p.Modes) == 0 && p.Protocol == "" && p.MaxReferenceImages == 0 {
		return row
	}
	out := map[string]any{}
	for k, v := range row {
		out[k] = v
	}
	if len(p.Modes) > 0 {
		out["modes"] = append([]string{}, p.Modes...)
	}
	if p.Protocol != "" {
		out["protocol"] = p.Protocol
	}
	params, _ := out["parameters"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	} else {
		cp := map[string]any{}
		for k, v := range params {
			cp[k] = v
		}
		params = cp
	}
	if len(p.Modes) > 0 {
		params["modes"] = append([]string{}, p.Modes...)
		params["supports_r2v"] = modeListContains(p.Modes, "r2v")
		params["supports_v2v"] = modeListContains(p.Modes, "v2v")
		params["supports_first_last"] = modeListContains(p.Modes, "flf")
		params["supports_flf"] = modeListContains(p.Modes, "flf")
		params["supports_t2v"] = modeListContains(p.Modes, "t2v")
		params["supports_i2v"] = modeListContains(p.Modes, "i2v")
	}
	if p.MaxReferenceImages > 0 {
		params["max_reference_images"] = p.MaxReferenceImages
	}
	if p.MaxReferenceVideos > 0 {
		params["max_reference_videos"] = p.MaxReferenceVideos
	}
	if p.MaxDurationSec > 0 {
		params["max_duration_sec"] = p.MaxDurationSec
	}
	if p.Protocol != "" {
		params["protocol"] = p.Protocol
	}
	if p.V2VSlot != "" {
		params["v2v_slot"] = p.V2VSlot
	}
	out["parameters"] = params
	return out
}

// ApplyImageParamsToModelRow merges image catalog into a registry row.
func ApplyImageParamsToModelRow(row map[string]any, p ImageModelParams) map[string]any {
	if len(p.Modes) == 0 && p.Protocol == "" && p.MaxReferenceImages == 0 {
		return row
	}
	out := map[string]any{}
	for k, v := range row {
		out[k] = v
	}
	if len(p.Modes) > 0 {
		out["modes"] = append([]string{}, p.Modes...)
	}
	if p.Protocol != "" {
		out["protocol"] = p.Protocol
	}
	params, _ := out["parameters"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	} else {
		cp := map[string]any{}
		for k, v := range params {
			cp[k] = v
		}
		params = cp
	}
	if len(p.Modes) > 0 {
		params["modes"] = append([]string{}, p.Modes...)
		params["supports_img2img"] = modeListContains(p.Modes, "i2i") || modeListContains(p.Modes, "img2img")
		params["supports_i2i"] = params["supports_img2img"]
		params["supports_edit"] = modeListContains(p.Modes, "edit")
		params["supports_inpaint"] = modeListContains(p.Modes, "inpaint")
	}
	if p.MaxReferenceImages > 0 {
		params["max_reference_images"] = p.MaxReferenceImages
	}
	if p.Protocol != "" {
		params["protocol"] = p.Protocol
	}
	out["parameters"] = params
	return out
}

func modeListContains(modes []string, want string) bool {
	want = strings.ToLower(want)
	for _, m := range modes {
		if strings.EqualFold(strings.TrimSpace(m), want) {
			return true
		}
	}
	return false
}

// SeedVideoModelParamsFromPresets — known video model capabilities (editable in settings).
func SeedVideoModelParamsFromPresets() map[string]VideoModelParams {
	all := []string{"t2v", "i2v", "flf", "r2v", "v2v"}
	i2vFlf := []string{"t2v", "i2v", "flf"}
	i2vOnly := []string{"t2v", "i2v"}
	return map[string]VideoModelParams{
		// DashScope Wan
		"wan3.0-video":       {Modes: all, Protocol: VideoProtocolDashMedia, MaxReferenceImages: 10, MaxReferenceVideos: 5, MaxDurationSec: 30},
		"wan3.0-video-prime": {Modes: all, Protocol: VideoProtocolDashMedia, MaxReferenceImages: 10, MaxReferenceVideos: 5, MaxDurationSec: 30},
		"wan2.7-i2v": {Modes: []string{"t2v", "i2v", "flf", "v2v"}, Protocol: VideoProtocolDashMedia, V2VSlot: "first_clip", MaxDurationSec: 15},
		"wan2.7-t2v": {Modes: []string{"t2v"}, Protocol: VideoProtocolDashMedia, MaxDurationSec: 15},
		"wan2.6-i2v":         {Modes: i2vOnly, Protocol: VideoProtocolDashImgURL},
		"wan2.6-t2v":         {Modes: []string{"t2v"}, Protocol: VideoProtocolDashImgURL},
		"wan2.5-i2v-preview": {Modes: i2vOnly, Protocol: VideoProtocolDashImgURL},
		"wan2.2-i2v-plus":    {Modes: i2vOnly, Protocol: VideoProtocolDashImgURL},
		"wan2.2-kf2v-flash":  {Modes: i2vFlf, Protocol: VideoProtocolDashImgURL},
		"wan2.1-i2v-plus":    {Modes: i2vOnly, Protocol: VideoProtocolDashImgURL},
		// MiniMax H3+
		"MiniMax-H3":     {Modes: all, Protocol: VideoProtocolMiniMaxH3, MaxReferenceImages: 9, MaxReferenceVideos: 3, MaxDurationSec: 15},
		"MiniMax-H3-Max": {Modes: all, Protocol: VideoProtocolMiniMaxH3, MaxReferenceImages: 9, MaxReferenceVideos: 3, MaxDurationSec: 15},
		// Volcengine Seedance (content roles)
		"doubao-seedance-1-0-pro-250528": {Modes: i2vFlf, Protocol: VideoProtocolVolcContent},
		"doubao-seedance-2-0":            {Modes: all, Protocol: VideoProtocolVolcContent, MaxReferenceImages: 9, MaxReferenceVideos: 3},
		"doubao-seedance-2-0-260128":     {Modes: all, Protocol: VideoProtocolVolcContent, MaxReferenceImages: 9, MaxReferenceVideos: 3},
	}
}

// SeedImageModelParamsFromPresets — known image model capabilities.
func SeedImageModelParamsFromPresets() map[string]ImageModelParams {
	t2i := []string{"t2i"}
	t2iI2i := []string{"t2i", "i2i"}
	t2iI2iEdit := []string{"t2i", "i2i", "edit"}
	return map[string]ImageModelParams{
		"qwen-image-3.0-pro":   {Modes: t2iI2iEdit, MaxReferenceImages: 4},
		"qwen-image":           {Modes: t2iI2i, MaxReferenceImages: 4},
		"wan2.7-image":         {Modes: t2iI2i, MaxReferenceImages: 4},
		"wan2.6-image":         {Modes: t2iI2i, MaxReferenceImages: 4},
		"z-image-turbo":        {Modes: t2iI2iEdit, MaxReferenceImages: 4},
		"image-01":             {Modes: t2i},
		"gpt-image-1":          {Modes: t2i},
		"doubao-seedream-3-0-t2i-250415": {Modes: t2i},
	}
}

// ensureMediaModelParamsSeeds fills missing keys from seed catalogs (does not overwrite user rows).
func ensureMediaModelParamsSeeds(s Settings) {
	providers := s.Map("providers")
	vid := parseVideoModelParamsMap(providers["video_model_params"])
	if vid == nil {
		vid = map[string]VideoModelParams{}
	}
	img := parseImageModelParamsMap(providers["image_model_params"])
	if img == nil {
		img = map[string]ImageModelParams{}
	}
	changed := providers["video_model_params"] == nil || providers["image_model_params"] == nil
	for id, p := range SeedVideoModelParamsFromPresets() {
		if _, ok := vid[id]; !ok {
			vid[id] = p
			changed = true
		}
	}
	for id, p := range SeedImageModelParamsFromPresets() {
		if _, ok := img[id]; !ok {
			img[id] = p
			changed = true
		}
	}
	if changed {
		providers["video_model_params"] = VideoModelParamsToAny(vid)
		providers["image_model_params"] = ImageModelParamsToAny(img)
	}
}
