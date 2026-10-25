package scriptparse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputProtocolShape(t *testing.T) {
	p := outputProtocol(`{"a":"<x>"}`, `- a: required`, `{"a":"ok"}`)
	for _, want := range []string{
		"## Output (critical)",
		jsonSchemaInstruction,
		"## Schema (placeholders)",
		"## Field notes",
		"## Example (filled",
		`"a":"<x>"`,
		`"a":"ok"`,
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("protocol missing %q\n%s", want, p)
		}
	}
}

func TestPipelineSystemsHaveThreePartProtocol(t *testing.T) {
	systems := map[string]string{
		"StoryMap":   StoryMapSystem,
		"Breakdown":  BreakdownSystem,
		"Analysis":   AnalysisSystem,
		"Continuity": ContinuitySystem,
		"ShotList":   ShotListSystem,
		"GenPack":    GenPackSystem,
		"Audit":      AuditSystem,
		"Repair":     RepairSystem,
	}
	for name, sys := range systems {
		for _, section := range []string{
			"## Input (user message contains)",
			"## Schema (placeholders)",
			"## Field notes",
			"## Example (filled",
			jsonSchemaInstruction,
		} {
			if !strings.Contains(sys, section) {
				t.Errorf("%s missing section %q", name, section)
			}
		}
		// Extract example JSON fence and ensure it unmarshals.
		ex := extractFencedJSONAfter(sys, "## Example")
		if ex == "" {
			t.Errorf("%s: no example JSON fence", name)
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(ex), &v); err != nil {
			t.Errorf("%s example JSON invalid: %v\n%s", name, err, ex)
		}
	}
}

func TestRepairSystemRequiresPartial(t *testing.T) {
	if !strings.Contains(RepairSystem, "PARTIAL") && !strings.Contains(RepairSystem, "partial") {
		t.Fatal("RepairSystem must require partial patch")
	}
	if strings.Contains(strings.ToLower(RepairSystem), "full corrected artifact") {
		t.Fatal("RepairSystem must not ask for full artifact")
	}
}

func TestContinuitySceneSummariesIncludeElementEvidence(t *testing.T) {
	pipe := &ScriptPipeline{
		SceneIndex: []SceneIndexEntry{{SceneID: "S001", Heading: "内景 夜"}},
		SceneArtifacts: map[string]*SceneArtifact{
			"S001": {
				SceneID: "S001",
				Breakdown: &SceneBreakdown{
					SceneID:     "S001",
					Heading:     "内景 夜",
					CastPresent: []string{"林夏"},
					Elements: []BreakdownElement{
						{
							Category: "prop",
							Name:     "旧钥匙",
							Basis:    "explicit",
							Evidence: &Evidence{Quote: "门缝下有一把旧钥匙"},
							ContinuityStates: []string{"地上"},
						},
					},
				},
			},
		},
	}
	got := continuitySceneSummaries(pipe)
	for _, want := range []string{"S001", "旧钥匙", "prop", "门缝下有一把旧钥匙", "cast_present"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q\n%s", want, got)
		}
	}
}

func extractFencedJSONAfter(s, marker string) string {
	i := strings.Index(s, marker)
	if i < 0 {
		return ""
	}
	rest := s[i:]
	start := strings.Index(rest, "```json")
	if start < 0 {
		return ""
	}
	rest = rest[start+len("```json"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
