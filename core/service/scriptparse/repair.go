package scriptparse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/danmo-ai/danmo-film/core/domain/jsonout"
	"github.com/danmo-ai/danmo-film/core/port"
)

// RepairTarget is a unified selectable fix item (rule or LLM review).
type RepairTarget struct {
	FindingID         string `json:"finding_id"`
	Source            string `json:"source"` // rule|continuity|audit
	Severity          string `json:"severity,omitempty"`
	Action            string `json:"action,omitempty"`
	ArtifactID        string `json:"artifact_id,omitempty"`
	SceneID           string `json:"scene_id,omitempty"`
	FieldPath         string `json:"field_path,omitempty"`
	Problem           string `json:"problem"`
	RepairInstruction string `json:"repair_instruction,omitempty"`
	EvidenceQuote     string `json:"evidence_quote,omitempty"`
	SuggestedValue    any    `json:"suggested_value,omitempty"`
	NeedsLLM          bool   `json:"needs_llm"`
}

// RepairPatch summarizes one field-level change for the UI diff.
type RepairPatch struct {
	FindingID      string `json:"finding_id"`
	ArtifactID     string `json:"artifact_id,omitempty"`
	FieldPath      string `json:"field_path,omitempty"`
	Action         string `json:"action,omitempty"`
	Before         any    `json:"before,omitempty"`
	After          any    `json:"after,omitempty"`
	AppliedLocally bool   `json:"applied_locally"`
	Note           string `json:"note,omitempty"`
}

// RepairPreviewOpts selects findings and optional user guidance.
type RepairPreviewOpts struct {
	Pipeline   *ScriptPipeline
	FindingIDs []string
	UserNote   string
	Model      string
}

// RepairPreviewResult is returned to the client for confirm/apply.
type RepairPreviewResult struct {
	Patches        []RepairPatch   `json:"patches"`
	ScriptPipeline *ScriptPipeline `json:"script_pipeline"`
	LLMCalls       int             `json:"llm_calls"`
}

// CollectRepairTargets flattens rule findings + continuity + audit into selectable targets.
func CollectRepairTargets(pipe *ScriptPipeline) []RepairTarget {
	if pipe == nil {
		return nil
	}
	var out []RepairTarget
	for _, f := range pipe.RuleFindings {
		if f.Status == "resolved" {
			continue
		}
		out = append(out, RepairTarget{
			FindingID: f.FindingID, Source: "rule", Severity: f.Severity, Action: f.Action,
			ArtifactID: f.ArtifactID, SceneID: f.SceneID, FieldPath: f.FieldPath,
			Problem: f.Problem, RepairInstruction: f.RepairInstruction,
			EvidenceQuote: f.EvidenceQuote, SuggestedValue: f.SuggestedValue, NeedsLLM: f.NeedsLLM,
		})
	}
	if pipe.Continuity != nil {
		for _, iss := range pipe.Continuity.Issues {
			if iss.Status == "resolved" {
				continue
			}
			out = append(out, RepairTarget{
				FindingID: iss.IssueID, Source: "continuity", Severity: iss.Severity,
				Action: firstNonEmpty(iss.Action, ActionFillField), ArtifactID: continuityArtifactID(iss), SceneID: iss.SceneID,
				FieldPath: iss.FieldPath, Problem: iss.Problem,
				RepairInstruction: iss.RepairHint, EvidenceQuote: iss.Evidence,
				NeedsLLM: strings.TrimSpace(iss.FieldPath) != "",
			})
		}
	}
	if pipe.AuditReport != nil {
		for _, iss := range pipe.AuditReport.Issues {
			if iss.Status == "resolved" {
				continue
			}
			out = append(out, RepairTarget{
				FindingID: iss.IssueID, Source: "audit", Severity: iss.Severity,
				Action: firstNonEmpty(iss.Action, ActionFillField), ArtifactID: firstNonEmpty(iss.ArtifactID, "AUDIT"),
				FieldPath: iss.FieldPath, Problem: iss.Problem,
				RepairInstruction: iss.RepairInstruction, EvidenceQuote: iss.EvidenceQuote,
				NeedsLLM: strings.TrimSpace(iss.FieldPath) != "",
			})
		}
	}
	return out
}

func continuityArtifactID(iss ContinuityIssue) string {
	if iss.SceneID != "" {
		return "SCENE_BREAKDOWN_" + iss.SceneID
	}
	if iss.FieldPath != "" {
		return "CONTINUITY"
	}
	return "CONTINUITY"
}

