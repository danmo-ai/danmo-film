package config

import (
	"fmt"
	"strings"
)

// LLMModel is one selectable chat model under a provider profile.
// Capacity/sampling live in providers.llm_model_params (by model id).
// Legacy ContextWindow/MaxOutputTokens are read for migration only; not written back.
type LLMModel struct {
	ID               string `json:"id"`
	Label            string `json:"label,omitempty"`
	Enabled          bool   `json:"enabled"`
	ReasoningDialect string `json:"reasoning_dialect,omitempty"` // protocol dialect (thinking wire shape)
	ContextWindow    int    `json:"context_window,omitempty"`    // legacy migrate
	MaxOutputTokens  int    `json:"max_output_tokens,omitempty"` // legacy migrate
}

// LLMProfile is one OpenAI/Anthropic/Responses endpoint with its own models.
// Protocol is bound at profile (vendor×protocol preset), not per model.
type LLMProfile struct {
	ID             string     `json:"id"`
	Label          string     `json:"label,omitempty"`
	Protocol       string     `json:"protocol"` // openai | openai_responses | anthropic
	BaseURL        string     `json:"base_url"`
	AuthStyle      string     `json:"auth_style,omitempty"` // bearer | x_api_key
	APIKey         string     `json:"api_key,omitempty"`
	Enabled        bool       `json:"enabled"`
	DefaultDialect string     `json:"default_dialect,omitempty"`
	PresetID       string     `json:"preset_id,omitempty"`
	Models         []LLMModel `json:"models"`
}

