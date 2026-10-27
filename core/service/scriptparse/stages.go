package scriptparse

import (
	"context"
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/domain/jsonout"
	"github.com/danmo-ai/danmo-film/core/port"
)

type ProgressEvent struct {
	Stage      string          `json:"stage"`
	SceneID    string          `json:"scene_id,omitempty"`
	Message    string          `json:"message"`
	Pct        int             `json:"pct,omitempty"`
	Checkpoint bool            `json:"checkpoint,omitempty"`
	Pipeline   *ScriptPipeline `json:"script_pipeline,omitempty"`
}

type StageProgressFn func(ev ProgressEvent)

func emit(on StageProgressFn, stage, sceneID, msg string, pct int) {
	if on != nil {
		on(ProgressEvent{Stage: stage, SceneID: sceneID, Message: msg, Pct: pct})
	}
}

func emitCheckpoint(on StageProgressFn, pipe *ScriptPipeline, stage string, pct int) {
	if on == nil || pipe == nil {
		return
	}
	on(ProgressEvent{
		Stage: stage, Message: "checkpoint", Pct: pct,
		Checkpoint: true, Pipeline: pipe,
	})
}

func RunStoryMap(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, existing *ExistingRoster, on StageProgressFn) (int, error) {
	emit(on, StageR1, "", "story_map", 10)
	loc := NormalizeLocale(pipe.Locale)
	idxLines := make([]string, 0, len(pipe.SceneIndex))
	for _, sc := range pipe.SceneIndex {
		idxLines = append(idxLines, fmt.Sprintf("%s %s (L%d–%d)", sc.SceneID, sc.Heading, sc.SourceStartLine, sc.SourceEndLine))
	}
	rosterBlock := formatExistingRosterBlock(existing)
	if rosterBlock == "" {
		rosterBlock = "(none — create all characters and locations)"
	}
	user := fmt.Sprintf("Title: %s\nScene index:\n%s\n\nNumbered script:\n%s\n\n%s\n\n%s",
		pipe.Title, strings.Join(idxLines, "\n"), truncRunes(pipe.SourceDocument.NumberedText, 12000),
		rosterBlock, localeBlock(loc))
	resp, n, err := invokeWithReview(ctx, llm, model, StoryMapSystem, user, tokenBudget(4000), func(s string) (bool, string) {
		data, err := jsonout.ExtractObject(s)
		if err != nil {
			return false, err.Error()
		}
		if asString(data["logline"]) == "" {
			return false, "logline required"
		}
		chars := asMaps(data["characters"])
		if len(chars) < 1 {
			return false, "characters required"
		}
		protag := 0
		for _, c := range chars {
			if asString(c["role"]) == "protagonist" {
				protag++
			}
		}
		if protag != 1 {
			return false, fmt.Sprintf("expected 1 protagonist, got %d", protag)
		}
		return true, ""
	}, adaptProgress(on, StageR1), "story_map")
	if err != nil {
		return n, err
	}
	data, err := jsonout.ExtractObject(resp)
	if err != nil {
		return n, err
	}
	sm := &StoryMap{
		Genre:         asString(data["genre"]),
		Logline:       asString(data["logline"]),
		Mood:          asString(data["mood"]),
		StyleAnchor:   asString(data["style_anchor"]),
		CoreConflict:  asString(data["core_conflict"]),
		Hypotheses:    asStrings(data["hypotheses"]),
		OpenQuestions: asStrings(data["open_questions"]),
		ReviewStatus:  ReviewPending,
	}
	lookLabel := defaultLookLabel(loc)
	for _, c := range asMaps(data["characters"]) {
		name := asString(c["name"])
		if name == "" {
			continue
		}
		dec := sanitizeDecision(asString(c["decision"]))
		if dec == "" {
			if strings.TrimSpace(asString(c["existing_character_id"])) != "" {
				dec = DecisionReuse
			} else {
				dec = DecisionCreate
			}
		}
		scope := sanitizeScope(asString(c["scope"]), dec)
		existID := strings.TrimSpace(asString(c["existing_character_id"]))
		rc := RegistryCharacter{
			Decision:            dec,
			ExistingCharacterID: existID,
			Scope:               scope,
			Name:                name,
			Aliases:             asStrings(c["aliases"]),
			Role:                sanitizeCastRole(asString(c["role"])),
			Basis:               "explicit",
			AgeRange:            strings.TrimSpace(asString(c["age_range"])),
			GenderPresentation:  strings.TrimSpace(asString(c["gender_presentation"])),
			IdentityBaseline:    strings.TrimSpace(asString(c["identity_baseline"])),
			VisualIdentity:      strings.TrimSpace(asString(c["visual_identity"])),
			VoiceManner:         strings.TrimSpace(asString(c["voice_manner"])),
			PerformanceNotes:    strings.TrimSpace(asString(c["performance_notes"])),
			LookBody:            asString(c["look_body"]),
			ReviewStatus:        ReviewPending,
		}
		rc.CharacterID = resolveCharacterID(rc, name)
		prefix := rc.CharacterID + "_look_"
		rc.Looks = parseRegistryLooks(asMaps(c["looks"]), rc.LookBody, lookLabel, prefix)
		normalizeCharacterLooks(&rc, loc)
		sm.Characters = append(sm.Characters, rc)
	}
	for _, locn := range asMaps(data["locations"]) {
		name := asString(locn["name"])
		if name == "" {
			continue
		}
		dec := sanitizeDecision(asString(locn["decision"]))
		if dec == "" {
			if strings.TrimSpace(asString(locn["existing_location_id"])) != "" {
				dec = DecisionReuse
			} else {
				dec = DecisionCreate
			}
		}
		scope := sanitizeScope(asString(locn["scope"]), dec)
		existID := strings.TrimSpace(asString(locn["existing_location_id"]))
		rl := RegistryLocation{
			Decision:           dec,
			ExistingLocationID: existID,
			Scope:              scope,
			Name:               name,
			Body:               asString(locn["body"]),
			ReviewStatus:       ReviewPending,
		}
		rl.LocationID = resolveLocationID(rl, name)
		prefix := rl.LocationID + "_look_"
		rl.Looks = parseRegistryLooks(asMaps(locn["looks"]), rl.Body, lookLabel, prefix)
		normalizeLocationLooks(&rl, loc)
		sm.Locations = append(sm.Locations, rl)
	}
	pipe.StoryMap = sm
	pipe.EntityRegistry = &EntityRegistry{
		Characters: append([]RegistryCharacter{}, sm.Characters...),
		Locations:  append([]RegistryLocation{}, sm.Locations...),
	}
	pipe.Workflow.Stage = StageR1
	pipe.Workflow.Status = "completed"
	return n, nil
}