// RepairPreview applies local fixes and LLM repairs for selected finding IDs.
func RepairPreview(ctx context.Context, llm port.LLMProvider, opts RepairPreviewOpts) (*RepairPreviewResult, error) {
	if opts.Pipeline == nil {
		return nil, fmt.Errorf("pipeline required")
	}
	if len(opts.FindingIDs) == 0 {
		return nil, fmt.Errorf("finding_ids required")
	}
	pipe := clonePipeline(opts.Pipeline)
	if pipe.RuleFindings == nil {
		LintPipeline(pipe)
	}
	wanted := map[string]bool{}
	for _, id := range opts.FindingIDs {
		wanted[id] = true
	}
	var selected []RepairTarget
	for _, t := range CollectRepairTargets(pipe) {
		if wanted[t.FindingID] {
			selected = append(selected, t)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no matching findings for finding_ids")
	}

	var patches []RepairPatch
	calls := 0

	// 1) Local deterministic fixes first.
	var llmTargets []RepairTarget
	for _, t := range selected {
		if !t.NeedsLLM && t.SuggestedValue != nil {
			before, after, err := applyLocalTarget(pipe, t)
			if err != nil {
				patches = append(patches, RepairPatch{
					FindingID: t.FindingID, ArtifactID: t.ArtifactID, FieldPath: t.FieldPath,
					Action: t.Action, AppliedLocally: false, Note: err.Error(),
				})
				continue
			}
			patches = append(patches, RepairPatch{
				FindingID: t.FindingID, ArtifactID: t.ArtifactID, FieldPath: t.FieldPath,
				Action: t.Action, Before: before, After: after, AppliedLocally: true,
			})
			markFindingResolved(pipe, t.FindingID)
			continue
		}
		if strings.TrimSpace(t.FieldPath) == "" {
			patches = append(patches, RepairPatch{
				FindingID: t.FindingID, ArtifactID: t.ArtifactID, FieldPath: t.FieldPath,
				Action: t.Action, AppliedLocally: false, Note: "missing field_path; not auto-repairable",
			})
			continue
		}
		llmTargets = append(llmTargets, t)
	}

	// 2) Group LLM targets by artifact and repair once each.
	if len(llmTargets) > 0 {
		if llm == nil {
			return nil, fmt.Errorf("llm not configured")
		}
		groups := map[string][]RepairTarget{}
		order := []string{}
		for _, t := range llmTargets {
			key := firstNonEmpty(t.ArtifactID, firstNonEmpty(t.SceneID, "PIPELINE"))
			if _, ok := groups[key]; !ok {
				order = append(order, key)
			}
			groups[key] = append(groups[key], t)
		}
		for _, key := range order {
			ts := groups[key]
			n, p, err := repairArtifactGroup(ctx, llm, opts.Model, pipe, ts, opts.UserNote)
			calls += n
			if err != nil {
				return nil, fmt.Errorf("repair %s: %w", key, err)
			}
			patches = append(patches, p...)
			for _, t := range ts {
				markFindingResolved(pipe, t.FindingID)
			}
		}
	}

	LintPipeline(pipe)
	pipe.LLMCalls += calls
	return &RepairPreviewResult{Patches: patches, ScriptPipeline: pipe, LLMCalls: calls}, nil
}

// ApplyRepairPreview bumps version_id, re-lints, and returns the confirmed pipeline.
func ApplyRepairPreview(pipe *ScriptPipeline) (*ScriptPipeline, error) {
	if pipe == nil {
		return nil, fmt.Errorf("pipeline required")
	}
	pipe.VersionID = bumpVersionID(pipe.VersionID)
	LintPipeline(pipe)
	return pipe, nil
}

func bumpVersionID(cur string) string {
	base := strings.TrimSpace(cur)
	if base == "" {
		base = "ver"
	}
	return fmt.Sprintf("%s_r%d", base, time.Now().Unix()%100000)
}

func applyLocalTarget(pipe *ScriptPipeline, t RepairTarget) (before, after any, err error) {
	switch {
	case strings.HasPrefix(t.ArtifactID, "SCENE_BREAKDOWN_") && strings.Contains(t.FieldPath, "story_time"):
		sid := strings.TrimPrefix(t.ArtifactID, "SCENE_BREAKDOWN_")
		art := pipe.SceneArtifacts[sid]
		if art == nil || art.Breakdown == nil {
			return nil, nil, fmt.Errorf("breakdown %s missing", sid)
		}
		before = art.Breakdown.StoryTime
		if s, ok := t.SuggestedValue.(string); ok {
			art.Breakdown.StoryTime = s
			after = s
			return before, after, nil
		}
		return nil, nil, fmt.Errorf("story_time suggested_value not string")
	case strings.HasPrefix(t.ArtifactID, "SCENE_BREAKDOWN_") && strings.Contains(t.FieldPath, "evidence"):
		return applyEvidenceClamp(pipe, t)
	case strings.HasPrefix(t.ArtifactID, "SCENE_ANALYSIS_") && strings.Contains(t.FieldPath, "beats["):
		return applyBeatClamp(pipe, t)
	case strings.HasPrefix(t.ArtifactID, "SHOT_") && strings.Contains(t.FieldPath, "source_beat_ids"):
		shotID := strings.TrimPrefix(t.ArtifactID, "SHOT_")
		for i := range pipe.ShotList {
			if pipe.ShotList[i].ShotID != shotID {
				continue
			}
			before = append([]string{}, pipe.ShotList[i].SourceBeatIDs...)
			ids := anyToStrings(t.SuggestedValue)
			pipe.ShotList[i].SourceBeatIDs = ids
			after = ids
			return before, after, nil
		}
		return nil, nil, fmt.Errorf("shot %s missing", shotID)
	default:
		return nil, nil, fmt.Errorf("no local applicator for %s %s", t.ArtifactID, t.FieldPath)
	}
}

func applyEvidenceClamp(pipe *ScriptPipeline, t RepairTarget) (before, after any, err error) {
	sid := firstNonEmpty(t.SceneID, strings.TrimPrefix(t.ArtifactID, "SCENE_BREAKDOWN_"))
	art := pipe.SceneArtifacts[sid]
	if art == nil || art.Breakdown == nil {
		return nil, nil, fmt.Errorf("breakdown missing")
	}
	// field_path like breakdown.elements[N].evidence
	idx := indexInPath(t.FieldPath, "elements[")
	if idx < 0 || idx >= len(art.Breakdown.Elements) {
		return nil, nil, fmt.Errorf("element index invalid")
	}
	el := &art.Breakdown.Elements[idx]
	if el.Evidence == nil {
		return nil, nil, fmt.Errorf("evidence missing")
	}
	before = map[string]any{"source_start_line": el.Evidence.SourceStartLine, "source_end_line": el.Evidence.SourceEndLine}
	m, _ := t.SuggestedValue.(map[string]any)
	if m == nil {
		return nil, nil, fmt.Errorf("suggested_value map required")
	}
	el.Evidence.SourceStartLine = asInt(m["source_start_line"])
	el.Evidence.SourceEndLine = asInt(m["source_end_line"])
	after = map[string]any{"source_start_line": el.Evidence.SourceStartLine, "source_end_line": el.Evidence.SourceEndLine}
	return before, after, nil
}

func applyBeatClamp(pipe *ScriptPipeline, t RepairTarget) (before, after any, err error) {
	sid := firstNonEmpty(t.SceneID, strings.TrimPrefix(t.ArtifactID, "SCENE_ANALYSIS_"))
	art := pipe.SceneArtifacts[sid]
	if art == nil || art.Analysis == nil {
		return nil, nil, fmt.Errorf("analysis missing")
	}
	idx := indexInPath(t.FieldPath, "beats[")
	if idx < 0 || idx >= len(art.Analysis.Beats) {
		return nil, nil, fmt.Errorf("beat index invalid")
	}
	b := &art.Analysis.Beats[idx]
	before = map[string]any{"source_start_line": b.SourceStartLine, "source_end_line": b.SourceEndLine}
	m, _ := t.SuggestedValue.(map[string]any)
	if m == nil {
		return nil, nil, fmt.Errorf("suggested_value map required")
	}
	b.SourceStartLine = asInt(m["source_start_line"])
	b.SourceEndLine = asInt(m["source_end_line"])
	after = map[string]any{"source_start_line": b.SourceStartLine, "source_end_line": b.SourceEndLine}
	return before, after, nil
}

func indexInPath(path, marker string) int {
	i := strings.Index(path, marker)
	if i < 0 {
		return -1
	}
	rest := path[i+len(marker):]
	end := strings.Index(rest, "]")
	if end < 0 {
		return -1
	}
	var n int
	fmt.Sscanf(rest[:end], "%d", &n)
	return n
}

func repairArtifactGroup(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, targets []RepairTarget, userNote string) (int, []RepairPatch, error) {
	if len(targets) == 0 {
		return 0, nil, nil
	}
	t0 := targets[0]
	artifact, sceneID, kind := resolveArtifactJSON(pipe, t0)
	if artifact == nil {
		// Still allow character / registry text repair via stub object.
		artifact = map[string]any{"artifact_id": t0.ArtifactID, "note": "partial context"}
		kind = "generic"
	}
	excerpt := sourceExcerpt(pipe, sceneID)
	var targetLines []string
	for _, t := range targets {
		b, _ := json.Marshal(map[string]any{
			"finding_id": t.FindingID, "action": t.Action, "field_path": t.FieldPath,
			"problem": t.Problem, "repair_instruction": t.RepairInstruction,
			"evidence_quote": t.EvidenceQuote, "suggested_value": t.SuggestedValue,
		})
		targetLines = append(targetLines, string(b))
	}
	artJSON, _ := json.MarshalIndent(artifact, "", "  ")
	user := fmt.Sprintf(`Artifact type: %s
Artifact id: %s

## Current artifact JSON
%s

## Source excerpt (numbered; do not invent facts absent here)
%s

## Repair targets (modify ONLY these field_paths)
%s

## User note
%s`,
		kind, t0.ArtifactID, string(artJSON), excerpt, strings.Join(targetLines, "\n"), strings.TrimSpace(userNote))

	resp, err := invokeChat(ctx, llm, model, RepairSystem+localeBlock(pipe.Locale), user, tokenBudget(3500))
	if err != nil {
		return 1, nil, err
	}
	data, err := jsonout.ExtractObject(resp)
	if err != nil {
		return 1, nil, err
	}
	beforeSnap := artifact
	paths := make([]string, 0, len(targets))
	for _, t := range targets {
		if p := strings.TrimSpace(t.FieldPath); p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return 1, nil, fmt.Errorf("no field_path on repair targets")
	}
	if err := mergeArtifactFields(pipe, t0, kind, beforeSnap, data, paths); err != nil {
		return 1, nil, err
	}
	var patches []RepairPatch
	for _, t := range targets {
		patches = append(patches, RepairPatch{
			FindingID: t.FindingID, ArtifactID: t.ArtifactID, FieldPath: t.FieldPath,
			Action: t.Action, Before: summarizePath(beforeSnap, t.FieldPath),
			After: lookupMerged(data, t.FieldPath), AppliedLocally: false,
		})
	}
	return 1, patches, nil
}

func resolveArtifactJSON(pipe *ScriptPipeline, t RepairTarget) (any, string, string) {
	aid := t.ArtifactID
	switch {
	case strings.HasPrefix(aid, "SCENE_BREAKDOWN_"):
		sid := strings.TrimPrefix(aid, "SCENE_BREAKDOWN_")
		if art := pipe.SceneArtifacts[sid]; art != nil {
			return art.Breakdown, sid, "scene_breakdown"
		}
	case strings.HasPrefix(aid, "SCENE_ANALYSIS_"):
		sid := strings.TrimPrefix(aid, "SCENE_ANALYSIS_")
		if art := pipe.SceneArtifacts[sid]; art != nil {
			return art.Analysis, sid, "scene_analysis"
		}
	case strings.HasPrefix(aid, "CHARACTER_"):
		cid := strings.TrimPrefix(aid, "CHARACTER_")
		if pipe.EntityRegistry != nil {
			for i := range pipe.EntityRegistry.Characters {
				if pipe.EntityRegistry.Characters[i].CharacterID == cid {
					return pipe.EntityRegistry.Characters[i], t.SceneID, "character"
				}
			}
		}
	case aid == "ENTITY_REGISTRY":
		return pipe.EntityRegistry, "", "entity_registry"
	case strings.HasPrefix(aid, "SHOT_"):
		sid := strings.TrimPrefix(aid, "SHOT_")
		for i := range pipe.ShotList {
			if pipe.ShotList[i].ShotID == sid {
				return pipe.ShotList[i], pipe.ShotList[i].SceneID, "shot"
			}
		}
	}
	if t.SceneID != "" {
		if art := pipe.SceneArtifacts[t.SceneID]; art != nil && art.Breakdown != nil {
			return art.Breakdown, t.SceneID, "scene_breakdown"
		}
	}
	return nil, t.SceneID, "generic"
}

// mergeArtifactFields applies only whitelist field_paths from patch onto the live artifact.
func mergeArtifactFields(pipe *ScriptPipeline, t RepairTarget, kind string, before any, patch map[string]any, paths []string) error {
	base, _ := toMap(before)
	if base == nil {
		base = map[string]any{}
	}
	merged := shallowCopyMap(base)
	for _, path := range paths {
		val, ok := lookupPath(patch, path)
		if !ok {
			// Allow patch to be a full/partial object rooted at artifact keys.
			val, ok = lookupPath(patch, stripArtifactPrefix(path, kind))
		}
		if !ok {
			continue
		}
		setPath(merged, stripArtifactPrefix(path, kind), val)
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	switch kind {
	case "scene_breakdown":
		sid := firstNonEmpty(t.SceneID, strings.TrimPrefix(t.ArtifactID, "SCENE_BREAKDOWN_"))
		var bd SceneBreakdown
		if err := json.Unmarshal(raw, &bd); err != nil {
			return err
		}
		if pipe.SceneArtifacts[sid] == nil {
			pipe.SceneArtifacts[sid] = &SceneArtifact{SceneID: sid}
		}
		if bd.ArtifactID == "" {
			bd.ArtifactID = "SCENE_BREAKDOWN_" + sid
		}
		bd.SceneID = sid
		pipe.SceneArtifacts[sid].Breakdown = &bd
	case "scene_analysis":
		sid := firstNonEmpty(t.SceneID, strings.TrimPrefix(t.ArtifactID, "SCENE_ANALYSIS_"))
		var an SceneAnalysis
		if err := json.Unmarshal(raw, &an); err != nil {
			return err
		}
		if pipe.SceneArtifacts[sid] == nil {
			pipe.SceneArtifacts[sid] = &SceneArtifact{SceneID: sid}
		}
		if an.ArtifactID == "" {
			an.ArtifactID = "SCENE_ANALYSIS_" + sid
		}
		an.SceneID = sid
		pipe.SceneArtifacts[sid].Analysis = &an
	case "character":
		var ch RegistryCharacter
		if err := json.Unmarshal(raw, &ch); err != nil {
			return err
		}
		if pipe.EntityRegistry == nil {
			return fmt.Errorf("entity registry missing")
		}
		cid := strings.TrimPrefix(t.ArtifactID, "CHARACTER_")
		for i := range pipe.EntityRegistry.Characters {
			if pipe.EntityRegistry.Characters[i].CharacterID == ch.CharacterID ||
				pipe.EntityRegistry.Characters[i].CharacterID == cid {
				if ch.CharacterID == "" {
					ch.CharacterID = pipe.EntityRegistry.Characters[i].CharacterID
				}
				pipe.EntityRegistry.Characters[i] = ch
				return nil
			}
		}
		return fmt.Errorf("character not found")
	case "entity_registry":
		var reg EntityRegistry
		if err := json.Unmarshal(raw, &reg); err != nil {
			return err
		}
		pipe.EntityRegistry = &reg
	case "shot":
		var sh ShotListItem
		if err := json.Unmarshal(raw, &sh); err != nil {
			return err
		}
		sid := strings.TrimPrefix(t.ArtifactID, "SHOT_")
		for i := range pipe.ShotList {
			if pipe.ShotList[i].ShotID == sh.ShotID || pipe.ShotList[i].ShotID == sid {
				if sh.ShotID == "" {
					sh.ShotID = pipe.ShotList[i].ShotID
				}
				pipe.ShotList[i] = sh
				return nil
			}
		}
		return fmt.Errorf("shot not found")
	default:
		return fmt.Errorf("unsupported artifact kind %s", kind)
	}
	return nil
}

func toMap(v any) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func shallowCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func stripArtifactPrefix(path, kind string) string {
	p := strings.TrimSpace(path)
	prefixes := []string{"breakdown.", "analysis.", "entity_registry.", "shot_list[].", "shot."}
	for _, pre := range prefixes {
		if strings.HasPrefix(p, pre) {
			return strings.TrimPrefix(p, pre)
		}
	}
	_ = kind
	return p
}

func lookupPath(m map[string]any, path string) (any, bool) {
	if m == nil || path == "" {
		return nil, false
	}
	parts := splitPath(path)
	var cur any = m
	for _, part := range parts {
		key, idx := parsePathPart(part)
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[key]
		if !ok {
			return nil, false
		}
		if idx >= 0 {
			arr, ok := next.([]any)
			if !ok || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
		} else {
			cur = next
		}
	}
	return cur, true
}

func setPath(m map[string]any, path string, val any) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}
	cur := m
	for i, part := range parts {
		key, idx := parsePathPart(part)
		last := i == len(parts)-1
		if last {
			if idx >= 0 {
				arr, _ := cur[key].([]any)
				for len(arr) <= idx {
					arr = append(arr, nil)
				}
				arr[idx] = val
				cur[key] = arr
				return
			}
			cur[key] = val
			return
		}
		if idx >= 0 {
			arr, _ := cur[key].([]any)
			for len(arr) <= idx {
				arr = append(arr, map[string]any{})
			}
			child, _ := arr[idx].(map[string]any)
			if child == nil {
				child = map[string]any{}
				arr[idx] = child
			}
			cur[key] = arr
			cur = child
			continue
		}
		child, _ := cur[key].(map[string]any)
		if child == nil {
			child = map[string]any{}
			cur[key] = child
		}
		cur = child
	}
}

