package gen

import "testing"

func TestFilterMediaUntypedShowsAll(t *testing.T) {
	rows := []map[string]any{
		{"id": "a", "label": "A"},
		{"id": "b", "label": "B"},
	}
	got := filterMedia(rows, "image")
	if len(got) != 2 {
		t.Fatalf("untyped list should pass through, got %+v", got)
	}
}

func TestParseOpenAIStyleModelsNoInventedMedia(t *testing.T) {
	payload := map[string]any{
		"data": []any{
			map[string]any{"id": "qwen-image-3.0-pro"},
			map[string]any{"id": "wan2.7-image"},
			map[string]any{"id": "qwen3.8-flash"},
			map[string]any{"id": "wan3.0-video"},
		},
	}
	items, err := parseOpenAIStyleModels(payload, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("want all 4 models, got %d", len(items))
	}
	for _, it := range items {
		if strVal(it["media"]) != "" {
			t.Fatalf("should not invent media tag: %+v", it)
		}
	}
}

func TestNormalizeDashScopeBase(t *testing.T) {
	got := normalizeDashScopeBase("https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1")
	if got != "https://token-plan.cn-beijing.maas.aliyuncs.com" {
		t.Fatalf("got %q", got)
	}
}
