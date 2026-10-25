package config

import (
	"strings"
	"testing"
)

func TestLLMProfilesMigratesLegacy(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"llm": map[string]any{
				"protocol": "openai",
				"base_url": "https://api.deepseek.com/v1",
				"api_key":  "sk-x",
				"model":    "deepseek-chat",
			},
			"llm_profiles": []any{},
		},
	})
	profiles := LLMProfiles(s)
	if len(profiles) != 1 {
		t.Fatalf("len=%d", len(profiles))
	}
	if profiles[0].Models[0].ID != "deepseek-chat" {
		t.Fatalf("model=%q", profiles[0].Models[0].ID)
	}
	if profiles[0].BaseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("base=%q", profiles[0].BaseURL)
	}
}

func TestProfileFromPreset(t *testing.T) {
	p, ok := ProfileFromPreset("deepseek-chat", "deepseek-1")
	if !ok || p.ID != "deepseek-1" || p.Protocol != "openai" {
		t.Fatalf("%+v ok=%v", p, ok)
	}
	if !strings.Contains(p.BaseURL, "deepseek.com") {
		t.Fatalf("base=%q", p.BaseURL)
	}
	if p.DefaultDialect != DialectDeepSeek {
		t.Fatalf("dialect=%q", p.DefaultDialect)
	}
}

func TestResolveReasoningDialect(t *testing.T) {
	p := LLMProfile{
		DefaultDialect: DialectQwen,
		Models: []LLMModel{
			{ID: "custom-m", ReasoningDialect: DialectDeepSeek},
			{ID: "inherit-m"},
		},
	}
	if got := ResolveReasoningDialect(p, "custom-m"); got != DialectDeepSeek {
		t.Fatalf("model override=%q", got)
	}
	if got := ResolveReasoningDialect(p, "inherit-m"); got != DialectQwen {
		t.Fatalf("profile default=%q", got)
	}
	// Infer only when this provider's model row has no dialect and no profile default.
	p2 := LLMProfile{Models: []LLMModel{{ID: "deepseek-chat"}}}
	if got := ResolveReasoningDialect(p2, "deepseek-chat"); got != DialectDeepSeek {
		t.Fatalf("infer under provider=%q", got)
	}
	if got := ResolveReasoningDialect(LLMProfile{DefaultDialect: DialectOpenAI}, "missing"); got != DialectOpenAI {
		t.Fatalf("unknown model on provider=%q", got)
	}
}

func TestValidateLLMProfilesConflict(t *testing.T) {
	// Same model id on different providers is OK (refs disambiguate).
	if err := ValidateLLMProfiles([]LLMProfile{
		{ID: "a", Protocol: "openai", Enabled: true, Models: []LLMModel{{ID: "m1", Enabled: true}}},
		{ID: "b", Protocol: "openai_responses", Enabled: true, Models: []LLMModel{{ID: "m1", Enabled: true}}},
	}); err != nil {
		t.Fatalf("cross-provider ok: %v", err)
	}
	err := ValidateLLMProfiles([]LLMProfile{
		{ID: "a", Protocol: "openai", Enabled: true, Models: []LLMModel{
			{ID: "m1", Enabled: true}, {ID: "m1", Enabled: true},
		}},
	})
	if err == nil {
		t.Fatal("expected duplicate on same provider")
	}
}

func TestLLMModelRef(t *testing.T) {
	if got := FormatLLMModelRef("openai-chat", "gpt-5-mini"); got != "openai-chat/gpt-5-mini" {
		t.Fatalf("format=%q", got)
	}
	p, m, err := ParseLLMModelRef("openai-chat/gpt-5-mini")
	if err != nil || p != "openai-chat" || m != "gpt-5-mini" {
		t.Fatalf("parse=%q/%q err=%v", p, m, err)
	}
	if _, _, err := ParseLLMModelRef("gpt-5-mini"); err == nil {
		t.Fatal("bare id should fail")
	}
}

