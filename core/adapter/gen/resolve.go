package gen

import (
	"fmt"
	"os"
	"strings"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

// Resolve returns a GenerationProvider that routes by provider_id/model_id
// (same shape as LLM). Env FILM_GENERATION_PROVIDER forces a single vendor
// (CI / quick smoke). Fail loud on unknown provider — no silent cross-vendor fallback.
//
// Each vendor adapter speaks that cloud's official async video API (pay-as-you-go
// open platform). Subscription / Token Plan keys often use a different Base URL
// and protocol face — do not treat them as a drop-in base_url swap unless the
// native task API is identical.
func Resolve(settings config.Settings) (port.GenerationProvider, error) {
	if strings.TrimSpace(os.Getenv("FILM_GENERATION_PROVIDER")) != "" {
		return resolveSingle(settings)
	}
	return NewRouter(settings)
}

func resolveSingle(settings config.Settings) (port.GenerationProvider, error) {
	gen := settings.Map("providers", "generation")
	raw := strings.ToLower(config.FirstNonEmpty(
		os.Getenv("FILM_GENERATION_PROVIDER"),
		strMap(gen, "provider"),
		"volcengine",
	))
	switch raw {
	case "danmo-make", "make", "danqing":
		v := config.GenVendorConfig(settings, "danmo-make")
		base := config.FirstNonEmpty(os.Getenv("FILM_MAKE_BASE_URL"), v.BaseURL, strMap(gen, "base_url"), "http://127.0.0.1:7800")
		key := config.FirstNonEmpty(os.Getenv("FILM_MAKE_API_KEY"), v.APIKey, strMap(gen, "api_key"))
		return NewMake(base, key), nil
	}
	kind := config.NormalizeGenVendorID(raw)
	switch kind {
	case "volcengine":
		cfg := settings.Map("providers", "volcengine")
		if len(cfg) == 0 {
			cfg = gen
		}
		v := config.GenVendorConfig(settings, "volcengine")
		model := firstEnabledModel(v, strMap(cfg, "model"))
		return NewVolcengine(
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_MODEL"), model),
		), nil
	case "dashscope":
		cfg := settings.Map("providers", "dashscope")
		if len(cfg) == 0 {
			cfg = gen
		}
		v := config.GenVendorConfig(settings, "dashscope")
		model := firstEnabledModel(v, strMap(cfg, "model"))
		return NewDashScope(
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_MODEL"), model),
		), nil
	case "minimax":
		cfg := settings.Map("providers", "minimax")
		if len(cfg) == 0 {
			cfg = gen
		}
		v := config.GenVendorConfig(settings, "minimax")
		model := firstEnabledModel(v, strMap(cfg, "model"))
		return NewMiniMax(
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_MODEL"), model),
		), nil
	default:
		return nil, fmt.Errorf("unknown generation provider %q (fail loud)", kind)
	}
}

func firstEnabledModel(v config.GenVendor, legacy string) string {
	for _, m := range v.Models {
		if m.Enabled && m.ID != "" {
			return m.ID
		}
	}
	return legacy
}

func strMap(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return strings.TrimSpace(v)
}
