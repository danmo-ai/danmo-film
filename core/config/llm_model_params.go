package config

import "strings"

// LLMModelParams is per-model capacity + sampling + reasoning effort
// (keyed by model id in settings). Zero / missing fields fall back to providers.llm_default_*.
// ThinkingMode is a value (off|low|medium|high|max); dialect only decides how to wire it.
type LLMModelParams struct {
	ContextWindow   int     `json:"context_window,omitempty"`
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"top_p,omitempty"`
	ThinkingMode    string  `json:"thinking_mode,omitempty"`   // off|low|medium|high|max
	ThinkingBudget  int     `json:"thinking_budget,omitempty"` // optional tokens for budget dialects
}

// ResolvedModelParams is the fully filled view after defaults merge.
type ResolvedModelParams struct {
	ContextWindow   int
	MaxOutputTokens int
	Temperature     float64
	TopP            float64 // 0 = omit / upstream default
	ThinkingMode    string  // off|low|medium|high|max
	ThinkingBudget  int
}

// DefaultLLMTemperature returns providers.llm_default_temperature (fallback 0.35).
func DefaultLLMTemperature(s Settings) float64 {
	raw := s.Map("providers")["llm_default_temperature"]
	if raw == nil {
		return 0.35
	}
	f, ok := asFloat(raw)
	if !ok {
		return 0.35
	}
	return f
}

// DefaultLLMTopP returns providers.llm_default_top_p (0 = unset / omit).
func DefaultLLMTopP(s Settings) float64 {
	raw := s.Map("providers")["llm_default_top_p"]
	if raw == nil {
		return 0
	}
	f, ok := asFloat(raw)
	if !ok {
		return 0
	}
	return f
}

// DefaultLLMThinkingMode returns providers.llm_default_thinking_mode (fallback "off").
func DefaultLLMThinkingMode(s Settings) string {
	raw := strings.TrimSpace(strings.ToLower(s.String("providers", "llm_default_thinking_mode")))
	if raw == "" {
		return "off"
	}
	return normalizeThinkingMode(raw)
}

func normalizeThinkingMode(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "", "off", "disabled", "none":
		return "off"
	case "on", "enabled", "adaptive":
		return "high"
	case "low", "medium", "high", "max":
		return s
	default:
		return s
	}
}

// LLMModelParamsMap returns providers.llm_model_params keyed by model id.
func LLMModelParamsMap(s Settings) map[string]LLMModelParams {
	raw := s.Map("providers")["llm_model_params"]
	return parseLLMModelParamsMap(raw)
}

func parseLLMModelParamsMap(raw any) map[string]LLMModelParams {
	out := map[string]LLMModelParams{}
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return out
	}
	for id, v := range m {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		row, ok := v.(map[string]any)
		if !ok {
			continue
		}
		p := LLMModelParams{
			ContextWindow:   intAny(row["context_window"]),
			MaxOutputTokens: intAny(row["max_output_tokens"]),
			ThinkingMode:    strAny(row["thinking_mode"]),
			ThinkingBudget:  intAny(row["thinking_budget"]),
		}
		if f, ok := asFloat(row["temperature"]); ok {
			p.Temperature = f
		}
		if f, ok := asFloat(row["top_p"]); ok {
			p.TopP = f
		}
		out[id] = p
	}
	return out
}

// ModelParamsToAny serializes the catalog for settings JSON.
func ModelParamsToAny(m map[string]LLMModelParams) map[string]any {
	out := make(map[string]any, len(m))
	for id, p := range m {
		row := map[string]any{}
		if p.ContextWindow > 0 {
			row["context_window"] = p.ContextWindow
		}
		if p.MaxOutputTokens > 0 {
			row["max_output_tokens"] = p.MaxOutputTokens
		}
		if p.Temperature > 0 {
			row["temperature"] = p.Temperature
		}
		if p.TopP > 0 {
			row["top_p"] = p.TopP
		}
		if strings.TrimSpace(p.ThinkingMode) != "" {
			row["thinking_mode"] = normalizeThinkingMode(p.ThinkingMode)
		}
		if p.ThinkingBudget > 0 {
			row["thinking_budget"] = p.ThinkingBudget
		}
		if len(row) == 0 {
			continue
		}
		out[id] = row
	}
	return out
}

// SeedLLMModelParamsFromPresets builds a catalog from provider presets
// (capacity + default temp + thinking defaults for reasoner-class ids).
func SeedLLMModelParamsFromPresets() map[string]LLMModelParams {
	out := map[string]LLMModelParams{}
	for _, preset := range LLMProviderPresets() {
		for _, m := range preset.Models {
			id := strings.TrimSpace(m.ID)
			if id == "" {
				continue
			}
			cur, exists := out[id]
			if !exists {
				cur = LLMModelParams{Temperature: 0.35, ThinkingMode: "off"}
			}
			if m.ContextWindow > 0 && cur.ContextWindow == 0 {
				cur.ContextWindow = m.ContextWindow
			}
			if m.MaxOutputTokens > 0 && cur.MaxOutputTokens == 0 {
				cur.MaxOutputTokens = m.MaxOutputTokens
			}
			low := strings.ToLower(id)
			if strings.Contains(low, "deepseek") {
				cur.Temperature = 0.3
			}
			if strings.Contains(low, "reasoner") ||
				strings.Contains(low, "o4-mini") ||
				strings.HasPrefix(low, "o1") ||
				strings.HasPrefix(low, "o3") {
				cur.ThinkingMode = "high"
			}
			out[id] = cur
		}
	}
	return out
}

// ResolveModelParams merges model catalog → global defaults.
func ResolveModelParams(s Settings, modelID string) ResolvedModelParams {
	modelID = strings.TrimSpace(modelID)
	cat := LLMModelParamsMap(s)[modelID]
	ctx := cat.ContextWindow
	if ctx <= 0 {
		ctx = DefaultLLMContextWindow(s)
	}
	maxOut := cat.MaxOutputTokens
	if maxOut <= 0 {
		maxOut = DefaultLLMMaxOutputTokens(s)
	}
	temp := cat.Temperature
	if temp <= 0 {
		temp = DefaultLLMTemperature(s)
	}
	topP := cat.TopP
	if topP <= 0 {
		topP = DefaultLLMTopP(s)
	}
	mode := strings.TrimSpace(cat.ThinkingMode)
	if mode == "" {
		mode = DefaultLLMThinkingMode(s)
	} else {
		mode = normalizeThinkingMode(mode)
	}
	return ResolvedModelParams{
		ContextWindow:   ctx,
		MaxOutputTokens: maxOut,
		Temperature:     temp,
		TopP:            topP,
		ThinkingMode:    mode,
		ThinkingBudget:  cat.ThinkingBudget,
	}
}

// migrateProfileCapacityToModelParams lifts legacy capacity on profile.models into llm_model_params
// without overwriting catalog fields that are already set.
func migrateProfileCapacityToModelParams(s Settings) {
	providers := s.Map("providers")
	catalog := parseLLMModelParamsMap(providers["llm_model_params"])
	if catalog == nil {
		catalog = map[string]LLMModelParams{}
	}
	changed := false
	for _, p := range LLMProfiles(s) {
		for _, m := range p.Models {
			id := strings.TrimSpace(m.ID)
			if id == "" {
				continue
			}
			if m.ContextWindow <= 0 && m.MaxOutputTokens <= 0 {
				continue
			}
			cur := catalog[id]
			if m.ContextWindow > 0 && cur.ContextWindow == 0 {
				cur.ContextWindow = m.ContextWindow
				changed = true
			}
			if m.MaxOutputTokens > 0 && cur.MaxOutputTokens == 0 {
				cur.MaxOutputTokens = m.MaxOutputTokens
				changed = true
			}
			catalog[id] = cur
		}
	}
	if changed || providers["llm_model_params"] == nil {
		providers["llm_model_params"] = ModelParamsToAny(catalog)
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
