package scriptparse

import (
	"fmt"
	"strings"
)

// LintPipeline runs deterministic checks and stores results on pipe.RuleFindings.
func LintPipeline(pipe *ScriptPipeline) []RuleFinding {
	if pipe == nil {
		return nil
	}
	var out []RuleFinding
	n := 0
	nextID := func() string {
		n++
		return fmt.Sprintf("RULE_%03d", n)
	}
	lineCount := 0
	if pipe.SourceDocument != nil {
		lineCount = len(pipe.SourceDocument.Lines)
	}

	protag := 0
	if pipe.EntityRegistry != nil {
		for i, c := range pipe.EntityRegistry.Characters {
			aid := "CHARACTER_" + c.CharacterID
			pathBase := fmt.Sprintf("entity_registry.characters[%d]", i)
			if strings.TrimSpace(c.Name) == "" {
				out = append(out, RuleFinding{
					FindingID: nextID(), Source: "rule", Severity: "blocker",
					Action: ActionFixValue, ArtifactID: aid, FieldPath: pathBase + ".name",
					Problem: "character name is empty", RepairInstruction: "set a non-empty name from the script",
					NeedsLLM: true, Status: "open",
				})
			}
			if c.Role == "protagonist" {
				protag++
			}
			if c.Role == "supporting" || c.Role == "protagonist" {
				if strings.TrimSpace(c.VisualIdentity) == "" && strings.TrimSpace(c.LookBody) == "" {
					out = append(out, RuleFinding{
						FindingID: nextID(), Source: "rule", Severity: "info",
						Action: ActionFillField, ArtifactID: aid, FieldPath: pathBase + ".visual_identity",
						Problem:           fmt.Sprintf("%s has no visual_identity or look_body", firstNonEmpty(c.Name, c.CharacterID)),
						RepairInstruction: "fill visual_identity (face/build) from explicit script description; leave empty if none",
						NeedsLLM:          true, Status: "open",
					})
				}
			}
		}
	}
	if pipe.StoryMap != nil || (pipe.EntityRegistry != nil && len(pipe.EntityRegistry.Characters) > 0) {
		if protag != 1 {
			out = append(out, RuleFinding{
				FindingID: nextID(), Source: "rule", Severity: "blocker",
				Action: ActionFixValue, ArtifactID: "ENTITY_REGISTRY", FieldPath: "entity_registry.characters[].role",
				Problem:           fmt.Sprintf("expected exactly 1 protagonist, got %d", protag),
				RepairInstruction: "set exactly one character role to protagonist",
				NeedsLLM:          true, Status: "open",
			})
		}
	}

	for _, sc := range pipe.SceneIndex {
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil {
			continue
		}
		if bd := art.Breakdown; bd != nil {
			aid := bd.ArtifactID
			if aid == "" {
				aid = "SCENE_BREAKDOWN_" + sc.SceneID
			}
			if strings.TrimSpace(bd.StoryTime) == "" && strings.TrimSpace(sc.StoryTime) == "" {
				out = append(out, RuleFinding{
					FindingID: nextID(), Source: "rule", Severity: "info",
					Action: ActionFillField, ArtifactID: aid, SceneID: sc.SceneID,
					FieldPath:         "breakdown.story_time",
					Problem:           fmt.Sprintf("scene %s missing story_time", sc.SceneID),
					RepairInstruction: "set story_time from slugline or scene text (DAY/NIGHT/…); empty if unknown",
					NeedsLLM:          true, Status: "open",
				})
			} else if strings.TrimSpace(bd.StoryTime) == "" && strings.TrimSpace(sc.StoryTime) != "" {
				out = append(out, RuleFinding{
					FindingID: nextID(), Source: "rule", Severity: "minor",
					Action: ActionFillField, ArtifactID: aid, SceneID: sc.SceneID,
					FieldPath:         "breakdown.story_time",
					Problem:           fmt.Sprintf("scene %s breakdown story_time empty but index has %q", sc.SceneID, sc.StoryTime),
					RepairInstruction: "copy story_time from scene index",
					SuggestedValue:    sc.StoryTime, NeedsLLM: false, Status: "open",
				})
			}
			if len(bd.CastPresent) == 0 {
				out = append(out, RuleFinding{
					FindingID: nextID(), Source: "rule", Severity: "major",
					Action: ActionFillField, ArtifactID: aid, SceneID: sc.SceneID,
					FieldPath:         "breakdown.cast_present",
					Problem:           fmt.Sprintf("scene %s has empty cast_present", sc.SceneID),
					RepairInstruction: "list characters physically present in this scene from the source excerpt",
					NeedsLLM:          true, Status: "open",
				})
			}
			for ei, el := range bd.Elements {
				if el.Evidence == nil {
					continue
				}
				lo, hi := el.Evidence.SourceStartLine, el.Evidence.SourceEndLine
				if lineCount > 0 && (lo < 1 || hi > lineCount || (hi > 0 && lo > hi)) {
					clampedLo := clamp(lo, 1, lineCount)
					clampedHi := clamp(hi, clampedLo, lineCount)
					out = append(out, RuleFinding{
						FindingID: nextID(), Source: "rule", Severity: "minor",
						Action: ActionClampRef, ArtifactID: aid, SceneID: sc.SceneID,
						FieldPath:         fmt.Sprintf("breakdown.elements[%d].evidence", ei),
						Problem:           fmt.Sprintf("element %q evidence lines %d–%d out of range (1–%d)", el.Name, lo, hi, lineCount),
						RepairInstruction: "clamp evidence line numbers to document bounds",
						SuggestedValue: map[string]any{
							"source_start_line": clampedLo,
							"source_end_line":   clampedHi,
						},
						NeedsLLM: false, Status: "open",
					})
				}
			}
		}
		if an := art.Analysis; an != nil {
			aid := an.ArtifactID
			if aid == "" {
				aid = "SCENE_ANALYSIS_" + sc.SceneID
			}
			beatIDs := map[string]bool{}
			for bi, b := range an.Beats {
				beatIDs[b.BeatID] = true
				if lineCount > 0 && b.SourceStartLine > 0 {
					lo, hi := b.SourceStartLine, b.SourceEndLine
					if hi == 0 {
						hi = lo
					}
					if lo < 1 || hi > lineCount || lo > hi {
						clampedLo := clamp(lo, 1, lineCount)
						clampedHi := clamp(hi, clampedLo, lineCount)
						out = append(out, RuleFinding{
							FindingID: nextID(), Source: "rule", Severity: "minor",
							Action: ActionClampRef, ArtifactID: aid, SceneID: sc.SceneID,
							FieldPath:         fmt.Sprintf("analysis.beats[%d]", bi),
							Problem:           fmt.Sprintf("beat %s line refs %d–%d out of range", b.BeatID, lo, hi),
							RepairInstruction: "clamp beat source line numbers",
							SuggestedValue: map[string]any{
								"source_start_line": clampedLo,
								"source_end_line":   clampedHi,
							},
							NeedsLLM: false, Status: "open",
						})
					}
				}
			}
			_ = beatIDs
		}
	}

	validBeats := map[string]bool{}
	for _, art := range pipe.SceneArtifacts {
		if art == nil || art.Analysis == nil {
			continue
		}
		for _, b := range art.Analysis.Beats {
			validBeats[b.BeatID] = true
		}
	}
	for si, sh := range pipe.ShotList {
		var bad []string
		for _, bid := range sh.SourceBeatIDs {
			if bid != "" && !validBeats[bid] {
				bad = append(bad, bid)
			}
		}
		if len(bad) > 0 {
			kept := make([]string, 0, len(sh.SourceBeatIDs))
			for _, bid := range sh.SourceBeatIDs {
				if validBeats[bid] {
					kept = append(kept, bid)
				}
			}
			out = append(out, RuleFinding{
				FindingID: nextID(), Source: "rule", Severity: "major",
				Action: ActionFixValue, ArtifactID: "SHOT_" + sh.ShotID, SceneID: sh.SceneID,
				FieldPath:         fmt.Sprintf("shot_list[%d].source_beat_ids", si),
				Problem:           fmt.Sprintf("shot %s references unknown beats: %s", sh.ShotID, strings.Join(bad, ", ")),
				RepairInstruction: "drop invalid beat ids; keep only existing analysis beats",
				SuggestedValue:    kept, NeedsLLM: false, Status: "open",
			})
		}
	}

	if pipe.Workflow.AwaitingReview == "shots" || pipe.Workflow.Stage == StageGate2 || pipe.Workflow.Stage == StageR7 {
		if pipe.AuditReport != nil {
			for _, iss := range pipe.AuditReport.Issues {
				if iss.Severity == "blocker" && (iss.Status == "" || iss.Status == "open") {
					out = append(out, RuleFinding{
						FindingID: nextID(), Source: "rule", Severity: "blocker",
						Action: ActionFixValue, ArtifactID: firstNonEmpty(iss.ArtifactID, "AUDIT"),
						FieldPath:         iss.FieldPath,
						Problem:           "open audit blocker: " + iss.Problem,
						RepairInstruction: firstNonEmpty(iss.RepairInstruction, "resolve audit blocker before approving shots"),
						EvidenceQuote:     iss.EvidenceQuote, NeedsLLM: true, Status: "open",
					})
				}
			}
		}
	}

	pipe.RuleFindings = out
	return out
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
