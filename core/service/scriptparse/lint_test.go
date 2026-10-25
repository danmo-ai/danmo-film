package scriptparse

import "testing"

func TestLintPipelineProtagonistAndClamp(t *testing.T) {
	pipe := &ScriptPipeline{
		SourceDocument: &SourceDocument{Lines: []string{"a", "b", "c"}},
		EntityRegistry: &EntityRegistry{
			Characters: []RegistryCharacter{
				{CharacterID: "c1", Name: "甲", Role: "supporting", LookBody: ""},
				{CharacterID: "c2", Name: "乙", Role: "supporting", LookBody: "袍"},
			},
		},
		SceneIndex: []SceneIndexEntry{
			{SceneID: "S001", Heading: "INT", StoryTime: "DAY", SourceStartLine: 1, SourceEndLine: 3},
		},
		SceneArtifacts: map[string]*SceneArtifact{
			"S001": {
				SceneID: "S001",
				Breakdown: &SceneBreakdown{
					ArtifactID:  "SCENE_BREAKDOWN_S001",
					SceneID:     "S001",
					CastPresent: []string{"甲"},
					Elements: []BreakdownElement{
						{
							ElementID: "el1", Name: "杖", Category: "prop",
							Evidence: &Evidence{SourceStartLine: 0, SourceEndLine: 99},
						},
					},
				},
				Analysis: &SceneAnalysis{
					ArtifactID: "SCENE_ANALYSIS_S001",
					SceneID:    "S001",
					Beats:      []SceneBeat{{BeatID: "b1", SourceEvent: "进门", SourceStartLine: 1, SourceEndLine: 1}},
				},
			},
		},
		ShotList: []ShotListItem{
			{ShotID: "sh1", SceneID: "S001", SourceBeatIDs: []string{"b1", "missing"}},
		},
		Workflow: WorkflowState{Stage: StageGate1},
	}
	findings := LintPipeline(pipe)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	var sawProtag, sawClamp, sawBeat, sawVisual bool
	for _, f := range findings {
		switch {
		case f.Action == ActionFixValue && f.FieldPath == "entity_registry.characters[].role":
			sawProtag = true
			if f.Severity != "blocker" {
				t.Fatalf("protag severity %s", f.Severity)
			}
		case f.Action == ActionClampRef && f.NeedsLLM == false:
			sawClamp = true
			m, _ := f.SuggestedValue.(map[string]any)
			if asInt(m["source_start_line"]) != 1 {
				t.Fatalf("clamp start %#v", m)
			}
		case f.Action == ActionFixValue && !f.NeedsLLM && f.FieldPath != "":
			if len(anyToStrings(f.SuggestedValue)) == 1 {
				sawBeat = true
			}
		case f.Action == ActionFillField && f.NeedsLLM && f.FieldPath != "" && f.ArtifactID == "CHARACTER_c1":
			sawVisual = true
		}
	}
	if !sawProtag {
		t.Fatal("missing protagonist finding")
	}
	if !sawClamp {
		t.Fatal("missing evidence clamp")
	}
	if !sawBeat {
		t.Fatal("missing invalid beat fix")
	}
	if !sawVisual {
		t.Fatal("missing visual_identity fill")
	}
}

func TestApplyLocalStoryTime(t *testing.T) {
	pipe := &ScriptPipeline{
		SceneArtifacts: map[string]*SceneArtifact{
			"S001": {SceneID: "S001", Breakdown: &SceneBreakdown{ArtifactID: "SCENE_BREAKDOWN_S001", SceneID: "S001"}},
		},
	}
	t0 := RepairTarget{
		FindingID: "RULE_001", ArtifactID: "SCENE_BREAKDOWN_S001", SceneID: "S001",
		FieldPath: "breakdown.story_time", Action: ActionFillField,
		SuggestedValue: "NIGHT", NeedsLLM: false,
	}
	_, after, err := applyLocalTarget(pipe, t0)
	if err != nil {
		t.Fatal(err)
	}
	if after != "NIGHT" || pipe.SceneArtifacts["S001"].Breakdown.StoryTime != "NIGHT" {
		t.Fatalf("got %#v", after)
	}
}
