package scriptparse

import (
	"context"
	"fmt"

	"github.com/danmo-ai/danmo-film/core/port"
)

// ProcessOpts controls how far the pipeline runs.
type ProcessOpts struct {
	ScriptText         string
	Title              string
	Locale             string
	Model              string
	Existing           *ScriptPipeline // for incremental
	ExistingRoster     *ExistingRoster // text catalog for Story Map reuse decisions
	StopAt             string          // stage to stop after (default gate1 for facts pass)
	UntilShots         bool            // continue through R5–R7 after facts approved
	TargetDurationSec  float64
	SegmentDurationSec float64
	MaxClipSec         float64
	OnlySceneIDs       []string
	RerunStage         string
}

// RunProcess executes R0→… up to StopAt / UntilShots.
func RunProcess(ctx context.Context, llm port.LLMProvider, opts ProcessOpts, on StageProgressFn) (*ScriptPipeline, error) {
	var pipe *ScriptPipeline
	var err error
	calls := 0
	add := func(n int) { calls += n }

	if opts.RerunStage != "" && opts.Existing != nil {
		pipe = opts.Existing
		pipe.Locale = NormalizeLocale(firstNonEmpty(opts.Locale, pipe.Locale))
		n, err := rerunStage(ctx, llm, opts, pipe, on)
		add(n)
		pipe.LLMCalls = calls
		return pipe, err
	}

	emit(on, StageR0, "", "preprocess", 5)
	pipe, err = Preprocess(opts.ScriptText, opts.Title)
	if err != nil {
		return nil, err
	}
	pipe.Locale = NormalizeLocale(opts.Locale)

	if opts.Existing != nil && opts.Existing.SourceHash != "" {
		pipe.SceneChangeSet = DiffSceneIndex(opts.Existing.SceneIndex, pipe.SceneIndex)
		for _, ch := range pipe.SceneChangeSet {
			if ch.Status != "unchanged" {
				continue
			}
			prev, ok := opts.Existing.SceneArtifacts[ch.SceneID]
			if !ok || prev == nil || prev.Breakdown == nil {
				continue
			}
			pipe.SceneArtifacts[ch.SceneID] = prev
		}
		if opts.Existing.Workflow.FactsApproved {
			pipe.Workflow.FactsApproved = true
		}
		if opts.Existing.StoryMap != nil && allScenesReusable(pipe.SceneChangeSet) {
			pipe.StoryMap = opts.Existing.StoryMap
			pipe.EntityRegistry = opts.Existing.EntityRegistry
			if opts.Existing.Continuity != nil {
				pipe.Continuity = opts.Existing.Continuity
			}
		}
		// Same-script resume after a failed run: keep whatever Existing already finished.
		if opts.Existing.SourceHash == pipe.SourceHash {
			if pipe.StoryMap == nil && opts.Existing.StoryMap != nil {
				pipe.StoryMap = opts.Existing.StoryMap
				pipe.EntityRegistry = opts.Existing.EntityRegistry
			}
			for id, prev := range opts.Existing.SceneArtifacts {
				if prev == nil {
					continue
				}
				cur := pipe.SceneArtifacts[id]
				if cur == nil {
					pipe.SceneArtifacts[id] = prev
					continue
				}
				if cur.Breakdown == nil && prev.Breakdown != nil {
					cur.Breakdown = prev.Breakdown
				}
				if cur.Analysis == nil && prev.Analysis != nil {
					cur.Analysis = prev.Analysis
				}
			}
			if pipe.Continuity == nil {
				pipe.Continuity = opts.Existing.Continuity
			}
			if len(pipe.ShotList) == 0 {
				pipe.ShotList = opts.Existing.ShotList
			}
			if len(pipe.GenPackages) == 0 {
				pipe.GenPackages = opts.Existing.GenPackages
			}
		}
	}

	stop := opts.StopAt
	if stop == "" {
		if opts.UntilShots {
			stop = StageGate2
		} else {
			stop = StageGate1
		}
	}

	pctFor := map[string]int{
		StageR1: 15, StageR2: 40, StageR3: 55, StageR4: 70,
		StageR5: 80, StageR6: 90, StageR7: 95,
	}
	run := func(stage string, fn func() (int, error)) error {
		n, e := fn()
		add(n)
		if e != nil {
			return e
		}
		pipe.LLMCalls = calls
		emitCheckpoint(on, pipe, stage, pctFor[stage])
		return nil
	}

	needStoryMap := pipe.StoryMap == nil
	skipFactsStages := pipe.Workflow.FactsApproved && pipe.Continuity != nil && allScenesReusable(pipe.SceneChangeSet)
	if needStoryMap {
		if err := run(StageR1, func() (int, error) {
			return RunStoryMap(ctx, llm, opts.Model, pipe, opts.ExistingRoster, on)
		}); err != nil {
			pipe.Workflow.Status = "failed"
			pipe.Workflow.Error = err.Error()
			pipe.LLMCalls = calls
			return pipe, err
		}
	}
	if stageOrder(stop) < stageOrder(StageR2) {
		pipe.LLMCalls = calls
		return pipe, nil
	}

	if !skipFactsStages {
		if err := run(StageR2, func() (int, error) {
			return RunBreakdowns(ctx, llm, opts.Model, pipe, opts.OnlySceneIDs, on)
		}); err != nil {
			pipe.Workflow.Status = "failed"
			pipe.Workflow.Error = err.Error()
			pipe.LLMCalls = calls
			return pipe, err
		}
		if stageOrder(stop) < stageOrder(StageR3) {
			pipe.LLMCalls = calls
			return pipe, nil
		}

		if err := run(StageR3, func() (int, error) {
			return RunAnalyses(ctx, llm, opts.Model, pipe, opts.OnlySceneIDs, on)
		}); err != nil {
			pipe.Workflow.Status = "failed"
			pipe.Workflow.Error = err.Error()
			pipe.LLMCalls = calls
			return pipe, err
		}
		if stageOrder(stop) < stageOrder(StageR4) {
			pipe.LLMCalls = calls
			return pipe, nil
		}

		if err := run(StageR4, func() (int, error) {
			return RunContinuity(ctx, llm, opts.Model, pipe, on)
		}); err != nil {
			pipe.Workflow.Status = "failed"
			pipe.Workflow.Error = err.Error()
			pipe.LLMCalls = calls
			return pipe, err
		}
	}

	if !opts.UntilShots && stop == StageGate1 {
		LintPipeline(pipe)
		pipe.LLMCalls = calls
		emit(on, StageGate1, "", "awaiting_facts_review", 100)
		emitCheckpoint(on, pipe, StageGate1, 100)
		return pipe, nil
	}

	if !pipe.Workflow.FactsApproved {
		pipe.LLMCalls = calls
		return pipe, fmt.Errorf("cannot continue to shots: facts gate not approved")
	}

	seg := opts.SegmentDurationSec
	if seg <= 0 {
		seg = 8
	}
	maxClip := opts.MaxClipSec
	if maxClip <= 0 {
		maxClip = 15
	}
	target := opts.TargetDurationSec
	if target <= 0 {
		target = 60
	}

	if err := run(StageR5, func() (int, error) {
		return RunShotLists(ctx, llm, opts.Model, pipe, target, seg, maxClip, opts.OnlySceneIDs, on)
	}); err != nil {
		pipe.Workflow.Status = "failed"
		pipe.Workflow.Error = err.Error()
		pipe.LLMCalls = calls
		return pipe, err
	}

	if err := run(StageR6, func() (int, error) {
		return RunGenPackages(ctx, llm, opts.Model, pipe, on)
	}); err != nil {
		pipe.Workflow.Status = "failed"
		pipe.Workflow.Error = err.Error()
		pipe.LLMCalls = calls
		return pipe, err
	}

	if err := run(StageR7, func() (int, error) {
		return RunAudit(ctx, llm, opts.Model, pipe, on)
	}); err != nil {
		pipe.Workflow.Status = "failed"
		pipe.Workflow.Error = err.Error()
		pipe.LLMCalls = calls
		return pipe, err
	}

	LintPipeline(pipe)
	pipe.LLMCalls = calls
	emit(on, StageGate2, "", "awaiting_shots_review", 100)
	emitCheckpoint(on, pipe, StageGate2, 100)
	return pipe, nil
}

