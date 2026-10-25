package llm

import (
	"fmt"
	"os"
	"strings"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

func Resolve(settings config.Settings) (port.LLMProvider, error) {
	// Env override: single-endpoint mode (CI / quick smoke).
	if proto := strings.TrimSpace(os.Getenv("FILM_LLM_PROTOCOL")); proto != "" || os.Getenv("FILM_LLM_BASE_URL") != "" {
		return resolveLegacyEnv(settings)
	}
	return NewRouter(settings)
}

func resolveLegacyEnv(settings config.Settings) (port.LLMProvider, error) {
	llm := settings.Map("providers", "llm")
	protocol := strings.ToLower(config.FirstNonEmpty(
		os.Getenv("FILM_LLM_PROTOCOL"),
		strMap(llm, "protocol"),
		strMap(llm, "provider"),
		"openai",
	))
	base := config.FirstNonEmpty(
		os.Getenv("FILM_LLM_BASE_URL"),
		strMap(llm, "base_url"),
		"https://api.openai.com/v1",
	)
	key := config.FirstNonEmpty(os.Getenv("FILM_LLM_API_KEY"), strMap(llm, "api_key"))
	model := config.FirstNonEmpty(os.Getenv("FILM_LLM_MODEL"), strMap(llm, "model"))
	pid := config.FirstNonEmpty(strMap(llm, "provider"), protocol)

	switch strings.ReplaceAll(protocol, "-", "_") {
	case "openai", "openai_compatible", "local":
		return NewOpenAIChatClient(base, key, pid, model), nil
	case "openai_responses", "responses":
		return NewOpenAIResponsesClient(base, key, pid, model), nil
	case "anthropic":
		return NewAnthropicClient(base, key, pid, model), nil
	case "mock":
		return &MockClient{}, nil
	default:
		if strings.Contains(protocol, "openai") {
			return NewOpenAIChatClient(base, key, pid, model), nil
		}
		return nil, fmt.Errorf("unknown LLM protocol %q (fail loud; set providers.llm.protocol)", protocol)
	}
}

func strMap(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return strings.TrimSpace(v)
}
