package gen

import (
	"fmt"
	"os"
	"strings"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

func Resolve(settings config.Settings) (port.GenerationProvider, error) {
	gen := settings.Map("providers", "generation")
	kind := strings.ToLower(config.FirstNonEmpty(
		os.Getenv("FILM_GENERATION_PROVIDER"),
		strMap(gen, "provider"),
		"volcengine",
	))
	switch kind {
	case "danmo-make", "make", "danqing":
		// Legacy peer endpoint only — not offered in Settings UI.
		base := config.FirstNonEmpty(os.Getenv("FILM_MAKE_BASE_URL"), strMap(gen, "base_url"), "http://127.0.0.1:7800")
		key := config.FirstNonEmpty(os.Getenv("FILM_MAKE_API_KEY"), strMap(gen, "api_key"))
		return NewMake(base, key), nil
	case "volcengine", "bytedance", "seedance", "ark":
		cfg := settings.Map("providers", "volcengine")
		if len(cfg) == 0 {
			cfg = gen
		}
		return NewVolcengine(
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_MODEL"), strMap(cfg, "model")),
		), nil
	case "dashscope", "alibaba", "ali", "wanxiang":
		cfg := settings.Map("providers", "dashscope")
		if len(cfg) == 0 {
			cfg = gen
		}
		return NewDashScope(
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_MODEL"), strMap(cfg, "model")),
		), nil
	case "minimax", "hailuo":
		cfg := settings.Map("providers", "minimax")
		if len(cfg) == 0 {
			cfg = gen
		}
		return NewMiniMax(
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_BASE_URL"), strMap(cfg, "base_url")),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_API_KEY"), strMap(cfg, "api_key")),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_MODEL"), strMap(cfg, "model")),
		), nil
	default:
		return nil, fmt.Errorf("unknown generation provider %q (fail loud)", kind)
	}
}

func strMap(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return strings.TrimSpace(v)
}