func rerunStage(ctx context.Context, llm port.LLMProvider, opts ProcessOpts, pipe *ScriptPipeline, on StageProgressFn) (int, error) {
	seg := opts.SegmentDurationSec
	if seg <= 0 {
		seg = 8
	}
	maxClip := opts.MaxClipSec
	if maxClip <= 0 {
		maxClip = 15
	}
	target := opts.TargetDurationSec
	if target <= 0 {
		target = 60
	}
	switch opts.RerunStage {
	case StageR1:
		return RunStoryMap(ctx, llm, opts.Model, pipe, opts.ExistingRoster, on)
	case StageR2:
		invalidateDownstreamFrom(pipe, StageR2, opts.OnlySceneIDs)
		return RunBreakdowns(ctx, llm, opts.Model, pipe, opts.OnlySceneIDs, on)
	case StageR3:
		invalidateDownstreamFrom(pipe, StageR3, opts.OnlySceneIDs)
		return RunAnalyses(ctx, llm, opts.Model, pipe, opts.OnlySceneIDs, on)
	case StageR4:
		invalidateDownstreamFrom(pipe, StageR4, nil)
		return RunContinuity(ctx, llm, opts.Model, pipe, on)
	case StageR5:
		invalidateDownstreamFrom(pipe, StageR5, opts.OnlySceneIDs)
		return RunShotLists(ctx, llm, opts.Model, pipe, target, seg, maxClip, opts.OnlySceneIDs, on)
	case StageR6:
		return RunGenPackages(ctx, llm, opts.Model, pipe, on)
	case StageR7:
		return RunAudit(ctx, llm, opts.Model, pipe, on)
	default:
		return 0, fmt.Errorf("unknown rerun stage %q", opts.RerunStage)
	}
}