// EnabledLLMModel is a flattened picker row for the creation UI.
// Ref is the create-page value: provider_id/model_id.
type EnabledLLMModel struct {
	Ref              string  `json:"ref"` // provider_id/model_id — picker / project field
	ID               string  `json:"id"`  // bare model id (upstream + llm_model_params key)
	Label            string  `json:"label"`
	ProviderID       string  `json:"provider_id"`
	ProviderLabel    string  `json:"provider_label"`
	Protocol         string  `json:"protocol"`
	ReasoningDialect string  `json:"reasoning_dialect,omitempty"`
	ContextWindow    int     `json:"context_window,omitempty"`
	MaxOutputTokens  int     `json:"max_output_tokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	TopP             float64 `json:"top_p,omitempty"`
	ThinkingMode     string  `json:"thinking_mode,omitempty"`
	ThinkingBudget   int     `json:"thinking_budget,omitempty"`
}

func DefaultLLMProfiles() []LLMProfile {
	p, ok := ProfileFromPreset("openai-chat", "openai-chat")
	if !ok {
		return nil
	}
	return []LLMProfile{p}
}

// LLMProfiles returns configured profiles, migrating from legacy providers.llm when empty.
func LLMProfiles(s Settings) []LLMProfile {
	raw := s.Map("providers")["llm_profiles"]
	profiles := parseLLMProfiles(raw)
	if len(profiles) > 0 {
		return profiles
	}
	return migrateLegacyLLM(s)
}

func migrateLegacyLLM(s Settings) []LLMProfile {
	llm := s.Map("providers", "llm")
	protocol := FirstNonEmpty(strAny(llm["protocol"]), "openai")
	base := FirstNonEmpty(strAny(llm["base_url"]), "https://api.openai.com/v1")
	key := strAny(llm["api_key"])
	model := FirstNonEmpty(strAny(llm["model"]), "gpt-4.1")
	id := FirstNonEmpty(strAny(llm["provider"]), "openai-chat")
	if id == "danmo-make" || id == "make" || id == "danqing" || id == "openai" {
		id = "openai-chat"
	}
	label := FirstNonEmpty(strAny(llm["label"]), "OpenAI · Chat")
	if label == "danmo-make" || label == "make" || label == "Danmo Make" || label == "OpenAI" {
		label = "OpenAI · Chat"
	}
	dialect := InferReasoningDialect(model)
	if dialect == "" {
		dialect = DialectOpenAI
	}
	return []LLMProfile{{
		ID:             id,
		Label:          label,
		Protocol:       protocol,
		BaseURL:        base,
		AuthStyle:      "bearer",
		APIKey:         key,
		Enabled:        true,
		DefaultDialect: dialect,
		PresetID:       "openai-chat",
		Models: []LLMModel{{
			ID: model, Label: model, Enabled: true, ReasoningDialect: dialect,
		}},
	}}
}

func parseLLMProfiles(raw any) []LLMProfile {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]LLMProfile, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := LLMProfile{
			ID:             strAny(m["id"]),
			Label:          strAny(m["label"]),
			Protocol:       FirstNonEmpty(strAny(m["protocol"]), "openai"),
			BaseURL:        strAny(m["base_url"]),
			AuthStyle:      FirstNonEmpty(strAny(m["auth_style"]), "bearer"),
			APIKey:         SanitizeAPIKey(strAny(m["api_key"])),
			Enabled:        boolAny(m["enabled"], true),
			DefaultDialect: strAny(m["default_dialect"]),
			PresetID:       strAny(m["preset_id"]),
			Models:         parseLLMModels(m["models"]),
		}
		if p.ID == "" {
			p.ID = "llm-" + fmt.Sprintf("%d", len(out)+1)
		}
		if p.Label == "" {
			p.Label = p.ID
		}
		out = append(out, p)
	}
	return out
}

func parseLLMModels(raw any) []LLMModel {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]LLMModel, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strAny(m["id"])
		if id == "" {
			continue
		}
		out = append(out, LLMModel{
			ID:               id,
			Label:            FirstNonEmpty(strAny(m["label"]), id),
			Enabled:          boolAny(m["enabled"], true),
			ReasoningDialect: strAny(m["reasoning_dialect"]),
			ContextWindow:    intAny(m["context_window"]),
			MaxOutputTokens:  intAny(m["max_output_tokens"]),
		})
	}
	return out
}

// ListEnabledLLMModels returns picker rows for enabled models on enabled profiles.
func ListEnabledLLMModels(s Settings) []EnabledLLMModel {
	var out []EnabledLLMModel
	for _, p := range LLMProfiles(s) {
		if !p.Enabled {
			continue
		}
		for _, m := range p.Models {
			if !m.Enabled || strings.TrimSpace(m.ID) == "" {
				continue
			}
			dialect := ""
			if p.Protocol == "openai" || p.Protocol == "openai_compatible" {
				dialect = ResolveReasoningDialect(p, m.ID)
			}
			rp := ResolveModelParams(s, m.ID)
			out = append(out, EnabledLLMModel{
				Ref:              FormatLLMModelRef(p.ID, m.ID),
				ID:               m.ID,
				Label:            FirstNonEmpty(m.Label, m.ID),
				ProviderID:       p.ID,
				ProviderLabel:    FirstNonEmpty(p.Label, p.ID),
				Protocol:         p.Protocol,
				ReasoningDialect: dialect,
				ContextWindow:    rp.ContextWindow,
				MaxOutputTokens:  rp.MaxOutputTokens,
				Temperature:      rp.Temperature,
				TopP:             rp.TopP,
				ThinkingMode:     rp.ThinkingMode,
				ThinkingBudget:   rp.ThinkingBudget,
			})
		}
	}
	return out
}

// DefaultLLMModelID returns providers.llm_default_model as provider_id/model_id,
// or the first enabled model ref.
func DefaultLLMModelID(s Settings) string {
	def := strings.TrimSpace(s.String("providers", "llm_default_model"))
	enabled := ListEnabledLLMModels(s)
	if def != "" {
		for _, m := range enabled {
			if m.Ref == def {
				return m.Ref
			}
		}
	}
	if len(enabled) > 0 {
		return enabled[0].Ref
	}
	return ""
}

// FindLLMProfileForModel returns the enabled profile for provider_id/model_id.
// Both parts are required; the model must be enabled on that provider (dialect
// is bound to the provider×model row, not model id alone).
func FindLLMProfileForModel(s Settings, ref string) (LLMProfile, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = DefaultLLMModelID(s)
	}
	providerID, modelID, err := ParseLLMModelRef(ref)
	if err != nil {
		return LLMProfile{}, err
	}
	for _, p := range LLMProfiles(s) {
		if !p.Enabled || p.ID != providerID {
			continue
		}
		for _, m := range p.Models {
			if m.Enabled && m.ID == modelID {
				return p, nil
			}
		}
		return LLMProfile{}, fmt.Errorf("model %q is not enabled on provider %q", modelID, providerID)
	}
	return LLMProfile{}, fmt.Errorf("no enabled LLM provider %q for model %q", providerID, modelID)
}

// FindLLMModel returns the model row on profile, if any.
func FindLLMModel(profile LLMProfile, modelID string) (LLMModel, bool) {
	modelID = strings.TrimSpace(modelID)
	for _, m := range profile.Models {
		if m.ID == modelID {
			return m, true
		}
	}
	return LLMModel{}, false
}

// ValidateLLMProfiles fails loud when the same model id is enabled twice on
// one provider (duplicate picker refs). Same model id on different providers
// is allowed — create page uses provider_id/model_id refs.
func ValidateLLMProfiles(profiles []LLMProfile) error {
	for _, p := range profiles {
		if !p.Enabled {
			continue
		}
		seen := map[string]struct{}{}
		for _, m := range p.Models {
			if !m.Enabled || strings.TrimSpace(m.ID) == "" {
				continue
			}
			id := strings.TrimSpace(m.ID)
			if _, ok := seen[id]; ok {
				return fmt.Errorf("model %q enabled twice on provider %q", id, p.ID)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func ProfilesToAny(profiles []LLMProfile) []any {
	out := make([]any, 0, len(profiles))
	for _, p := range profiles {
		models := make([]any, 0, len(p.Models))
		for _, m := range p.Models {
			row := map[string]any{
				"id": m.ID, "label": m.Label, "enabled": m.Enabled,
			}
			if m.ReasoningDialect != "" {
				row["reasoning_dialect"] = m.ReasoningDialect
			}
			// Capacity/sampling are not written on profile rows (see llm_model_params).
			models = append(models, row)
		}
		row := map[string]any{
			"id":       p.ID,
			"label":    p.Label,
			"protocol": p.Protocol,
			"base_url": p.BaseURL,
			"api_key":  p.APIKey,
			"enabled":  p.Enabled,
			"models":   models,
		}
		if p.AuthStyle != "" {
			row["auth_style"] = p.AuthStyle
		}
		if p.DefaultDialect != "" {
			row["default_dialect"] = p.DefaultDialect
		}
		if p.PresetID != "" {
			row["preset_id"] = p.PresetID
		}
		out = append(out, row)
	}
	return out
}

// DefaultLLMContextWindow returns providers.llm_default_context_window.
// Fallback is the post-2025 mainstream floor (256k).
func DefaultLLMContextWindow(s Settings) int {
	n := s.Int("providers", "llm_default_context_window")
	if n > 0 {
		return n
	}
	return 256_000
}

// DefaultLLMMaxOutputTokens returns providers.llm_default_max_output_tokens.
// Fallback matches gpt-5 / gpt-5-mini class (post-2025 mainstream).
func DefaultLLMMaxOutputTokens(s Settings) int {
	n := s.Int("providers", "llm_default_max_output_tokens")
	if n > 0 {
		return n
	}
	return 128_000
}

// ClampMaxTokens caps request max_tokens by model.max_output_tokens.
// Zero request or zero model limit leaves the request unchanged (caller/adapter default).
func ClampMaxTokens(maxTokens, modelMaxOutput int) int {
	if modelMaxOutput <= 0 || maxTokens <= 0 {
		return maxTokens
	}
	if maxTokens > modelMaxOutput {
		return modelMaxOutput
	}
	return maxTokens
}

func strAny(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolAny(v any, def bool) bool {
	if v == nil {
		return def
	}
	b, ok := v.(bool)
	if ok {
		return b
	}
	return def
}

func intAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
