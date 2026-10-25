package config

import "testing"

func TestImageProfilesMultiAccount(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"image_profiles": []any{
				map[string]any{
					"id": "make-local", "label": "Make", "vendor": "danmo-make",
					"enabled": true, "base_url": "http://127.0.0.1:7800",
					"models": []any{map[string]any{"id": "flux-dev", "enabled": true}},
				},
				map[string]any{
					"id": "oa-work", "label": "OpenAI Work", "vendor": "openai",
					"enabled": true, "api_key": "sk-x",
					"models": []any{map[string]any{"id": "gpt-image-1", "enabled": true}},
				},
			},
			"image_default_model": "oa-work/gpt-image-1",
		},
	})
	if got := DefaultImageModelID(s); got != "oa-work/gpt-image-1" {
		t.Fatalf("default=%q", got)
	}
	v, bare, err := FindImageVendorForModel(s, "make-local/flux-dev")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "make-local" || v.Vendor != "danmo-make" || bare != "flux-dev" {
		t.Fatalf("got %+v bare=%q", v, bare)
	}
	enabled := ListEnabledImageModels(s)
	if len(enabled) != 2 {
		t.Fatalf("enabled=%d", len(enabled))
	}
}
