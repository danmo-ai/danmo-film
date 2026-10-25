package config

import "strings"

// LLMProviderPreset is one vendor × wire-protocol endpoint template.
// Protocol is bound here (not per model). DefaultDialect seeds Chat Completions reasoning.
type LLMProviderPreset struct {
	ID             string     `json:"id"`
	Label          string     `json:"label"`
	Protocol       string     `json:"protocol"` // openai | openai_responses | anthropic
	BaseURL        string     `json:"base_url"`
	AuthStyle      string     `json:"auth_style"` // bearer | x_api_key
	DefaultDialect string     `json:"default_dialect,omitempty"`
	Models         []LLMModel `json:"models"`
}

// m builds a Chat Completions model row with dialect + capacity.
func m(id, label, dialect string, enabled bool, contextWindow, maxOutput int) LLMModel {
	return LLMModel{
		ID: id, Label: label, Enabled: enabled,
		ReasoningDialect: dialect,
		ContextWindow:    contextWindow,
		MaxOutputTokens:  maxOutput,
	}
}

// mc builds a non-Chat model row (Responses / Anthropic) with capacity only.
func mc(id, label string, enabled bool, contextWindow, maxOutput int) LLMModel {
	return LLMModel{
		ID: id, Label: label, Enabled: enabled,
		ContextWindow:   contextWindow,
		MaxOutputTokens: maxOutput,
	}
}