func TestListEnabledLLMModels(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"llm_profiles": ProfilesToAny([]LLMProfile{
				{
					ID: "a", Label: "A", Protocol: "openai", BaseURL: "http://a", Enabled: true,
					Models: []LLMModel{
						{ID: "m1", Label: "M1", Enabled: true},
						{ID: "m2", Label: "M2", Enabled: false},
					},
				},
				{
					ID: "b", Label: "B", Protocol: "anthropic", BaseURL: "http://b", Enabled: false,
					Models: []LLMModel{{ID: "m3", Enabled: true}},
				},
			}),
		},
	})
	items := ListEnabledLLMModels(s)
	if len(items) != 1 || items[0].ID != "m1" || items[0].Ref != "a/m1" {
		t.Fatalf("%+v", items)
	}
}

func TestFindLLMProfileForModel(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"llm_profiles": ProfilesToAny([]LLMProfile{
				{ID: "make", Protocol: "openai", BaseURL: "http://make", Enabled: true,
					Models: []LLMModel{{ID: "qwen", Enabled: true}}},
				{ID: "oa", Protocol: "openai", BaseURL: "http://oa", Enabled: true,
					Models: []LLMModel{{ID: "gpt-4.1", Enabled: true}}},
			}),
		},
	})
	p, err := FindLLMProfileForModel(s, "oa/gpt-4.1")
	if err != nil || p.ID != "oa" {
		t.Fatalf("ref p=%+v err=%v", p, err)
	}
	if _, err := FindLLMProfileForModel(s, "gpt-4.1"); err == nil {
		t.Fatal("bare id should fail")
	}
	def := DefaultLLMModelID(s)
	if def != "make/qwen" {
		t.Fatalf("default ref=%q want make/qwen", def)
	}
}

func TestDefaultLLMCapacityInSettings(t *testing.T) {
	d := Defaults()
	if got := DefaultLLMContextWindow(d); got != 256_000 {
		t.Fatalf("context default=%d", got)
	}
	if got := DefaultLLMMaxOutputTokens(d); got != 128_000 {
		t.Fatalf("max_output default=%d", got)
	}
	if DefaultLLMTemperature(d) != 0.35 {
		t.Fatalf("temp=%v", DefaultLLMTemperature(d))
	}
	if DefaultLLMThinkingMode(d) != "off" {
		t.Fatalf("thinking=%q", DefaultLLMThinkingMode(d))
	}
	if rp := ResolveModelParams(d, "deepseek-reasoner"); rp.ThinkingMode != "high" {
		t.Fatalf("reasoner seed thinking=%q", rp.ThinkingMode)
	}
	params := LLMModelParamsMap(d)
	if params["gpt-5-mini"].ContextWindow <= 0 || params["gpt-5-mini"].MaxOutputTokens <= 0 {
		t.Fatalf("seed missing gpt-5-mini: %+v", params["gpt-5-mini"])
	}
	// Existing configs without the keys pick them up via Hydrate.
	s := Hydrate(Settings{"providers": map[string]any{"llm_default_model": "gpt-5-mini"}})
	if DefaultLLMContextWindow(s) != 256_000 || DefaultLLMMaxOutputTokens(s) != 128_000 {
		t.Fatalf("hydrate defaults missing: %+v", s.Map("providers"))
	}
	rp := ResolveModelParams(s, "unknown-model")
	if rp.MaxOutputTokens != 128_000 || rp.Temperature != 0.35 {
		t.Fatalf("fallback resolve=%+v", rp)
	}
	rp = ResolveModelParams(s, "gpt-5-mini")
	if rp.ContextWindow < 256_000 {
		t.Fatalf("seeded resolve=%+v", rp)
	}
}

