package gen

import (
	"encoding/json"
	"testing"
)

func TestParseMakeModelsPayloadMap(t *testing.T) {
	// Make GET /api/models?installed=true&media=video returns only matching installed models.
	raw := []byte(`{
		"models": {
			"wan2.2-t2v": {
				"media": "video",
				"family": "wan",
				"installed": true,
				"name": {"zh": "万相 2.2", "en": "Wan 2.2"}
			}
		}
	}`)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	items, err := parseMakeModelsPayload(m, raw, "video")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if strVal(items[0]["id"]) != "wan2.2-t2v" {
		t.Fatalf("id=%q", items[0]["id"])
	}
	if strVal(items[0]["label"]) != "万相 2.2" {
		t.Fatalf("label=%q", items[0]["label"])
	}
	if strVal(items[0]["media"]) != "video" {
		t.Fatalf("media=%q", items[0]["media"])
	}
}