// LLMProviderPresets returns post-2025 mainstream vendor×protocol presets
// (aligned with danmo-router CHANNEL_PRESETS + danmo-work reasoning dialects).
// Capacity numbers are catalog defaults; users may override on the model row.
func LLMProviderPresets() []LLMProviderPreset {
	return []LLMProviderPreset{
		// ── 海外原厂 ──
		{
			ID: "openai-chat", Label: "OpenAI · Chat", Protocol: "openai",
			BaseURL: "https://api.openai.com/v1", AuthStyle: "bearer",
			DefaultDialect: DialectOpenAI,
			Models: []LLMModel{
				m("gpt-5", "GPT-5", DialectOpenAI, true, 400_000, 128_000),
				m("gpt-5-mini", "GPT-5 Mini", DialectOpenAI, true, 400_000, 128_000),
				m("o4-mini", "o4-mini", DialectOpenAI, false, 200_000, 100_000),
			},
		},
		{
			ID: "openai-responses", Label: "OpenAI · Responses", Protocol: "openai_responses",
			BaseURL: "https://api.openai.com/v1", AuthStyle: "bearer",
			Models: []LLMModel{
				mc("gpt-5", "GPT-5", true, 400_000, 128_000),
				mc("gpt-5-mini", "GPT-5 Mini", true, 400_000, 128_000),
			},
		},
		{
			ID: "anthropic-messages", Label: "Anthropic · Messages", Protocol: "anthropic",
			BaseURL: "https://api.anthropic.com", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("claude-sonnet-4-5", "Claude Sonnet 4.5", true, 200_000, 64_000),
				mc("claude-opus-4", "Claude Opus 4", false, 200_000, 32_000),
				mc("claude-haiku-4-5", "Claude Haiku 4.5", false, 200_000, 64_000),
			},
		},
		{
			ID: "gemini-chat", Label: "Gemini · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", AuthStyle: "bearer",
			DefaultDialect: DialectGemini,
			Models: []LLMModel{
				m("gemini-2.5-pro", "Gemini 2.5 Pro", DialectGemini, true, 1_048_576, 65_536),
				m("gemini-2.5-flash", "Gemini 2.5 Flash", DialectGemini, true, 1_048_576, 65_536),
			},
		},

		// ── 国产原厂 · Chat Completions（带推理方言）──
		{
			ID: "deepseek-chat", Label: "DeepSeek · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://api.deepseek.com/v1", AuthStyle: "bearer",
			DefaultDialect: DialectDeepSeek,
			Models: []LLMModel{
				m("deepseek-chat", "DeepSeek Chat", DialectDeepSeek, true, 128_000, 8_192),
				m("deepseek-reasoner", "DeepSeek Reasoner", DialectDeepSeek, true, 128_000, 64_000),
			},
		},
		{
			ID: "deepseek-messages", Label: "DeepSeek · Anthropic Messages", Protocol: "anthropic",
			BaseURL: "https://api.deepseek.com/anthropic", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("deepseek-chat", "DeepSeek Chat", true, 128_000, 8_192),
				mc("deepseek-reasoner", "DeepSeek Reasoner", true, 128_000, 64_000),
			},
		},
		{
			ID: "moonshot-chat", Label: "Kimi · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://api.moonshot.cn/v1", AuthStyle: "bearer",
			DefaultDialect: DialectKimi,
			Models: []LLMModel{
				m("kimi-k2.5", "Kimi K2.5", DialectKimi, true, 256_000, 8_192),
				m("kimi-k2.6", "Kimi K2.6", DialectKimi, true, 256_000, 8_192),
				m("kimi-k2.7-code", "Kimi K2.7 Code", DialectKimiCode, false, 256_000, 8_192),
				m("kimi-k3", "Kimi K3", DialectKimiK3, false, 256_000, 8_192),
			},
		},
		{
			ID: "moonshot-messages", Label: "Kimi · Anthropic Messages", Protocol: "anthropic",
			BaseURL: "https://api.moonshot.cn/anthropic", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("kimi-k2.5", "Kimi K2.5", true, 256_000, 8_192),
				mc("kimi-k2.6", "Kimi K2.6", true, 256_000, 8_192),
			},
		},
		{
			ID: "dashscope-chat", Label: "Qwen 百炼 · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", AuthStyle: "bearer",
			DefaultDialect: DialectQwen,
			Models: []LLMModel{
				m("qwen-plus", "Qwen Plus", DialectQwen, true, 131_072, 16_384),
				m("qwen3-max", "Qwen3 Max", DialectQwen, true, 262_144, 32_768),
				m("qwen3-coder-plus", "Qwen3 Coder Plus", DialectQwen, false, 262_144, 65_536),
			},
		},
		{
			ID: "dashscope-responses", Label: "Qwen 百炼 · OpenAI Responses", Protocol: "openai_responses",
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", AuthStyle: "bearer",
			Models: []LLMModel{
				mc("qwen-plus", "Qwen Plus", true, 131_072, 16_384),
				mc("qwen3-max", "Qwen3 Max", true, 262_144, 32_768),
			},
		},
		{
			ID: "dashscope-messages", Label: "Qwen 百炼 · Anthropic Messages", Protocol: "anthropic",
			BaseURL: "https://dashscope.aliyuncs.com/apps/anthropic", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("qwen3-max", "Qwen3 Max", true, 262_144, 32_768),
				mc("qwen3-coder-plus", "Qwen3 Coder Plus", true, 262_144, 65_536),
			},
		},
		{
			ID: "zhipu-chat", Label: "智谱 GLM · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://open.bigmodel.cn/api/paas/v4", AuthStyle: "bearer",
			DefaultDialect: DialectGLM,
			Models: []LLMModel{
				m("glm-4.6", "GLM-4.6", DialectGLM, true, 200_000, 16_384),
				m("glm-4.5-air", "GLM-4.5 Air", DialectGLM, true, 128_000, 16_384),
			},
		},
		{
			ID: "zhipu-messages", Label: "智谱 GLM · Anthropic Messages", Protocol: "anthropic",
			BaseURL: "https://open.bigmodel.cn/api/anthropic", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("glm-4.6", "GLM-4.6", true, 200_000, 16_384),
				mc("glm-4.5-air", "GLM-4.5 Air", true, 128_000, 16_384),
			},
		},
		{
			// Ark Chat uses thinking.type (+ effort) — same shape as DeepSeek dialect in Film.
			ID: "volcengine-chat", Label: "豆包 方舟 · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://ark.cn-beijing.volces.com/api/v3", AuthStyle: "bearer",
			DefaultDialect: DialectDeepSeek,
			Models: []LLMModel{
				m("doubao-seed-1-6-250615", "Doubao Seed 1.6", DialectDeepSeek, true, 256_000, 16_384),
			},
		},
		{
			ID: "volcengine-responses", Label: "豆包 方舟 · OpenAI Responses", Protocol: "openai_responses",
			BaseURL: "https://ark.cn-beijing.volces.com/api/v3", AuthStyle: "bearer",
			Models: []LLMModel{
				mc("doubao-seed-1-6-250615", "Doubao Seed 1.6", true, 256_000, 16_384),
			},
		},
		{
			ID: "minimax-chat", Label: "MiniMax · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://api.minimaxi.com/v1", AuthStyle: "bearer",
			DefaultDialect: DialectMiniMax,
			Models: []LLMModel{
				m("MiniMax-M2", "MiniMax M2", DialectMiniMax, true, 204_800, 8_192),
			},
		},
		{
			ID: "minimax-messages", Label: "MiniMax · Anthropic Messages", Protocol: "anthropic",
			BaseURL: "https://api.minimaxi.com/anthropic", AuthStyle: "x_api_key",
			Models: []LLMModel{
				mc("MiniMax-M2", "MiniMax M2", true, 204_800, 8_192),
			},
		},

		// ── 聚合 / 本地 ──
		{
			ID: "siliconflow-chat", Label: "硅基流动 · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://api.siliconflow.cn/v1", AuthStyle: "bearer",
			DefaultDialect: DialectOpenAI,
			Models: []LLMModel{
				m("deepseek-ai/DeepSeek-V3.2-Exp", "DeepSeek V3.2 Exp", DialectDeepSeek, true, 128_000, 8_192),
				m("Qwen/Qwen3-8B", "Qwen3 8B", DialectQwen, false, 131_072, 8_192),
			},
		},
		{
			ID: "openrouter-chat", Label: "OpenRouter · OpenAI Chat", Protocol: "openai",
			BaseURL: "https://openrouter.ai/api/v1", AuthStyle: "bearer",
			DefaultDialect: DialectOpenAI,
			Models: []LLMModel{
				m("openai/gpt-5-mini", "GPT-5 Mini", DialectOpenAI, true, 400_000, 128_000),
				m("deepseek/deepseek-chat-v3.1", "DeepSeek Chat V3.1", DialectDeepSeek, false, 128_000, 8_192),
				m("anthropic/claude-sonnet-4.5", "Claude Sonnet 4.5", DialectOpenAI, false, 200_000, 64_000),
			},
		},
		{
			ID: "ollama-chat", Label: "Ollama · OpenAI Chat", Protocol: "openai",
			BaseURL: "http://127.0.0.1:11434/v1", AuthStyle: "bearer",
			DefaultDialect: DialectOpenAI,
			Models: []LLMModel{
				m("llama3.2", "Llama 3.2", DialectOpenAI, true, 128_000, 4_096),
				m("qwen3:8b", "Qwen3 8B", DialectQwen, false, 131_072, 8_192),
			},
		},
		{
			ID: "custom", Label: "Custom · OpenAI Chat", Protocol: "openai",
			BaseURL: "", AuthStyle: "bearer",
			DefaultDialect: DialectOpenAI,
			Models:         nil,
		},
	}
}

