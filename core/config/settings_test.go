package config

import "testing"

func TestSanitizeAPIKey(t *testing.T) {
	if SanitizeAPIKey(SecretSentinel) != "" {
		t.Fatal("sentinel")
	}
	if SanitizeAPIKey("********") != "" {
		t.Fatal("stars")
	}
	if SanitizeAPIKey("sk-live") != "sk-live" {
		t.Fatal("keep")
	}
}

func TestMergeIncomingKeepsGenProfileAPIKey(t *testing.T) {
	existing := Hydrate(Settings{
		"providers": map[string]any{
			"generation_profiles": []any{
				map[string]any{
					"id": "dashscope", "vendor": "dashscope", "enabled": true,
					"base_url": "https://token-plan.example/compatible-mode/v1",
					"api_key":  "sk-sp-live",
					"models":   []any{map[string]any{"id": "wan3.0-video", "enabled": true}},
				},
			},
		},
	})
	incoming := Settings{
		"providers": map[string]any{
			"generation_profiles": []any{
				map[string]any{
					"id": "dashscope", "vendor": "dashscope", "enabled": true,
					"base_url": "https://token-plan.example/compatible-mode/v1",
					"api_key":  SecretSentinel,
					"models":   []any{map[string]any{"id": "wan3.0-video", "enabled": true}},
				},
			},
		},
	}
	got := MergeIncoming(existing, incoming)
	profiles := GenerationProfiles(got)
	if len(profiles) != 1 || profiles[0].APIKey != "sk-sp-live" {
		t.Fatalf("%+v", profiles)
	}
}

func TestMergeIncomingKeepsMaskedAPIKey(t *testing.T) {
	existing := Hydrate(Settings{
		"providers": map[string]any{
			"llm": map[string]any{"api_key": "sk-live", "model": "old"},
		},
	})
	incoming := Settings{
		"providers": map[string]any{
			"llm": map[string]any{"api_key": SecretSentinel, "model": "new-model"},
		},
	}
	got := MergeIncoming(existing, incoming)
	if got.String("providers", "llm", "api_key") != "sk-live" {
		t.Fatalf("api_key=%q", got.String("providers", "llm", "api_key"))
	}
	if got.String("providers", "llm", "model") != "new-model" {
		t.Fatalf("model=%q", got.String("providers", "llm", "model"))
	}
}

func TestPublicViewMasksAPIKey(t *testing.T) {
	s := Hydrate(Settings{"providers": map[string]any{"llm": map[string]any{"api_key": "sk-live"}}})
	pub := PublicView(s)
	if pub.String("providers", "llm", "api_key") != SecretSentinel {
		t.Fatal(pub.String("providers", "llm", "api_key"))
	}
	if s.String("providers", "llm", "api_key") != "sk-live" {
		t.Fatal("PublicView mutated original")
	}
}