func splitPath(path string) []string {
	raw := strings.Split(path, ".")
	var out []string
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parsePathPart(part string) (key string, idx int) {
	idx = -1
	if i := strings.Index(part, "["); i >= 0 && strings.HasSuffix(part, "]") {
		key = part[:i]
		fmt.Sscanf(part[i+1:len(part)-1], "%d", &idx)
		return key, idx
	}
	return part, -1
}

func lookupMerged(patch map[string]any, path string) any {
	if v, ok := lookupPath(patch, path); ok {
		return v
	}
	if v, ok := lookupPath(patch, stripArtifactPrefix(path, "")); ok {
		return v
	}
	return nil
}

func sourceExcerpt(pipe *ScriptPipeline, sceneID string) string {
	if pipe.SourceDocument == nil {
		return "(no source)"
	}
	if sceneID == "" {
		// Cap whole-doc excerpt
		lines := pipe.SourceDocument.Lines
		if len(lines) > 80 {
			lines = lines[:80]
		}
		var b strings.Builder
		for i, ln := range lines {
			fmt.Fprintf(&b, "%d|%s\n", i+1, ln)
		}
		return b.String()
	}
	for _, sc := range pipe.SceneIndex {
		if sc.SceneID == sceneID {
			return numberedSlice(pipe.SourceDocument.Lines, sc.SourceStartLine, sc.SourceEndLine)
		}
	}
	return pipe.SourceDocument.NumberedText
}

func summarizePath(artifact any, fieldPath string) any {
	if artifact == nil {
		return nil
	}
	b, _ := json.Marshal(artifact)
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	if fieldPath == "" {
		return nil
	}
	// Best-effort leaf peek for UI; full before is large.
	parts := strings.Split(fieldPath, ".")
	leaf := parts[len(parts)-1]
	if strings.Contains(leaf, "[") {
		leaf = leaf[:strings.Index(leaf, "[")]
	}
	if v, ok := m[leaf]; ok {
		return v
	}
	return fieldPath
}

func markFindingResolved(pipe *ScriptPipeline, id string) {
	for i := range pipe.RuleFindings {
		if pipe.RuleFindings[i].FindingID == id {
			pipe.RuleFindings[i].Status = "resolved"
		}
	}
	if pipe.Continuity != nil {
		for i := range pipe.Continuity.Issues {
			if pipe.Continuity.Issues[i].IssueID == id {
				pipe.Continuity.Issues[i].Status = "resolved"
			}
		}
	}
	if pipe.AuditReport != nil {
		for i := range pipe.AuditReport.Issues {
			if pipe.AuditReport.Issues[i].IssueID == id {
				pipe.AuditReport.Issues[i].Status = "resolved"
			}
		}
	}
}

func anyToStrings(v any) []string {
	if v == nil {
		return nil
	}
	if arr, ok := v.([]string); ok {
		return arr
	}
	if arr, ok := v.([]any); ok {
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}

func clonePipeline(pipe *ScriptPipeline) *ScriptPipeline {
	b, err := json.Marshal(pipe)
	if err != nil {
		cp := *pipe
		return &cp
	}
	var out ScriptPipeline
	if err := json.Unmarshal(b, &out); err != nil {
		cp := *pipe
		return &cp
	}
	return &out
}