func RunBreakdowns(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, onlySceneIDs []string, on StageProgressFn) (int, error) {
	calls := 0
	loc := NormalizeLocale(pipe.Locale)
	charDict := formatRegistryChars(pipe.EntityRegistry)
	targets := filterScenes(pipe.SceneIndex, onlySceneIDs, pipe.SceneChangeSet, pipe.SceneArtifacts, "breakdown")
	total := len(targets)
	for i, sc := range targets {
		pct := 20 + (i+1)*30/max(total, 1)
		emit(on, StageR2, sc.SceneID, "scene_breakdown", pct)
		numbered := numberedSlice(pipe.SourceDocument.Lines, sc.SourceStartLine, sc.SourceEndLine)
		user := fmt.Sprintf("SceneID: %s\nCharacter registry:\n%s\n\nNumbered scene text:\n%s\n%s",
			sc.SceneID, charDict, numbered, localeBlock(loc))
		resp, n, err := invokeWithReview(ctx, llm, model, BreakdownSystem, user, tokenBudget(3000), func(s string) (bool, string) {
			data, err := jsonout.ExtractObject(s)
			if err != nil {
				return false, err.Error()
			}
			if len(asStrings(data["cast_present"])) < 1 && len(asMaps(data["elements"])) < 1 {
				return false, "cast_present or elements required"
			}
			return true, ""
		}, adaptProgress(on, StageR2), "breakdown_"+sc.SceneID)
		calls += n
		if err != nil {
			return calls, fmt.Errorf("breakdown %s: %w", sc.SceneID, err)
		}
		data, err := jsonout.ExtractObject(resp)
		if err != nil {
			return calls, err
		}
		bd := &SceneBreakdown{
			ArtifactID:          "SCENE_BREAKDOWN_" + sc.SceneID,
			SceneID:             sc.SceneID,
			Heading:             firstNonEmpty(asString(data["heading"]), sc.Heading),
			IntExt:              firstNonEmpty(asString(data["int_ext"]), sc.IntExt),
			StoryTime:           firstNonEmpty(asString(data["story_time"]), sc.StoryTime),
			CastPresent:         asStrings(data["cast_present"]),
			MentionedNotPresent: asStrings(data["mentioned_not_present"]),
			ReviewQuestions:     asStrings(data["review_questions"]),
			ReviewStatus:        ReviewPending,
		}
		for _, el := range asMaps(data["elements"]) {
			evMap, _ := el["evidence"].(map[string]any)
			var ev *Evidence
			if evMap != nil {
				ev = &Evidence{
					ScriptVersion:   pipe.VersionID,
					SceneID:         sc.SceneID,
					SourceStartLine: asInt(evMap["source_start_line"]),
					SourceEndLine:   asInt(evMap["source_end_line"]),
					Quote:           asString(evMap["quote"]),
				}
			}
			name := asString(el["name"])
			bd.Elements = append(bd.Elements, BreakdownElement{
				ElementID:        stableID("el_", sc.SceneID+"_"+name),
				Category:         asString(el["category"]),
				Name:             name,
				Basis:            firstNonEmpty(asString(el["basis"]), "explicit"),
				Evidence:         ev,
				ContinuityStates: asStrings(el["continuity_states"]),
				ReviewStatus:     ReviewPending,
			})
		}
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil {
			art = &SceneArtifact{SceneID: sc.SceneID}
			pipe.SceneArtifacts[sc.SceneID] = art
		}
		art.Breakdown = bd
		art.ReviewStatus = ReviewPending
		mergePropsFromBreakdown(pipe, bd)
	}
	pipe.Workflow.Stage = StageR2
	pipe.Workflow.Status = "completed"
	return calls, nil
}

