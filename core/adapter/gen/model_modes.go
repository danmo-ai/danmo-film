package gen

import (
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

// AssertVideoModeSupported fails loud when model row advertises modes[] without mode.
// Empty modes → allow (caller may have already checked settings catalog).
func AssertVideoModeSupported(modelConfig map[string]any, mode port.VideoMode) error {
	modes := modelModesList(modelConfig)
	if len(modes) == 0 {
		return nil
	}
	want := string(mode)
	for _, m := range modes {
		if strings.EqualFold(m, want) {
			return nil
		}
	}
	return fmt.Errorf("model does not support video mode %q (supports: %s)", mode, strings.Join(modes, ", "))
}

// AssertImageModeSupported mirrors AssertVideoModeSupported for images.
func AssertImageModeSupported(modelConfig map[string]any, mode port.ImageMode) error {
	modes := modelModesList(modelConfig)
	if len(modes) == 0 {
		p := modelParams(modelConfig)
		if mode == port.ImageModeT2I {
			return nil
		}
		switch mode {
		case port.ImageModeI2I:
			if p["supports_img2img"] == true || p["supports_i2i"] == true {
				return nil
			}
		case port.ImageModeEdit:
			if p["supports_edit"] == true {
				return nil
			}
		case port.ImageModeInpaint:
			if p["supports_inpaint"] == true {
				return nil
			}
		}
		return nil
	}
	want := string(mode)
	for _, m := range modes {
		if strings.EqualFold(m, want) || (mode == port.ImageModeI2I && (m == "img2img" || m == "i2i")) {
			return nil
		}
	}
	return fmt.Errorf("model does not support image mode %q (supports: %s)", mode, strings.Join(modes, ", "))
}

func modelModesList(modelConfig map[string]any) []string {
	if modelConfig == nil {
		return nil
	}
	raw, ok := modelConfig["modes"]
	if !ok {
		if p := modelParams(modelConfig); p != nil {
			raw, ok = p["modes"]
		}
	}
	if !ok {
		return nil
	}
	switch t := raw.(type) {
	case []string:
		return filterNonEmptyURLs(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, v := range t {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func modelParams(modelConfig map[string]any) map[string]any {
	if modelConfig == nil {
		return nil
	}
	if p, ok := modelConfig["parameters"].(map[string]any); ok {
		return p
	}
	return modelConfig
}

func filmProtocolFromBody(body map[string]any) string {
	if body == nil {
		return ""
	}
	if s := strings.TrimSpace(strVal(body["film_protocol"])); s != "" {
		return s
	}
	if s := strings.TrimSpace(strVal(body["protocol"])); s != "" {
		return s
	}
	return ""
}