func invalidateDownstreamFrom(pipe *ScriptPipeline, from string, sceneIDs []string) {
	ord := stageOrder(from)
	if ord <= stageOrder(StageR2) {
		for _, id := range sceneIDsOrAll(pipe, sceneIDs) {
			if art := pipe.SceneArtifacts[id]; art != nil {
				art.Breakdown = nil
				art.Analysis = nil
				art.ReviewStatus = ReviewNeedsRevision
			}
		}
	}
	if ord <= stageOrder(StageR3) {
		for _, id := range sceneIDsOrAll(pipe, sceneIDs) {
			if art := pipe.SceneArtifacts[id]; art != nil {
				art.Analysis = nil
			}
		}
	}
	if ord <= stageOrder(StageR4) {
		pipe.Continuity = nil
		pipe.Workflow.FactsApproved = false
	}
	if ord <= stageOrder(StageR5) {
		if len(sceneIDs) == 0 {
			pipe.ShotList = nil
			pipe.GenPackages = nil
		} else {
			keep := map[string]bool{}
			for _, id := range sceneIDs {
				keep[id] = true
			}
			var shots []ShotListItem
			for _, sh := range pipe.ShotList {
				if !keep[sh.SceneID] {
					shots = append(shots, sh)
				} else {
					sh.ReviewStatus = ReviewNeedsRevision
				}
			}
			pipe.ShotList = shots
			var packs []GenerationPackage
			shotKeep := map[string]bool{}
			for _, sh := range pipe.ShotList {
				shotKeep[sh.ShotID] = true
			}
			for _, gp := range pipe.GenPackages {
				if shotKeep[gp.ShotID] {
					packs = append(packs, gp)
				} else {
					gp.ReviewStatus = ReviewObsolete
				}
			}
			pipe.GenPackages = packs
		}
		pipe.Workflow.ShotsApproved = false
	}
	if ord <= stageOrder(StageR7) {
		pipe.AuditReport = nil
	}
}

