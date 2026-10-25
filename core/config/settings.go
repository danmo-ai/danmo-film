package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danmo-ai/danmo-film/core/runtimepaths"
)

const SecretSentinel = "********"

type Settings map[string]any

func Load() (Settings, error) {
	path := runtimepaths.SettingsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return nil, fmt.Errorf("failed to read Film settings %s: %w", path, err)
	}
	var data Settings
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("Film settings must be a JSON object: %s: %w", path, err)
	}
	if data == nil {
		data = Settings{}
	}
	return Hydrate(data), nil
}

func Save(s Settings) error {
	path := runtimepaths.SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create Film settings dir: %w", err)
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("failed to write Film settings %s: %w", path, err)
	}
	return nil
}

func Defaults() Settings {
	return Settings{
		"providers": map[string]any{
			"llm": map[string]any{
				"protocol": "openai",
				"provider": "openai-chat",
				"base_url": "https://api.openai.com/v1",
				"api_key":  "",
				"model":    "gpt-5-mini",
			},
			"llm_default_model":             "openai-chat/gpt-5-mini",
			// Post-2025 mainstream floor; per-model overrides live in llm_model_params.
			"llm_default_context_window":    256000,
			"llm_default_max_output_tokens": 128000,
			"llm_default_temperature":       0.35,
			"llm_default_top_p":             0,
			"llm_default_thinking_mode":     "off",
			"llm_model_params":              ModelParamsToAny(SeedLLMModelParamsFromPresets()),
			"llm_profiles":                  ProfilesToAny(DefaultLLMProfiles()),
			"generation": map[string]any{
				"provider": "volcengine",
				"base_url": "",
				"api_key":  "",
			},
			"volcengine": map[string]any{
				"base_url": "https://ark.cn-beijing.volces.com",
				"api_key":  "",
				"model":    "doubao-seedance-1-0-pro-250528",
			},
			"dashscope": map[string]any{
				"base_url": "https://dashscope.aliyuncs.com",
				"api_key":  "",
				"model":    "wan2.2-t2v-plus",
			},
			"minimax": map[string]any{
				"base_url": "https://api.minimax.chat",
				"api_key":  "",
				"model":    "MiniMax-Hailuo-02",
			},
		},
	}
}

func Hydrate(s Settings) Settings {
	out := clone(Defaults())
	deepMerge(out, s)
	migrateProfileCapacityToModelParams(out)
	return out
}

func Clone(s Settings) Settings {
	return clone(s)
}

func PublicView(s Settings) Settings {
	out := clone(s)
	maskSecrets(out)
	return out
}

// MergeIncoming copies incoming over existing, keeping stored api_key when the
// client sends the masked sentinel.
func MergeIncoming(existing, incoming Settings) Settings {
	out := clone(Hydrate(existing))
	inc := clone(incoming)
	restoreSentinelKeys(out, inc)
	deepMerge(out, inc)
	return Hydrate(out)
}

func (s Settings) Map(keys ...string) map[string]any {
	cur := map[string]any(s)
	for _, k := range keys {
		v, ok := cur[k]
		if !ok {
			return map[string]any{}
		}
		m, ok := v.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		cur = m
	}
	return cur
}

func (s Settings) String(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	parent := s.Map(keys[:len(keys)-1]...)
	v, ok := parent[keys[len(keys)-1]]
	if !ok {
		return ""
	}
	str, _ := v.(string)
	return strings.TrimSpace(str)
}

func (s Settings) Int(keys ...string) int {
	if len(keys) == 0 {
		return 0
	}
	parent := s.Map(keys[:len(keys)-1]...)
	return intAny(parent[keys[len(keys)-1]])
}

func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func clone(s Settings) Settings {
	raw, err := json.Marshal(s)
	if err != nil {
		return Settings{}
	}
	var out Settings
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return Settings{}
	}
	return out
}

func deepMerge(dst, src map[string]any) {
	if src == nil {
		return
	}
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			dm, ok := dst[k].(map[string]any)
			if !ok {
				dm = map[string]any{}
				dst[k] = dm
			}
			deepMerge(dm, sm)
			continue
		}
		dst[k] = sv
	}
}

func maskSecrets(m map[string]any) {
	for k, v := range m {
		switch nested := v.(type) {
		case map[string]any:
			maskSecrets(nested)
		case []any:
			for _, item := range nested {
				if nm, ok := item.(map[string]any); ok {
					maskSecrets(nm)
				}
			}
		default:
			if k == "api_key" {
				if str, ok := v.(string); ok && strings.TrimSpace(str) != "" {
					m[k] = SecretSentinel
				}
			}
		}
	}
}

func restoreSentinelKeys(existing, incoming map[string]any) {
	for k, iv := range incoming {
		ev, ok := existing[k]
		if !ok {
			continue
		}
		im, iok := iv.(map[string]any)
		em, eok := ev.(map[string]any)
		if iok && eok {
			restoreSentinelKeys(em, im)
			continue
		}
		ia, iarr := iv.([]any)
		ea, earr := ev.([]any)
		if iarr && earr {
			restoreSentinelSlice(ea, ia)
			continue
		}
		if k == "api_key" {
			istr, _ := iv.(string)
			estr, _ := ev.(string)
			if istr == SecretSentinel {
				incoming[k] = estr
			}
		}
	}
}

func restoreSentinelSlice(existing, incoming []any) {
	byID := map[string]map[string]any{}
	for _, item := range existing {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strAny(m["id"])
		if id != "" {
			byID[id] = m
		}
	}
	for _, item := range incoming {
		im, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strAny(im["id"])
		em := byID[id]
		if em == nil {
			continue
		}
		restoreSentinelKeys(em, im)
	}
}
