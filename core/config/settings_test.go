package config

import "testing"

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