func sceneIDsOrAll(pipe *ScriptPipeline, ids []string) []string {
	if len(ids) > 0 {
		return ids
	}
	var out []string
	for _, sc := range pipe.SceneIndex {
		out = append(out, sc.SceneID)
	}
	return out
}

func stageOrder(s string) int {
	order := map[string]int{
		StageR0: 0, StageR1: 1, StageR2: 2, StageR3: 3, StageR4: 4,
		StageGate1: 5, StageR5: 6, StageR6: 7, StageR7: 8, StageGate2: 9, StageDone: 10,
	}
	return order[s]
}

func allScenesReusable(changes []SceneChange) bool {
	if len(changes) == 0 {
		return false
	}
	for _, ch := range changes {
		if ch.Status != "unchanged" && ch.Status != "moved" {
			return false
		}
	}
	return true
}

// ApproveFacts marks Gate1 passed and stamps registry + scene artifacts approved.
func ApproveFacts(pipe *ScriptPipeline) error {
	if pipe == nil {
		return fmt.Errorf("nil pipeline")
	}
	if pipe.StoryMap == nil || len(pipe.SceneArtifacts) == 0 {
		return fmt.Errorf("pipeline incomplete for facts approval")
	}
	pipe.Workflow.FactsApproved = true
	pipe.Workflow.AwaitingReview = ""
	pipe.Workflow.Status = "completed"
	pipe.Workflow.Stage = StageGate1
	if pipe.StoryMap != nil {
		pipe.StoryMap.ReviewStatus = ReviewApproved
	}
	if pipe.Continuity != nil {
		pipe.Continuity.ReviewStatus = ReviewApproved
	}
	for _, art := range pipe.SceneArtifacts {
		art.ReviewStatus = ReviewApproved
		if art.Breakdown != nil {
			art.Breakdown.ReviewStatus = ReviewApproved
		}
		if art.Analysis != nil {
			art.Analysis.ReviewStatus = ReviewApproved
		}
	}
	if pipe.EntityRegistry != nil {
		for i := range pipe.EntityRegistry.Characters {
			pipe.EntityRegistry.Characters[i].ReviewStatus = ReviewApproved
		}
		for i := range pipe.EntityRegistry.Locations {
			pipe.EntityRegistry.Locations[i].ReviewStatus = ReviewApproved
		}
		for i := range pipe.EntityRegistry.Props {
			pipe.EntityRegistry.Props[i].ReviewStatus = ReviewApproved
		}
	}
	return nil
}

// ApproveShots marks Gate2 passed.
func ApproveShots(pipe *ScriptPipeline) error {
	if pipe == nil {
		return fmt.Errorf("nil pipeline")
	}
	if !pipe.Workflow.FactsApproved || len(pipe.ShotList) == 0 {
		return fmt.Errorf("shots not ready for approval")
	}
	// Block if open blockers in audit.
	if pipe.AuditReport != nil {
		for _, iss := range pipe.AuditReport.Issues {
			if iss.Severity == "blocker" && iss.Status != "resolved" {
				return fmt.Errorf("audit blockers remain")
			}
		}
	}
	pipe.Workflow.ShotsApproved = true
	pipe.Workflow.AwaitingReview = ""
	pipe.Workflow.Status = "completed"
	pipe.Workflow.Stage = StageDone
	for i := range pipe.ShotList {
		pipe.ShotList[i].ReviewStatus = ReviewApproved
	}
	for i := range pipe.GenPackages {
		pipe.GenPackages[i].ReviewStatus = ReviewApproved
	}
	if pipe.AuditReport != nil {
		pipe.AuditReport.ReviewStatus = ReviewApproved
	}
	return nil
}