func RunAnalyses(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, onlySceneIDs []string, on StageProgressFn) (int, error) {
	calls := 0
	loc := NormalizeLocale(pipe.Locale)
	smJSON := "{}"
	if pipe.StoryMap != nil {
		smJSON = fmt.Sprintf("logline=%s conflict=%s", pipe.StoryMap.Logline, pipe.StoryMap.CoreConflict)
	}
	targets := filterScenes(pipe.SceneIndex, onlySceneIDs, pipe.SceneChangeSet, pipe.SceneArtifacts, "analysis")
	total := len(targets)
	for i, sc := range targets {
		pct := 50 + (i+1)*15/max(total, 1)
		emit(on, StageR3, sc.SceneID, "scene_analysis", pct)
		art := pipe.SceneArtifacts[sc.SceneID]
		bdSummary := ""
		if art != nil && art.Breakdown != nil {
			bdSummary = fmt.Sprintf("cast=%v elements=%d", art.Breakdown.CastPresent, len(art.Breakdown.Elements))
		}
		numbered := numberedSlice(pipe.SourceDocument.Lines, sc.SourceStartLine, sc.SourceEndLine)
		user := fmt.Sprintf("Story map: %s\nBreakdown: %s\nSceneID: %s\n\nNumbered scene:\n%s\n%s",
			smJSON, bdSummary, sc.SceneID, numbered, localeBlock(loc))
		resp, n, err := invokeWithReview(ctx, llm, model, AnalysisSystem, user, tokenBudget(3000), func(s string) (bool, string) {
			data, err := jsonout.ExtractObject(s)
			if err != nil {
				return false, err.Error()
			}
			if len(asMaps(data["beats"])) < 1 {
				return false, "beats required"
			}
			return true, ""
		}, adaptProgress(on, StageR3), "analysis_"+sc.SceneID)
		calls += n
		if err != nil {
			return calls, fmt.Errorf("analysis %s: %w", sc.SceneID, err)
		}
		data, err := jsonout.ExtractObject(resp)
		if err != nil {
			return calls, err
		}
		sa := &SceneAnalysis{
			ArtifactID:       "SCENE_ANALYSIS_" + sc.SceneID,
			SceneID:          sc.SceneID,
			DramaticFunction: asString(data["dramatic_function"]),
			EntryState:       asString(data["entry_state"]),
			ExitState:        asString(data["exit_state"]),
			Deletability:     asString(data["deletability"]),
			Confidence:       asString(data["confidence"]),
			OpenQuestions:    asStrings(data["open_questions"]),
			ReviewStatus:     ReviewPending,
		}
		for bi, b := range asMaps(data["beats"]) {
			sa.Beats = append(sa.Beats, SceneBeat{
				BeatID:          fmt.Sprintf("%s_B%02d", sc.SceneID, bi+1),
				SourceEvent:     asString(b["source_event"]),
				BeatType:        asString(b["beat_type"]),
				Effect:          asString(b["effect"]),
				SourceStartLine: asInt(b["source_start_line"]),
				SourceEndLine:   asInt(b["source_end_line"]),
				Interpretive:    true,
			})
		}
		if art == nil {
			art = &SceneArtifact{SceneID: sc.SceneID}
			pipe.SceneArtifacts[sc.SceneID] = art
		}
		art.Analysis = sa
	}
	pipe.Workflow.Stage = StageR3
	pipe.Workflow.Status = "completed"
	return calls, nil
}

