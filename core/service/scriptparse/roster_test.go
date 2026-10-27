package scriptparse

import (
	"strings"
	"testing"
)

func TestComposeCharacterLookBody(t *testing.T) {
	b := composeCharacterLookBody(RegistryLook{Hair: "湿发", Wardrobe: "青衫", Accessories: "竹杖"})
	if b != "发型：湿发 | 服装：青衫 | 随身物：竹杖" {
		t.Fatalf("got %q", b)
	}
	if composeCharacterLookBody(RegistryLook{Body: "plain"}) != "plain" {
		t.Fatal("body preferred")
	}
}

func TestParseRegistryLooksFallbackLookBody(t *testing.T) {
	looks := parseRegistryLooks(nil, "灰袍", "默认", "c1_look_")
	if len(looks) != 1 || looks[0].Wardrobe != "灰袍" || looks[0].Label != "默认" {
		t.Fatalf("%#v", looks)
	}
}

func TestRosterFromPipelineReuseCreate(t *testing.T) {
	pipe := &ScriptPipeline{
		Locale: "zh",
		EntityRegistry: &EntityRegistry{
			Characters: []RegistryCharacter{
				{
					CharacterID: "char_existing", Decision: DecisionReuse, Scope: ScopeSeries,
					Name: "行者", Role: "protagonist", VisualIdentity: "蓄须",
					Looks: []RegistryLook{{
						LookID: "look_old", Decision: DecisionReuse, Scope: ScopeSeries,
						Label: "青衫", Hair: "", Wardrobe: "青衫芒鞋", Accessories: "竹杖",
					}},
				},
				{
					CharacterID: "char_new", Decision: DecisionCreate, Scope: ScopeEpisode,
					Name: "路人", Role: "extra",
					Looks: []RegistryLook{{
						LookID: "char_new_default", Decision: DecisionCreate, Scope: ScopeEpisode,
						Label: "默认", Wardrobe: "褐衣",
					}},
				},
			},
			Locations: []RegistryLocation{
				{
					LocationID: "loc_rain", Decision: DecisionReuse, Scope: ScopeSeries, Name: "雨巷",
					Looks: []RegistryLook{{LookID: "sl1", Decision: DecisionReuse, Label: "夜", Body: "青石反光"}},
				},
			},
		},
	}
	chars, scenes := RosterFromPipeline(pipe)
	if len(chars) != 2 || len(scenes) != 1 {
		t.Fatalf("lens %d %d", len(chars), len(scenes))
	}
	if chars[0]["id"] != "char_existing" || chars[0]["episode_only"] != nil {
		t.Fatalf("reuse char %#v", chars[0])
	}
	if chars[1]["episode_only"] != true {
		t.Fatalf("create char should be episode_only %#v", chars[1])
	}
	looks0 := chars[0]["looks"].([]any)
	body := looks0[0].(map[string]any)["body"].(string)
	if body != "服装：青衫芒鞋 | 随身物：竹杖" {
		t.Fatalf("body %q", body)
	}
	if scenes[0]["id"] != "loc_rain" {
		t.Fatalf("scene %#v", scenes[0])
	}
}

func TestFormatExistingRosterBlock(t *testing.T) {
	s := formatExistingRosterBlock(&ExistingRoster{
		Characters: []ExistingCharacter{{
			ID: "c1", Name: "行者", VisualIdentity: "蓄须",
			Looks: []ExistingLook{{ID: "l1", Label: "默认", Body: "青衫"}},
		}},
		Scenes: []ExistingScene{{
			ID: "s1", Name: "雨巷",
			Looks: []ExistingLook{{ID: "sl1", Label: "夜", Body: "反光"}},
		}},
	})
	if !containsAll(s, "c1", "行者", "l1", "s1", "雨巷") {
		t.Fatalf("block %q", s)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestLintReuseMissingID(t *testing.T) {
	pipe := &ScriptPipeline{
		EntityRegistry: &EntityRegistry{
			Characters: []RegistryCharacter{
				{CharacterID: "c1", Name: "甲", Role: "protagonist", Decision: DecisionReuse, VisualIdentity: "短发"},
			},
		},
	}
	findings := LintPipeline(pipe)
	saw := false
	for _, f := range findings {
		if f.FieldPath == "entity_registry.characters[0].existing_character_id" {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("expected reuse id finding: %#v", findings)
	}
}