func TestMigrateProfileCapacityToModelParams(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"llm_model_params": map[string]any{},
			"llm_profiles": []any{
				map[string]any{
					"id": "a", "protocol": "openai", "base_url": "http://a", "enabled": true,
					"models": []any{
						map[string]any{
							"id": "legacy-m", "enabled": true,
							"context_window": 99000, "max_output_tokens": 2048,
						},
					},
				},
			},
		},
	})
	p := LLMModelParamsMap(s)["legacy-m"]
	if p.ContextWindow != 99_000 || p.MaxOutputTokens != 2048 {
		t.Fatalf("migrate=%+v", p)
	}
	// ProfilesToAny no longer persists capacity on model rows.
	for _, row := range ProfilesToAny(LLMProfiles(s)) {
		m := row.(map[string]any)
		models := m["models"].([]any)
		for _, mi := range models {
			mr := mi.(map[string]any)
			if _, ok := mr["context_window"]; ok {
				t.Fatalf("capacity still on profile model: %+v", mr)
			}
		}
	}
}

func TestClampMaxTokens(t *testing.T) {
	if got := ClampMaxTokens(0, 8192); got != 0 {
		t.Fatalf("unset req=%d", got)
	}
	if got := ClampMaxTokens(4096, 0); got != 4096 {
		t.Fatalf("unset model=%d", got)
	}
	if got := ClampMaxTokens(100_000, 8192); got != 8192 {
		t.Fatalf("clamp=%d", got)
	}
	if got := ClampMaxTokens(1000, 8192); got != 1000 {
		t.Fatalf("under=%d", got)
	}
}

func TestModelParamsRoundTrip(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"llm_model_params": map[string]any{
				"m1": map[string]any{
					"context_window": 128000, "max_output_tokens": 8192, "temperature": 0.2,
				},
			},
			"llm_profiles": ProfilesToAny([]LLMProfile{
				{
					ID: "a", Protocol: "openai", BaseURL: "http://a", Enabled: true,
					Models: []LLMModel{{ID: "m1", Enabled: true}},
				},
			}),
		},
	})
	rp := ResolveModelParams(s, "m1")
	if rp.ContextWindow != 128_000 || rp.MaxOutputTokens != 8192 || rp.Temperature != 0.2 {
		t.Fatalf("%+v", rp)
	}
	items := ListEnabledLLMModels(s)
	if len(items) != 1 || items[0].MaxOutputTokens != 8192 || items[0].Temperature != 0.2 {
		t.Fatalf("%+v", items)
	}
}

func TestProfileFromPresetHasCapacity(t *testing.T) {
	p, ok := ProfileFromPreset("deepseek-chat", "ds")
	if !ok {
		t.Fatal("missing preset")
	}
	m, ok := FindLLMModel(p, "deepseek-chat")
	if !ok || m.ContextWindow <= 0 || m.MaxOutputTokens <= 0 {
		t.Fatalf("%+v", m)
	}
}

func TestMergeIncomingKeepsProfileAPIKey(t *testing.T) {
	existing := Hydrate(Settings{
		"providers": map[string]any{
			"llm_profiles": ProfilesToAny([]LLMProfile{
				{ID: "make", Protocol: "openai", BaseURL: "http://make", APIKey: "sk-live", Enabled: true,
					Models: []LLMModel{{ID: "qwen", Enabled: true}}},
			}),
		},
	})
	incoming := Settings{
		"providers": map[string]any{
			"llm_profiles": ProfilesToAny([]LLMProfile{
				{ID: "make", Protocol: "openai", BaseURL: "http://make", APIKey: SecretSentinel, Enabled: true,
					Models: []LLMModel{{ID: "qwen", Enabled: true}, {ID: "new", Enabled: true}}},
			}),
		},
	}
	got := MergeIncoming(existing, incoming)
	profiles := LLMProfiles(got)
	if len(profiles) != 1 || profiles[0].APIKey != "sk-live" {
		t.Fatalf("%+v", profiles)
	}
	if len(profiles[0].Models) != 2 {
		t.Fatalf("models=%+v", profiles[0].Models)
	}
}