func RunContinuity(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, on StageProgressFn) (int, error) {
	emit(on, StageR4, "", "continuity", 70)
	// Deterministic matrices first.
	castMatrix := map[string][]string{}
	propMatrix := map[string][]string{}
	nameToID := map[string]string{}
	if pipe.EntityRegistry != nil {
		for _, c := range pipe.EntityRegistry.Characters {
			nameToID[c.Name] = c.CharacterID
			for _, a := range c.Aliases {
				nameToID[a] = c.CharacterID
			}
		}
	}
	for _, sc := range pipe.SceneIndex {
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil || art.Breakdown == nil {
			continue
		}
		for _, name := range art.Breakdown.CastPresent {
			id := nameToID[name]
			if id == "" {
				id = stableID("char_", name)
			}
			castMatrix[id] = appendUnique(castMatrix[id], sc.SceneID)
		}
		for _, el := range art.Breakdown.Elements {
			if el.Category == "prop" || el.Category == "wardrobe" || el.Category == "makeup" {
				propMatrix[el.ElementID] = appendUnique(propMatrix[el.ElementID], sc.SceneID)
			}
		}
	}

	loc := NormalizeLocale(pipe.Locale)
	logline := ""
	if pipe.StoryMap != nil {
		logline = pipe.StoryMap.Logline
	}
	user := fmt.Sprintf("Story: %s\n\nScene breakdown summaries:\n%s\n%s",
		logline, continuitySceneSummaries(pipe), localeBlock(loc))
	resp, n, err := invokeWithReview(ctx, llm, model, ContinuitySystem, user, tokenBudget(3000), func(s string) (bool, string) {
		_, err := jsonout.ExtractObject(s)
		if err != nil {
			return false, err.Error()
		}
		return true, ""
	}, adaptProgress(on, StageR4), "continuity")
	if err != nil {
		return n, err
	}
	data, _ := jsonout.ExtractObject(resp)
	report := &ContinuityReport{
		ArtifactID:       "CONTINUITY_" + pipe.VersionID,
		MergeSuggestions: asStrings(data["merge_suggestions"]),
		CastMatrix:       castMatrix,
		PropMatrix:       propMatrix,
		ReviewStatus:     ReviewPending,
	}
	for i, iss := range asMaps(data["issues"]) {
		report.Issues = append(report.Issues, ContinuityIssue{
			IssueID:    fmt.Sprintf("CONT_%03d", i+1),
			Severity:   firstNonEmpty(asString(iss["severity"]), "minor"),
			Action:     firstNonEmpty(asString(iss["action"]), ActionFillField),
			SceneID:    asString(iss["scene_id"]),
			FieldPath:  asString(iss["field_path"]),
			Problem:    asString(iss["problem"]),
			Evidence:   asString(iss["evidence_quote"]),
			RepairHint: asString(iss["repair_instruction"]),
			Status:     "open",
		})
	}
	pipe.Continuity = report
	if pipe.Workflow.FactsApproved {
		pipe.Workflow.Stage = StageGate1
		pipe.Workflow.Status = "completed"
		pipe.Workflow.AwaitingReview = ""
	} else {
		pipe.Workflow.Stage = StageGate1
		pipe.Workflow.Status = "awaiting_review"
		pipe.Workflow.AwaitingReview = "facts"
	}
	return n, nil
}