// ShotsFromPipeline maps approved gen packages into legacy shot maps for the editor.
func ShotsFromPipeline(pipe *ScriptPipeline) []map[string]any {
	packByShot := map[string]GenerationPackage{}
	for _, gp := range pipe.GenPackages {
		packByShot[gp.ShotID] = gp
	}
	nameByID := map[string]string{}
	lookByID := map[string]RegistryCharacter{}
	if pipe.EntityRegistry != nil {
		for _, c := range pipe.EntityRegistry.Characters {
			nameByID[c.CharacterID] = c.Name
			lookByID[c.CharacterID] = c
		}
	}
	var shots []map[string]any
	for i, sh := range pipe.ShotList {
		gp := packByShot[sh.ShotID]
		var castLooks []any
		if pipe.EntityRegistry != nil {
			for _, name := range sh.CharactersOnScreen {
				for _, c := range pipe.EntityRegistry.Characters {
					if c.Name == name {
						castLooks = append(castLooks, map[string]any{
							"character_id": c.CharacterID,
							"look_id":      defaultLookID(c.CharacterID, c.Looks),
						})
						break
					}
				}
			}
		}
		sceneLook := map[string]any{}
		if pipe.EntityRegistry != nil {
			for _, loc := range pipe.EntityRegistry.Locations {
				if gp.LocationID != "" && loc.LocationID == gp.LocationID {
					sceneLook = map[string]any{
						"scene_id": loc.LocationID,
						"look_id":  defaultLookID(loc.LocationID, loc.Looks),
					}
					break
				}
			}
			if len(sceneLook) == 0 && len(pipe.EntityRegistry.Locations) > 0 {
				loc := pipe.EntityRegistry.Locations[0]
				sceneLook = map[string]any{
					"scene_id": loc.LocationID,
					"look_id":  defaultLookID(loc.LocationID, loc.Looks),
				}
			}
		}
		beatIdx := 0
		for bi, sc := range pipe.SceneIndex {
			if sc.SceneID == sh.SceneID {
				beatIdx = bi
				break
			}
		}
		shots = append(shots, map[string]any{
			"id":                   sh.ShotID,
			"order":                i,
			"duration_sec":         sh.DurationSec,
			"shot_size":            sh.ShotSize,
			"camera_movement":      sh.CameraMovement,
			"visual_prompt":        firstNonEmpty(gp.KeyframePrompt, sh.VisualAction),
			"start_visual_prompt":  firstNonEmpty(gp.StartVisual, gp.KeyframePrompt),
			"video_prompt":         firstNonEmpty(gp.VideoPrompt, sh.VisualAction),
			"motion_prompt":        gp.VideoPrompt,
			"scene_prompt":         gp.KeyframePrompt,
			"dialogue":             sh.Dialogue,
			"narrative_beat_index": beatIdx,
			"segment_role":         "keyframe",
			"cast_looks":           castLooks,
			"scene_look":           sceneLook,
			"characters_on_screen": toStrAny(sh.CharactersOnScreen),
			"continuity_notes":     sh.ContinuityNotes,
			"negative_prompt":      gp.NegativePrompt,
		})
	}
	return shots
}