// ProfileFromPreset builds a new LLMProfile from a preset id.
func ProfileFromPreset(presetID, instanceID string) (LLMProfile, bool) {
	presetID = FirstNonEmpty(presetID, "custom")
	presets := LLMProviderPresets()
	var preset *LLMProviderPreset
	for i := range presets {
		if presets[i].ID == presetID {
			preset = &presets[i]
			break
		}
	}
	if preset == nil {
		return LLMProfile{}, false
	}
	id := FirstNonEmpty(instanceID, preset.ID)
	models := make([]LLMModel, len(preset.Models))
	copy(models, preset.Models)
	return LLMProfile{
		ID:             id,
		Label:          preset.Label,
		Protocol:       preset.Protocol,
		BaseURL:        preset.BaseURL,
		AuthStyle:      FirstNonEmpty(preset.AuthStyle, "bearer"),
		DefaultDialect: preset.DefaultDialect,
		APIKey:         "",
		Enabled:        true,
		Models:         models,
		PresetID:       preset.ID,
	}, true
}

// PresetAllowsProtocolEdit reports whether the UI may change protocol (Custom only).
func PresetAllowsProtocolEdit(presetID string) bool {
	return presetID == "" || presetID == "custom"
}

// PresetModelCapacity looks up catalog capacity for a model id on a preset.
// Used when enabling a remote model so refresh does not invent zeros over known defaults.
func PresetModelCapacity(presetID, modelID string) (contextWindow, maxOutput int, ok bool) {
	presetID = strings.TrimSpace(presetID)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return 0, 0, false
	}
	for _, p := range LLMProviderPresets() {
		if presetID != "" && p.ID != presetID {
			continue
		}
		for _, m := range p.Models {
			if m.ID == modelID {
				return m.ContextWindow, m.MaxOutputTokens, true
			}
		}
		if presetID != "" {
			return 0, 0, false
		}
	}
	return 0, 0, false
}