func RunShotLists(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, targetDur, segDur, maxClip float64, onlySceneIDs []string, on StageProgressFn) (int, error) {
	if !pipe.Workflow.FactsApproved {
		return 0, fmt.Errorf("facts gate not approved")
	}
	calls := 0
	loc := NormalizeLocale(pipe.Locale)
	targets := filterScenes(pipe.SceneIndex, onlySceneIDs, nil, pipe.SceneArtifacts, "")
	var allShots []ShotListItem
	shotN := 0
	for i, sc := range targets {
		emit(on, StageR5, sc.SceneID, "shot_list", 75+(i+1)*5/max(len(targets), 1))
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil || art.Analysis == nil {
			continue
		}
		beatLines := make([]string, 0, len(art.Analysis.Beats))
		for _, b := range art.Analysis.Beats {
			beatLines = append(beatLines, fmt.Sprintf("%s: %s (%s)", b.BeatID, b.SourceEvent, b.BeatType))
		}
		bd := ""
		if art.Breakdown != nil {
			bd = fmt.Sprintf("cast=%v", art.Breakdown.CastPresent)
		}
		user := fmt.Sprintf("Director constraints: target_total≈%gs segment_hint=%gs max_clip=%gs\nScene breakdown: %s\nBeats:\n%s\nScene text:\n%s\n%s",
			targetDur, segDur, maxClip, bd, strings.Join(beatLines, "\n"),
			truncRunes(sc.Text, 3000), localeBlock(loc))
		resp, n, err := invokeWithReview(ctx, llm, model, ShotListSystem, user, tokenBudget(3500), func(s string) (bool, string) {
			data, err := jsonout.ExtractObject(s)
			if err != nil {
				return false, err.Error()
			}
			if len(asMaps(data["shots"])) < 1 {
				return false, "shots required"
			}
			return true, ""
		}, adaptProgress(on, StageR5), "shots_"+sc.SceneID)
		calls += n
		if err != nil {
			return calls, fmt.Errorf("shots %s: %w", sc.SceneID, err)
		}
		data, err := jsonout.ExtractObject(resp)
		if err != nil {
			return calls, err
		}
		for _, row := range asMaps(data["shots"]) {
			shotN++
			dur := asFloat(row["duration_sec"], segDur)
			if maxClip > 0 && dur > maxClip {
				dur = maxClip
			}
			if dur < 4 {
				dur = 4
			}
			allShots = append(allShots, ShotListItem{
				ShotID:             fmt.Sprintf("%s_SH%02d", sc.SceneID, shotN),
				SceneID:            sc.SceneID,
				SourceBeatIDs:      asStrings(row["source_beat_ids"]),
				ShotSize:           asString(row["shot_size"]),
				CameraAngle:        asString(row["camera_angle"]),
				CameraMovement:     firstNonEmpty(asString(row["camera_movement"]), "static"),
				VisualAction:       asString(row["visual_action"]),
				Dialogue:           asString(row["dialogue"]),
				DurationSec:        dur,
				ContinuityNotes:    asString(row["continuity_notes"]),
				Basis:              "proposed",
				CharactersOnScreen: asStrings(row["characters_on_screen"]),
				ReviewStatus:       ReviewPending,
			})
		}
	}
	pipe.ShotList = allShots
	pipe.Workflow.Stage = StageR5
	pipe.Workflow.Status = "completed"
	return calls, nil
}

