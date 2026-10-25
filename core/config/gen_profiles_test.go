package config

import "testing"

func TestDefaultGenerationModelID(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"generation_profiles": []any{
				map[string]any{
					"id": "volcengine", "vendor": "volcengine", "enabled": true,
					"models": []any{
						map[string]any{"id": "doubao-seedance-1-0-pro-250528", "enabled": true},
					},
				},
				map[string]any{
					"id": "minimax", "vendor": "minimax", "enabled": true,
					"models": []any{
						map[string]any{"id": "MiniMax-Hailuo-02", "enabled": true},
					},
				},
			},
			"generation_default_model": "minimax/MiniMax-Hailuo-02",
		},
	})
	got := DefaultGenerationModelID(s)
	if got != "minimax/MiniMax-Hailuo-02" {
		t.Fatalf("default=%q", got)
	}
	v, bare, err := FindGenVendorForModel(s, "volcengine/doubao-seedance-1-0-pro-250528")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "volcengine" || v.Vendor != "volcengine" || bare != "doubao-seedance-1-0-pro-250528" {
		t.Fatalf("vendor=%+v bare=%q", v, bare)
	}
}

func TestFindGenVendorBareAmbiguous(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"generation_profiles": []any{
				map[string]any{
					"id": "volcengine", "vendor": "volcengine", "enabled": true,
					"models": []any{map[string]any{"id": "same-id", "enabled": true}},
				},
				map[string]any{
					"id": "dashscope", "vendor": "dashscope", "enabled": true,
					"models": []any{map[string]any{"id": "same-id", "enabled": true}},
				},
			},
		},
	})
	_, _, err := FindGenVendorForModel(s, "same-id")
	if err == nil {
		t.Fatal("expected ambiguous bare model error")
	}
}

func TestMultiProfileSameVendor(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"generation_profiles": []any{
				map[string]any{
					"id": "volc-api", "label": "火山 API", "vendor": "volcengine",
					"enabled": true, "api_key": "k1",
					"models": []any{map[string]any{"id": "m1", "enabled": true}},
				},
				map[string]any{
					"id": "volc-plan", "label": "火山 Plan", "vendor": "volcengine",
					"enabled": true, "api_key": "k2",
					"models": []any{map[string]any{"id": "m1", "enabled": true}},
				},
			},
		},
	})
	enabled := ListEnabledGenModels(s)
	if len(enabled) != 2 {
		t.Fatalf("enabled=%d", len(enabled))
	}
	v, bare, err := FindGenVendorForModel(s, "volc-plan/m1")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "volc-plan" || v.Vendor != "volcengine" || bare != "m1" || v.APIKey != "k2" {
		t.Fatalf("got %+v bare=%q", v, bare)
	}
	// Must not collapse custom profile id to vendor kind.
	if _, _, err := FindGenVendorForModel(s, "volcengine/m1"); err == nil {
		t.Fatal("expected missing profile volcengine")
	}
}

func TestMigrateLegacyStringProfiles(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{
			"generation_profiles": []any{"volcengine", "minimax"},
			"volcengine": map[string]any{
				"enabled": true,
				"api_key": "vk",
				"models":  []any{map[string]any{"id": "seedance", "enabled": true}},
			},
			"minimax": map[string]any{
				"enabled": true,
				"api_key": "mk",
				"models":  []any{map[string]any{"id": "hailuo", "enabled": true}},
			},
		},
	})
	profiles := GenerationProfiles(s)
	if len(profiles) != 2 {
		t.Fatalf("profiles=%d", len(profiles))
	}
	if profiles[0].ID != "volcengine" || profiles[0].Vendor != "volcengine" {
		t.Fatalf("p0=%+v", profiles[0])
	}
	if profiles[1].APIKey != "mk" {
		t.Fatalf("p1 key=%q", profiles[1].APIKey)
	}
	ref := DefaultGenerationModelID(s)
	if ref != "volcengine/seedance" {
		t.Fatalf("default=%q", ref)
	}
}
