package llm

import (
	"testing"
)

func TestParseModelsPayloadOpenAI(t *testing.T) {
	raw := []byte(`{"object":"list","data":[{"id":"gpt-4.1"},{"id":"qwen3.6-27b","owned_by":"make"}]}`)
	items, err := parseModelsPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].ID != "gpt-4.1" && items[1].ID != "gpt-4.1" {
		t.Fatalf("%+v", items)
	}
}

func TestParseModelsPayloadAnthropic(t *testing.T) {
	raw := []byte(`{"data":[{"id":"claude-sonnet-4-20250514","display_name":"Claude Sonnet 4"}]}`)
	items, err := parseModelsPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Label != "Claude Sonnet 4" {
		t.Fatalf("%+v", items)
	}
}

func TestModelsListURL(t *testing.T) {
	if got := modelsListURL("openai", "http://127.0.0.1:7800"); got != "http://127.0.0.1:7800/v1/models" {
		t.Fatal(got)
	}
	if got := modelsListURL("openai", "https://api.openai.com/v1"); got != "https://api.openai.com/v1/models" {
		t.Fatal(got)
	}
	if got := modelsListURL("anthropic", "https://api.anthropic.com"); got != "https://api.anthropic.com/v1/models" {
		t.Fatal(got)
	}
}