func RunGenPackages(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, on StageProgressFn) (int, error) {
	calls := 0
	loc := NormalizeLocale(pipe.Locale)
	style := ""
	if pipe.StoryMap != nil {
		style = pipe.StoryMap.StyleAnchor
	}
	lookByName := map[string]string{}
	if pipe.EntityRegistry != nil {
		for _, c := range pipe.EntityRegistry.Characters {
			lookByName[c.Name] = primaryLookBody(c)
		}
	}
	var packs []GenerationPackage
	total := len(pipe.ShotList)
	for i, sh := range pipe.ShotList {
		emit(on, StageR6, sh.SceneID, "generation_package", 85+(i+1)*5/max(total, 1))
		var lookLines []string
		for _, name := range sh.CharactersOnScreen {
			lookLines = append(lookLines, name+": "+lookByName[name])
		}
		user := fmt.Sprintf("Style bible: %s\nAsset looks:\n%s\nShot: size=%s move=%s action=%s dialogue=%s duration=%gs notes=%s\n%s",
			style, strings.Join(lookLines, "\n"), sh.ShotSize, sh.CameraMovement, sh.VisualAction, sh.Dialogue, sh.DurationSec, sh.ContinuityNotes, localeBlock(loc))
		resp, n, err := invokeWithReview(ctx, llm, model, GenPackSystem, user, tokenBudget(2500), func(s string) (bool, string) {
			data, err := jsonout.ExtractObject(s)
			if err != nil {
				return false, err.Error()
			}
			if asString(data["video_prompt"]) == "" && asString(data["keyframe_prompt"]) == "" {
				return false, "keyframe_prompt or video_prompt required"
			}
			return true, ""
		}, adaptProgress(on, StageR6), "genpack_"+sh.ShotID)
		calls += n
		if err != nil {
			return calls, fmt.Errorf("genpack %s: %w", sh.ShotID, err)
		}
		data, err := jsonout.ExtractObject(resp)
		if err != nil {
			return calls, err
		}
		packs = append(packs, GenerationPackage{
			PackageID:      "GP_" + sh.ShotID,
			ShotID:         sh.ShotID,
			SceneID:        sh.SceneID,
			KeyframePrompt: asString(data["keyframe_prompt"]),
			VideoPrompt:    asString(data["video_prompt"]),
			NegativePrompt: asString(data["negative_prompt"]),
			StartVisual:    firstNonEmpty(asString(data["start_visual"]), asString(data["keyframe_prompt"])),
			EndVisual:      asString(data["end_visual"]),
			CharacterIDs:   resolveCharIDs(pipe, asStrings(data["character_names"])),
			LocationID:     resolveLocID(pipe, asString(data["location_name"])),
			PropIDs:        resolvePropIDs(pipe, asStrings(data["prop_names"])),
			ReviewItems:    asStrings(data["review_items"]),
			ReviewStatus:   ReviewPending,
		})
	}
	pipe.GenPackages = packs
	pipe.Workflow.Stage = StageR6
	pipe.Workflow.Status = "completed"
	return calls, nil
}

func RunAudit(ctx context.Context, llm port.LLMProvider, model string, pipe *ScriptPipeline, on StageProgressFn) (int, error) {
	emit(on, StageR7, "", "audit", 95)
	loc := NormalizeLocale(pipe.Locale)
	user := fmt.Sprintf("Pipeline ID inventory:\n%s\n\nArtifact digests:\n%s\n\nNumbered script (excerpt):\n%s\n%s",
		auditIDInventory(pipe), auditArtifactDigests(pipe), truncRunes(pipe.SourceDocument.NumberedText, 8000), localeBlock(loc))
	resp, n, err := invokeWithReview(ctx, llm, model, AuditSystem, user, tokenBudget(3000), func(s string) (bool, string) {
		_, err := jsonout.ExtractObject(s)
		if err != nil {
			return false, err.Error()
		}
		return true, ""
	}, adaptProgress(on, StageR7), "audit")
	if err != nil {
		return n, err
	}
	data, _ := jsonout.ExtractObject(resp)
	report := &AuditReport{
		ArtifactID:   "AUDIT_" + pipe.VersionID,
		ReviewStatus: ReviewPending,
	}
	for i, iss := range asMaps(data["issues"]) {
		report.Issues = append(report.Issues, AuditIssue{
			IssueID:           fmt.Sprintf("AUD_%03d", i+1),
			Severity:          firstNonEmpty(asString(iss["severity"]), "minor"),
			Action:            firstNonEmpty(asString(iss["action"]), ActionFillField),
			ArtifactID:        asString(iss["artifact_id"]),
			FieldPath:         asString(iss["field_path"]),
			Problem:           asString(iss["problem"]),
			EvidenceQuote:     asString(iss["evidence_quote"]),
			RepairInstruction: asString(iss["repair_instruction"]),
			Status:            "open",
		})
	}
	pipe.AuditReport = report
	pipe.Workflow.Stage = StageGate2
	pipe.Workflow.Status = "awaiting_review"
	pipe.Workflow.AwaitingReview = "shots"
	return n, nil
}

