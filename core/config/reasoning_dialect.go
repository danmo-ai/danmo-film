package config

import "strings"

// Chat Completions reasoning dialect ids (aligned with danmo-work / danmo-router).
const (
	DialectOpenAI   = "openai"
	DialectDeepSeek = "deepseek"
	DialectQwen     = "qwen"
	DialectKimi     = "kimi"
	DialectKimiCode = "kimi_code"
	DialectKimiK3   = "kimi_k3"
	DialectGLM      = "glm"
	DialectMiniMax  = "minimax"
	DialectGemini   = "gemini"
	DialectGrok     = "grok"
)

// ReasoningDialectOptions is the UI / catalog list for Chat Completions dialects.
func ReasoningDialectOptions() []map[string]string {
	return []map[string]string{
		{"id": DialectOpenAI, "label": "OpenAI (reasoning_effort)"},
		{"id": DialectDeepSeek, "label": "DeepSeek (thinking.type)"},
		{"id": DialectQwen, "label": "Qwen (enable_thinking)"},
		{"id": DialectKimi, "label": "Kimi K2.5/K2.6"},
		{"id": DialectKimiCode, "label": "Kimi Code K2.7"},
		{"id": DialectKimiK3, "label": "Kimi K3"},
		{"id": DialectGLM, "label": "GLM / 智谱"},
		{"id": DialectMiniMax, "label": "MiniMax"},
		{"id": DialectGemini, "label": "Gemini"},
		{"id": DialectGrok, "label": "Grok"},
	}
}

// InferReasoningDialect guesses Chat Completions dialect from a model id.
func InferReasoningDialect(modelID string) string {
	s := strings.ToLower(strings.TrimSpace(modelID))
	base := s
	if i := strings.LastIndex(s, "/"); i >= 0 {
		base = s[i+1:]
	}
	switch {
	case strings.Contains(s, "deepseek"):
		return DialectDeepSeek
	case strings.Contains(s, "doubao"), strings.Contains(s, "seed-1-6"):
		// Volcengine Ark Chat thinking.shape ≈ DeepSeek thinking.type
		return DialectDeepSeek
	case strings.Contains(s, "qwen"):
		return DialectQwen
	case base == "k3",
		base == "k3-256k",
		base == "kimi-for-coding",
		strings.Contains(s, "kimi-k3"),
		strings.Contains(s, "kimi-k2.8"),
		strings.HasPrefix(base, "k3-"):
		return DialectKimiK3
	case strings.Contains(s, "kimi-k2.7"),
		strings.Contains(s, "kimi-for-coding-highspeed"):
		return DialectKimiCode
	case strings.Contains(s, "kimi"), strings.Contains(s, "moonshot"):
		return DialectKimi
	case strings.Contains(s, "glm"), strings.Contains(s, "zhipu"), strings.Contains(s, "chatglm"):
		return DialectGLM
	case strings.Contains(s, "minimax"):
		return DialectMiniMax
	case strings.Contains(s, "gemini"):
		return DialectGemini
	case strings.Contains(s, "grok"):
		return DialectGrok
	case strings.Contains(s, "gpt-5"), strings.HasPrefix(base, "o1"), strings.HasPrefix(base, "o3"), strings.HasPrefix(base, "o4"):
		return DialectOpenAI
	default:
		return ""
	}
}

// ResolveReasoningDialect picks protocol dialect for a provider×model pair:
// that model row on this profile → profile default_dialect → Infer(model) → openai.
// Dialect is never global by model id alone — always resolve under a concrete profile.
// Only meaningful for protocol openai (Chat Completions).
func ResolveReasoningDialect(profile LLMProfile, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	for _, m := range profile.Models {
		if m.ID != modelID {
			continue
		}
		if d := strings.TrimSpace(m.ReasoningDialect); d != "" {
			return d
		}
		if d := strings.TrimSpace(profile.DefaultDialect); d != "" {
			return d
		}
		if d := InferReasoningDialect(modelID); d != "" {
			return d
		}
		return DialectOpenAI
	}
	// Model not on this provider — should not happen after FindLLMProfileForModel.
	if d := strings.TrimSpace(profile.DefaultDialect); d != "" {
		return d
	}
	return DialectOpenAI
}
