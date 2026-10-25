package scriptparse

import "testing"

func TestMergeBeatShots(t *testing.T) {
	existing := []map[string]any{
		{"id": "a", "narrative_beat_index": 0, "order": 0},
		{"id": "b", "narrative_beat_index": 1, "order": 1},
	}
	neu := []map[string]any{{"id": "c", "narrative_beat_index": 1}}
	got := MergeBeatShots(existing, neu, 1)
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[1]["id"] != "c" {
		t.Fatalf("expected replaced beat: %#v", got)
	}
}

func TestMergeBeatShotsInsertsAtBeat(t *testing.T) {
	existing := []map[string]any{
		{"id": "a", "narrative_beat_index": 0},
		{"id": "b", "narrative_beat_index": 2},
	}
	neu := []map[string]any{{"id": "c", "narrative_beat_index": 1}, {"id": "d", "narrative_beat_index": 1}}
	got := MergeBeatShots(existing, neu, 1)
	if len(got) != 4 {
		t.Fatalf("len %d", len(got))
	}
	if got[0]["id"] != "a" || got[1]["id"] != "c" || got[2]["id"] != "d" || got[3]["id"] != "b" {
		t.Fatalf("order: %#v", got)
	}
}

func TestSimpleCastLockAliasesAndLooks(t *testing.T) {
	chars := []map[string]any{{
		"id": "char1", "name": "林婉", "aliases": []any{"婉婉"},
		"default_look_id": "look_default",
		"looks": []any{
			map[string]any{"id": "look_default", "label": "日常", "body": "便装"},
			map[string]any{"id": "look_gown", "label": "礼服", "body": "红裙"},
		},
	}}
	scenes := []map[string]any{{
		"id": "scene1", "name": "宴会厅", "default_look_id": "sl_default",
		"looks": []any{map[string]any{"id": "sl_default", "label": "夜"}},
	}}
	shots := []map[string]any{{
		"characters_on_screen": []any{"婉婉"},
		"narrative":            "她穿着礼服走进宴会厅",
		"location":             "scene1",
	}}
	simpleCastLock(shots, chars, scenes)
	cl := shots[0]["cast_looks"].([]any)
	if len(cl) != 1 {
		t.Fatalf("cast_looks len %d", len(cl))
	}
	row := cl[0].(map[string]any)
	if row["character_id"] != "char1" || row["look_id"] != "look_gown" {
		t.Fatalf("cast: %#v", row)
	}
	sl := shots[0]["scene_look"].(map[string]any)
	if sl["scene_id"] != "scene1" || sl["look_id"] != "sl_default" {
		t.Fatalf("scene: %#v", sl)
	}
}

func TestSanitizeRole(t *testing.T) {
	if sanitizeRole("face") != "face_anchor" {
		t.Fatal(sanitizeRole("face"))
	}
}

func TestSanitizeCastRole(t *testing.T) {
	if sanitizeCastRole("protagonist") != "protagonist" {
		t.Fatal(sanitizeCastRole("protagonist"))
	}
	if sanitizeCastRole("主角") != "protagonist" {
		t.Fatal(sanitizeCastRole("主角"))
	}
	if sanitizeCastRole("antagonist") != "supporting" {
		t.Fatal(sanitizeCastRole("antagonist"))
	}
	if sanitizeCastRole("") != "supporting" {
		t.Fatal("empty should default supporting")
	}
	// Must not confuse with segment roles.
	if sanitizeCastRole("face_anchor") != "supporting" {
		t.Fatal("segment role must not pass through as cast role")
	}
}

func TestSoftDurationBand(t *testing.T) {
	lo, hi := softDurationBand(60)
	if lo != 48 || hi != 72 {
		t.Fatalf("60 → got %g–%g want 48–72", lo, hi)
	}
	lo, hi = softDurationBand(0)
	if lo != 48 || hi != 72 {
		t.Fatalf("0 fallback → got %g–%g want 48–72", lo, hi)
	}
	lo, hi = softDurationBand(45)
	if lo != 36 || hi != 54 {
		t.Fatalf("45 → got %g–%g want 36–54", lo, hi)
	}
}