// --- helpers ---

// continuitySceneSummaries builds structured per-scene digests for R4 (enough for
// field_path + evidence_quote without dumping full breakdown JSON).
func continuitySceneSummaries(pipe *ScriptPipeline) string {
	var blocks []string
	for _, sc := range pipe.SceneIndex {
		art := pipe.SceneArtifacts[sc.SceneID]
		if art == nil || art.Breakdown == nil {
			continue
		}
		bd := art.Breakdown
		var elLines []string
		for i, el := range bd.Elements {
			quote := ""
			if el.Evidence != nil {
				quote = truncRunes(el.Evidence.Quote, 80)
			}
			states := ""
			if len(el.ContinuityStates) > 0 {
				states = " states=" + strings.Join(el.ContinuityStates, ",")
			}
			elLines = append(elLines, fmt.Sprintf("  [%d] %s/%s basis=%s%s quote=%q",
				i, el.Category, el.Name, el.Basis, states, quote))
		}
		if len(elLines) == 0 {
			elLines = append(elLines, "  (no elements)")
		}
		beatHint := ""
		if art.Analysis != nil && len(art.Analysis.Beats) > 0 {
			var beats []string
			for _, b := range art.Analysis.Beats {
				beats = append(beats, b.BeatID+":"+truncRunes(b.SourceEvent, 40))
			}
			beatHint = "\nbeats: " + strings.Join(beats, "; ")
		}
		blocks = append(blocks, fmt.Sprintf(
			"## %s %s\ncast_present=%v\nmentioned_not_present=%v\nelements:\n%s%s",
			sc.SceneID, firstNonEmpty(bd.Heading, sc.Heading),
			bd.CastPresent, bd.MentionedNotPresent,
			strings.Join(elLines, "\n"), beatHint,
		))
	}
	if len(blocks) == 0 {
		return "(no scene breakdowns)"
	}
	return strings.Join(blocks, "\n\n")
}

func auditIDInventory(pipe *ScriptPipeline) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("facts_approved=%v shots=%d packs=%d",
		pipe.Workflow.FactsApproved, len(pipe.ShotList), len(pipe.GenPackages)))
	for _, sc := range pipe.SceneIndex {
		lines = append(lines, fmt.Sprintf("scene %s heading=%s", sc.SceneID, sc.Heading))
		art := pipe.SceneArtifacts[sc.SceneID]
		if art != nil && art.Analysis != nil {
			for _, b := range art.Analysis.Beats {
				lines = append(lines, fmt.Sprintf("  beat %s", b.BeatID))
			}
		}
	}
	for _, sh := range pipe.ShotList {
		lines = append(lines, fmt.Sprintf("shot %s scene=%s beats=%v",
			sh.ShotID, sh.SceneID, sh.SourceBeatIDs))
	}
	for _, gp := range pipe.GenPackages {
		lines = append(lines, fmt.Sprintf("pack %s shot=%s", gp.PackageID, gp.ShotID))
	}
	return strings.Join(lines, "\n")
}