// RosterFromPipeline builds character/scene DTOs for the cast/scenes tabs after Gate1.
func RosterFromPipeline(pipe *ScriptPipeline) (characters, scenes []map[string]any) {
	if pipe.EntityRegistry == nil {
		return nil, nil
	}
	locale := ""
	if pipe != nil {
		locale = pipe.Locale
	}
	defLabel := defaultLookLabel(locale)
	for _, c := range pipe.EntityRegistry.Characters {
		looks := make([]any, 0, max(len(c.Looks), 1))
		for _, lk := range c.Looks {
			lookRow := map[string]any{
				"id":       firstNonEmpty(lk.LookID, c.CharacterID+"_default"),
				"label":    firstNonEmpty(lk.Label, defLabel),
				"body":     composeCharacterLookBody(lk),
				"decision": firstNonEmpty(lk.Decision, c.Decision),
				"scope":    firstNonEmpty(lk.Scope, c.Scope),
			}
			if episodeOnlyFromScope(lk.Scope, lk.Decision) {
				lookRow["episode_only"] = true
			}
			looks = append(looks, lookRow)
		}
		if len(looks) == 0 {
			lookID := c.CharacterID + "_default"
			lookRow := map[string]any{"id": lookID, "label": defLabel, "body": c.LookBody}
			if episodeOnlyFromScope(c.Scope, c.Decision) {
				lookRow["episode_only"] = true
			}
			looks = append(looks, lookRow)
		}
		defLook := defaultLookID(c.CharacterID, c.Looks)
		row := map[string]any{
			"id":                  c.CharacterID,
			"name":                c.Name,
			"role":                firstNonEmpty(c.Role, "supporting"),
			"aliases":             c.Aliases,
			"age_range":           c.AgeRange,
			"gender_presentation": c.GenderPresentation,
			"identity_baseline":   c.IdentityBaseline,
			"visual_identity":     c.VisualIdentity,
			"voice_manner":        c.VoiceManner,
			"performance_notes":   c.PerformanceNotes,
			"default_look_id":     defLook,
			"looks":               looks,
			"decision":            c.Decision,
			"scope":               c.Scope,
		}
		if episodeOnlyFromScope(c.Scope, c.Decision) {
			row["episode_only"] = true
		}
		characters = append(characters, row)
	}
	for _, loc := range pipe.EntityRegistry.Locations {
		looks := make([]any, 0, max(len(loc.Looks), 1))
		for _, lk := range loc.Looks {
			lookRow := map[string]any{
				"id":       firstNonEmpty(lk.LookID, loc.LocationID+"_default"),
				"label":    firstNonEmpty(lk.Label, defLabel),
				"body":     firstNonEmpty(lk.Body, loc.Body),
				"decision": firstNonEmpty(lk.Decision, loc.Decision),
				"scope":    firstNonEmpty(lk.Scope, loc.Scope),
			}
			if episodeOnlyFromScope(lk.Scope, lk.Decision) {
				lookRow["episode_only"] = true
			}
			looks = append(looks, lookRow)
		}
		if len(looks) == 0 {
			lookID := loc.LocationID + "_default"
			lookRow := map[string]any{"id": lookID, "label": defLabel, "body": loc.Body}
			if episodeOnlyFromScope(loc.Scope, loc.Decision) {
				lookRow["episode_only"] = true
			}
			looks = append(looks, lookRow)
		}
		defLook := defaultLookID(loc.LocationID, loc.Looks)
		row := map[string]any{
			"id":              loc.LocationID,
			"name":            loc.Name,
			"default_look_id": defLook,
			"looks":           looks,
			"decision":        loc.Decision,
			"scope":           loc.Scope,
		}
		if episodeOnlyFromScope(loc.Scope, loc.Decision) {
			row["episode_only"] = true
		}
		scenes = append(scenes, row)
	}
	return characters, scenes
}

// SceneBeatsFromPipeline flattens analysis beats for legacy UI/checklist.
func SceneBeatsFromPipeline(pipe *ScriptPipeline) []map[string]any {
	var out []map[string]any
	order := 0
	for _, sc := range pipe.SceneIndex {
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil || art.Analysis == nil || len(art.Analysis.Beats) == 0 {
			order++
			out = append(out, map[string]any{
				"order": order,
				"title": sc.Heading,
				"beat":  firstNonEmpty(sc.Synopsis, sc.Heading),
			})
			continue
		}
		for _, b := range art.Analysis.Beats {
			order++
			out = append(out, map[string]any{
				"order": order,
				"title": firstNonEmpty(b.BeatType, sc.Heading),
				"beat":  fmt.Sprintf("%s — %s", sc.LocationText, b.SourceEvent),
			})
		}
	}
	return out
}