func auditArtifactDigests(pipe *ScriptPipeline) string {
	var blocks []string
	blocks = append(blocks, continuitySceneSummaries(pipe))
	if len(pipe.ShotList) > 0 {
		var shotLines []string
		for _, sh := range pipe.ShotList {
			shotLines = append(shotLines, fmt.Sprintf("%s [%s] size=%s move=%s action=%s cast=%v",
				sh.ShotID, sh.SceneID, sh.ShotSize, sh.CameraMovement,
				truncRunes(sh.VisualAction, 60), sh.CharactersOnScreen))
		}
		blocks = append(blocks, "## shots\n"+strings.Join(shotLines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

func adaptProgress(on StageProgressFn, stage string) ProgressFn {
	return func(phase, message string) {
		emit(on, stage, "", firstNonEmpty(message, phase), 0)
	}
}

func formatRegistryChars(reg *EntityRegistry) string {
	if reg == nil {
		return "(empty)"
	}
	var lines []string
	for _, c := range reg.Characters {
		parts := []string{fmt.Sprintf("- %s (%s)", c.Name, firstNonEmpty(c.Role, "supporting"))}
		if len(c.Aliases) > 0 {
			parts = append(parts, fmt.Sprintf("aliases=%v", c.Aliases))
		}
		if c.IdentityBaseline != "" {
			parts = append(parts, "id="+c.IdentityBaseline)
		}
		if c.VisualIdentity != "" {
			parts = append(parts, "face="+c.VisualIdentity)
		}
		if look := primaryLookBody(c); look != "" {
			parts = append(parts, "look="+look)
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	if len(lines) == 0 {
		return "(empty)"
	}
	return strings.Join(lines, "\n")
}

func numberedSlice(lines []string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	var out []string
	for i := start; i <= end && i <= len(lines); i++ {
		out = append(out, fmt.Sprintf("%d|%s", i, lines[i-1]))
	}
	return strings.Join(out, "\n")
}

// filterScenes picks scenes to (re)run. Unchanged scenes are skipped only when their
// artifact already has the required piece (breakdown / analysis), so failed mid-runs resume cleanly.
func filterScenes(index []SceneIndexEntry, only []string, changes []SceneChange, arts map[string]*SceneArtifact, need string) []SceneIndexEntry {
	skip := map[string]bool{}
	if len(only) == 0 && len(changes) > 0 {
		for _, ch := range changes {
			if ch.Status != "unchanged" {
				continue
			}
			art := arts[ch.SceneID]
			switch need {
			case "breakdown":
				if art != nil && art.Breakdown != nil {
					skip[ch.SceneID] = true
				}
			case "analysis":
				if art != nil && art.Analysis != nil {
					skip[ch.SceneID] = true
				}
			default:
				skip[ch.SceneID] = true
			}
		}
	}
	onlySet := map[string]bool{}
	for _, id := range only {
		onlySet[id] = true
	}
	var out []SceneIndexEntry
	for _, sc := range index {
		if len(onlySet) > 0 && !onlySet[sc.SceneID] {
			continue
		}
		if skip[sc.SceneID] {
			continue
		}
		out = append(out, sc)
	}
	if len(out) == 0 && len(changes) > 0 && len(onlySet) == 0 {
		return nil
	}
	if len(out) == 0 && len(onlySet) == 0 {
		return index
	}
	return out
}

func mergePropsFromBreakdown(pipe *ScriptPipeline, bd *SceneBreakdown) {
	if pipe.EntityRegistry == nil {
		pipe.EntityRegistry = &EntityRegistry{}
	}
	seen := map[string]bool{}
	for _, p := range pipe.EntityRegistry.Props {
		seen[p.Name] = true
	}
	for _, el := range bd.Elements {
		if el.Category != "prop" && el.Category != "set_dressing" {
			continue
		}
		if seen[el.Name] {
			for i := range pipe.EntityRegistry.Props {
				if pipe.EntityRegistry.Props[i].Name == el.Name {
					pipe.EntityRegistry.Props[i].SceneIDs = appendUnique(pipe.EntityRegistry.Props[i].SceneIDs, bd.SceneID)
				}
			}
			continue
		}
		seen[el.Name] = true
		pipe.EntityRegistry.Props = append(pipe.EntityRegistry.Props, RegistryProp{
			ElementID:    el.ElementID,
			Name:         el.Name,
			Category:     el.Category,
			Basis:        el.Basis,
			SceneIDs:     []string{bd.SceneID},
			ReviewStatus: ReviewPending,
		})
	}
}

func appendUnique(ss []string, v string) []string {
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}

func resolveCharIDs(pipe *ScriptPipeline, names []string) []string {
	m := map[string]string{}
	if pipe.EntityRegistry != nil {
		for _, c := range pipe.EntityRegistry.Characters {
			m[c.Name] = c.CharacterID
		}
	}
	var ids []string
	for _, n := range names {
		if id := m[n]; id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func resolveLocID(pipe *ScriptPipeline, name string) string {
	if name == "" || pipe.EntityRegistry == nil {
		return ""
	}
	for _, l := range pipe.EntityRegistry.Locations {
		if l.Name == name {
			return l.LocationID
		}
	}
	return ""
}

func resolvePropIDs(pipe *ScriptPipeline, names []string) []string {
	m := map[string]string{}
	if pipe.EntityRegistry != nil {
		for _, p := range pipe.EntityRegistry.Props {
			m[p.Name] = p.ElementID
		}
	}
	var ids []string
	for _, n := range names {
		if id := m[n]; id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
