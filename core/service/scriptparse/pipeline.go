package scriptparse

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/danmo-ai/danmo-film/core/domain/jsonout"
	"github.com/danmo-ai/danmo-film/core/port"
)

func RunDecompose(ctx context.Context, llm port.LLMProvider, model, script, title, locale string, onProgress ProgressFn) (map[string]any, int, error) {
	text := strings.TrimSpace(script)
	if len([]rune(text)) < 20 {
		return nil, 0, fmt.Errorf("script_text is too short")
	}
	loc := NormalizeLocale(locale)
	lb := localeBlock(loc)
	calls := 0
	chunks := splitChunks(text, 3500)
	var user string
	if len(chunks) == 1 {
		user = fmt.Sprintf("Title: %s\n\nScript:\n%s\n%s", orUntitled(title), text, lb)
	} else {
		var lines []string
		for _, chunk := range chunks {
			resp, n, err := invokeWithReview(ctx, llm, model, ChunkSystem, "Excerpt:\n"+chunk+"\n"+lb, tokenBudget(2000), func(s string) (bool, string) {
				data, err := jsonout.ExtractObject(s)
				if err != nil {
					return false, err.Error()
				}
				if len(asMaps(data["beats"])) == 0 {
					return false, "beats array empty"
				}
				return true, ""
			}, onProgress, "decompose_chunk")
			calls += n
			if err != nil {
				return nil, calls, err
			}
			data, _ := jsonout.ExtractObject(resp)
			for _, b := range asMaps(data["beats"]) {
				lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", asString(b["title"]), asString(b["shot_size"]), asString(b["location"]), asString(b["narrative"])))
			}
		}
		user = fmt.Sprintf("Title: %s\n\nMerged beat outline:\n%s\n\nFull script (reference):\n%s\n%s", orUntitled(title), strings.Join(lines, "\n"), truncRunes(text, 8000), lb)
	}
	if onProgress != nil {
		onProgress("decompose", "decompose")
	}
	resp, n, err := invokeWithReview(ctx, llm, model, DecomposeSystem, user, tokenBudget(4000), func(s string) (bool, string) {
		data, err := jsonout.ExtractObject(s)
		if err != nil {
			return false, err.Error()
		}
		return validateDecompose(data)
	}, onProgress, "decompose")
	calls += n
	if err != nil {
		return nil, calls, err
	}
	data, err := jsonout.ExtractObject(resp)
	if err != nil {
		return nil, calls, err
	}
	if ok, fb := validateDecompose(data); !ok {
		return nil, calls, fmt.Errorf("decompose validation failed: %s", fb)
	}
	if asString(data["title"]) == "" {
		data["title"] = title
	}
	data["version"] = "2.0"
	assignIDs(data)
	return data, calls, nil
}

func validateDecompose(data map[string]any) (bool, string) {
	beats := asMaps(data["beats"])
	chars := asMaps(data["characters"])
	scenes := asMaps(data["scenes"])
	var issues []string
	if len(beats) < 2 || len(beats) > 24 {
		issues = append(issues, fmt.Sprintf("beats count %d not in 2–24", len(beats)))
	}
	if len(chars) < 1 {
		issues = append(issues, "characters required")
	}
	if len(scenes) < 1 {
		issues = append(issues, "scenes required")
	}
	protag := 0
	for _, c := range chars {
		if asString(c["role"]) == "protagonist" {
			protag++
		}
		if len(asMaps(c["looks"])) < 1 {
			issues = append(issues, "character look missing")
		}
	}
	if protag != 1 {
		issues = append(issues, fmt.Sprintf("expected exactly 1 protagonist, got %d", protag))
	}
	for _, b := range beats {
		if asString(b["narrative"]) == "" {
			issues = append(issues, "empty beat narrative")
		}
		if asString(b["location"]) == "" {
			issues = append(issues, "empty beat location")
		}
		if len(asStrings(b["enhancement_cues"])) == 0 {
			issues = append(issues, fmt.Sprintf("beat %d: enhancement_cues required", asInt(b["index"])))
		}
	}
	if asString(data["synopsis"]) == "" || len([]rune(asString(data["synopsis"]))) < 8 {
		issues = append(issues, "synopsis too short")
	}
	if len(issues) > 0 {
		if len(issues) > 12 {
			issues = issues[:12]
		}
		return false, strings.Join(issues, "\n")
	}
	return true, ""
}

func RunExpand(ctx context.Context, llm port.LLMProvider, model string, artifact map[string]any, target, segDur, maxClip float64, locale string, beatIndices []int, onProgress ProgressFn) (shots []map[string]any, calls int, warnings []string, err error) {
	if target <= 0 {
		target = 60
	}
	if segDur <= 0 {
		segDur = 5
	}
	if maxClip <= 0 {
		maxClip = 10
	}
	loc := NormalizeLocale(locale)
	plan, n, err := runBeatPlan(ctx, llm, model, artifact, target, segDur, maxClip, loc, beatIndices, onProgress)
	calls += n
	if err != nil {
		return nil, calls, nil, err
	}
	specs, n, err := runShotSpec(ctx, llm, model, artifact, plan, loc, onProgress)
	calls += n
	if err != nil {
		return nil, calls, nil, err
	}
	shots = specsToShots(specs)
	simpleCastLock(shots, asMaps(artifact["characters"]), asMaps(artifact["scenes"]))
	if onProgress != nil {
		onProgress("finalize", "cast_lock")
	}
	return shots, calls, warnings, nil
}

func runBeatPlan(ctx context.Context, llm port.LLMProvider, model string, art map[string]any, target, segDur, maxClip float64, loc string, beatIndices []int, onProgress ProgressFn) ([]map[string]any, int, error) {
	if onProgress != nil {
		onProgress("beat_plan", "beat_plan")
	}
	user := beatPlanUser(art, target, segDur, loc, beatIndices)
	resp, n, err := invokeWithReview(ctx, llm, model, BeatPlanSystem, user, tokenBudget(4000), func(s string) (bool, string) {
		data, err := jsonout.ExtractObject(s)
		if err != nil {
			return false, err.Error()
		}
		if len(asMaps(data["beats"])) == 0 {
			return false, "beats empty"
		}
		return true, ""
	}, onProgress, "beat_plan")
	if err != nil {
		return nil, n, err
	}
	data, err := jsonout.ExtractObject(resp)
	if err != nil {
		return nil, n, err
	}
	return expandPlan(art, asMaps(data["beats"]), maxClip), n, nil
}

func beatPlanUser(art map[string]any, target, segDur float64, loc string, beatIndices []int) string {
	allowed := map[int]bool{}
	for _, i := range beatIndices {
		allowed[i] = true
	}
	var rows []string
	for _, beat := range asMaps(art["beats"]) {
		idx := asInt(beat["index"])
		if len(allowed) > 0 && !allowed[idx] {
			continue
		}
		rows = append(rows, fmt.Sprintf("[%d] title=%s\nlocation=%s\nshot_size=%s\nestimated_duration_sec=%v\nnarrative=%s\nenhancement_cues=%s",
			idx, asString(beat["title"]), asString(beat["location"]), asString(beat["suggested_shot_size"]),
			asFloat(beat["estimated_duration_sec"], 5), asString(beat["narrative"]), strings.Join(asStrings(beat["enhancement_cues"]), "; ")))
	}
	var cast []string
	for _, c := range asMaps(art["characters"]) {
		cast = append(cast, asString(c["name"]))
	}
	anchor := formatRoster(asMaps(art["characters"]), asString(art["style_anchor"]), loc)
	var scenes []string
	for _, s := range asMaps(art["scenes"]) {
		body := ""
		looks := asMaps(s["looks"])
		if len(looks) > 0 {
			body = asString(looks[0]["body"])
		}
		scenes = append(scenes, "- "+asString(s["name"])+": "+body)
	}
	softLo, softHi := softDurationBand(target)
	return fmt.Sprintf("Synopsis:\n%s\n\nMood: %s\nTarget total duration: %vs (soft band %g–%gs; prefer story completeness over exact sum)\nDefault segment duration hint: %vs (per-segment usually 4s–model max; time from dialogue/action, do not pad empty shots to fill the budget)\nCast: %s\n\nCharacter anchor:\n%s\n\nBeats:\n%s\n\nScenes:\n%s\n%s",
		asString(art["synopsis"]), asString(art["mood"]), target, softLo, softHi, segDur, strings.Join(cast, ", "), anchor, strings.Join(rows, "\n\n"), strings.Join(scenes, "\n"), localeBlock(loc))
}

// softDurationBand returns ±20% soft bounds around target (seconds).
func softDurationBand(target float64) (lo, hi float64) {
	if target <= 0 {
		target = 60
	}
	lo = math.Floor(target * 0.8)
	hi = math.Ceil(target * 1.2)
	if lo < 1 {
		lo = 1
	}
	if hi < lo {
		hi = lo
	}
	return lo, hi
}

func expandPlan(art map[string]any, llmRows []map[string]any, maxClip float64) []map[string]any {
	cap := maxClip
	if cap < 2 {
		cap = 10
	}
	var segs []map[string]any
	segIdx := 0
	prev := ""
	for _, row := range llmRows {
		bi := asInt(row["beat_index"])
		gid := fmt.Sprintf("beat_%d", bi)
		groupI := 0
		faceSeen := false
		for _, seg := range asMaps(row["segments"]) {
			role := sanitizeRole(asString(seg["role"]))
			if role == "face_anchor" {
				if faceSeen {
					role = "post_anchor"
				} else {
					faceSeen = true
				}
			}
			dur := asFloat(seg["duration_sec"], 5)
			if dur > cap {
				// split
			}
			parts := splitDurationParts(dur, cap)
			for pi, part := range parts {
				r := role
				if pi > 0 {
					r = "tail_continuation"
				}
				mode := startFrameMode(r, prev)
				on := asStrings(seg["characters_on_screen"])
				if r == "face_anchor" && len(on) > 1 {
					on = on[:1]
				}
				faceID := ""
				if r == "face_anchor" {
					faceID = stableID("anchor", fmt.Sprintf("%s|%d", gid, segIdx))
				}
				sv := sanitizeVis(asString(seg["start_visibility"]), "partial")
				if r == "face_anchor" {
					sv = "full_face"
				} else if sv == "full_face" {
					sv = "partial"
				}
				item := map[string]any{
					"segment_index": segIdx, "beat_index": bi, "role": r, "duration_sec": part,
					"shot_size": asString(seg["shot_size"]), "characters_on_screen": toStrAny(on),
					"start_visibility": sv, "end_visibility": sanitizeVis(asString(seg["end_visibility"]), "full_face"),
					"first_frame_requirement": asString(seg["first_frame_requirement"]),
					"reachability":            asString(seg["reachability"]), "is_intentional_empty": seg["is_intentional_empty"],
					"start_frame_mode": mode, "segment_group_id": gid, "segment_group_index": groupI,
					"face_anchor_shot_id": faceID, "shot_intent": asString(row["shot_intent"]),
					"location": beatLocation(art, bi),
				}
				segs = append(segs, item)
				prev = r
				segIdx++
				groupI++
			}
		}
	}
	return segs
}

func beatLocation(art map[string]any, idx int) string {
	for _, b := range asMaps(art["beats"]) {
		if asInt(b["index"]) == idx {
			return asString(b["location"])
		}
	}
	return ""
}

func toStrAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func runShotSpec(ctx context.Context, llm port.LLMProvider, model string, art map[string]any, segs []map[string]any, loc string, onProgress ProgressFn) ([]map[string]any, int, error) {
	if onProgress != nil {
		onProgress("shot_spec", "shot_spec")
	}
	calls := 0
	byIndex := map[int]map[string]any{}
	const batch = 6
	for i := 0; i < len(segs); i += batch {
		end := i + batch
		if end > len(segs) {
			end = len(segs)
		}
		part := segs[i:end]
		user := shotSpecUser(art, part, loc)
		resp, n, err := invokeWithReview(ctx, llm, model, ShotSpecSystem, user, tokenBudget(4000), func(s string) (bool, string) {
			data, err := jsonout.ExtractObject(s)
			if err != nil {
				return false, err.Error()
			}
			if len(asMaps(data["shots"])) == 0 {
				return false, "shots empty"
			}
			return true, ""
		}, onProgress, "shot_spec")
		calls += n
		if err != nil {
			return nil, calls, err
		}
		data, err := jsonout.ExtractObject(resp)
		if err != nil {
			return nil, calls, err
		}
		for _, row := range asMaps(data["shots"]) {
			byIndex[asInt(row["segment_index"])] = row
		}
	}
	var out []map[string]any
	for _, seg := range segs {
		idx := asInt(seg["segment_index"])
		row := byIndex[idx]
		if row == nil {
			return nil, calls, fmt.Errorf("shot_spec missing segment_index %d", idx)
		}
		out = append(out, mergeSpec(seg, row))
	}
	return out, calls, nil
}

func shotSpecUser(art map[string]any, segs []map[string]any, loc string) string {
	var body []string
	var idxs []string
	for _, s := range segs {
		idxs = append(idxs, fmt.Sprint(s["segment_index"]))
		body = append(body, fmt.Sprintf("[%v] beat=%v role=%v duration=%vs\nlocation=%s\nshot_size=%s\ncharacters_on_screen=%v\nstart_visibility=%s end_visibility=%s\nis_intentional_empty=%v\nfirst_frame_requirement=%s",
			s["segment_index"], s["beat_index"], s["role"], s["duration_sec"], s["location"], s["shot_size"], s["characters_on_screen"],
			s["start_visibility"], s["end_visibility"], s["is_intentional_empty"], s["first_frame_requirement"]))
	}
	return fmt.Sprintf("Synopsis: %s\nMood: %s\nStyle anchor (do NOT paste into video_prompt or start_visual): %s\n\nShort-drama prompts: start_visual = t=0 still (shot size + pose + environment); video_prompt = timed action + camera + speaker/reaction matching duration_sec. No cinematic filler.\n\nDerive shot specs for these segment_index values ONLY:\n\n%s\n\nRequired segment_index values: %s\n%s",
		asString(art["synopsis"]), asString(art["mood"]), asString(art["style_anchor"]), strings.Join(body, "\n\n"), strings.Join(idxs, ", "), localeBlock(loc))
}

func mergeSpec(seg, row map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range seg {
		out[k] = v
	}
	out["five_aspect"] = row["five_aspect"]
	out["shot_language"] = row["shot_language"]
	out["shot_intent"] = firstNonEmpty(asString(row["shot_intent"]), asString(seg["shot_intent"]))
	out["video_prompt"] = asString(row["video_prompt"])
	out["start_visual"] = asString(row["start_visual"])
	if asString(seg["role"]) == "face_anchor" {
		out["anchor_visual"] = asString(row["anchor_visual"])
	} else {
		out["anchor_visual"] = ""
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func specsToShots(specs []map[string]any) []map[string]any {
	var shots []map[string]any
	for i, spec := range specs {
		fa, _ := spec["five_aspect"].(map[string]any)
		sl, _ := spec["shot_language"].(map[string]any)
		if fa == nil {
			fa = map[string]any{}
		}
		if sl == nil {
			sl = map[string]any{}
		}
		role := asString(spec["role"])
		flf := "none"
		if role == "tail_continuation" {
			flf = "continuation"
		}
		shots = append(shots, map[string]any{
			"id":                      fmt.Sprintf("shot_%d", asInt(spec["segment_index"])),
			"order":                   i,
			"visual_prompt":           spec["start_visual"],
			"start_visual_prompt":     spec["start_visual"],
			"video_prompt":            spec["video_prompt"],
			"motion_prompt":           spec["video_prompt"],
			"anchor_visual_prompt":    spec["anchor_visual"],
			"scene_prompt":            fa["scene"],
			"segment_role":            role,
			"start_frame_mode":        spec["start_frame_mode"],
			"segment_group_id":        spec["segment_group_id"],
			"segment_group_index":     spec["segment_group_index"],
			"face_anchor_shot_id":     spec["face_anchor_shot_id"],
			"flf_mode":                flf,
			"duration_sec":            spec["duration_sec"],
			"first_frame_visibility":  spec["start_visibility"],
			"end_visibility":          spec["end_visibility"],
			"characters_on_screen":    spec["characters_on_screen"],
			"first_frame_requirement": spec["first_frame_requirement"],
			"location":                spec["location"],
			"shot_size":               sl["shot_size"],
			"narrative_beat_index":    spec["beat_index"],
			"shot_intent":             spec["shot_intent"],
			"camera_movement":         sl["camera_movement"],
			"lighting_key":            sl["lighting_key"],
			"is_establishing_empty":   spec["is_intentional_empty"],
			"clip_start_state":        spec["start_visual"],
			"clip_end_state":          fa["subject_motion"],
			"first_frame_strategy":    "t2i_from_grounding",
		})
	}
	return shots
}

func orUntitled(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(untitled)"
	}
	return s
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func DecomposeResponse(art map[string]any, phases []map[string]any, calls int, parseRunID, projectID, locale string) map[string]any {
	chars := characterDTOs(asMaps(art["characters"]))
	anchor := formatRoster(asMaps(art["characters"]), asString(art["style_anchor"]), NormalizeLocale(locale))
	beats := asMaps(art["beats"])
	return map[string]any{
		"chapter_title": asString(art["title"]), "synopsis": asString(art["synopsis"]), "mood": asString(art["mood"]),
		"style_anchor": asString(art["style_anchor"]), "character_anchor": anchor,
		"characters": chars, "scenes": sceneDTOs(asMaps(art["scenes"])),
		"scene_beats": sceneBeats(art), "scene_count": len(beats),
		"script_artifact": art, "llm_calls": calls, "parse_phases": phases,
		"parse_run_id": parseRunID, "long_video_project_id": projectID,
	}
}

func ExpandResponse(art map[string]any, shots []map[string]any, phases []map[string]any, calls int, parseRunID, projectID string, warnings []string) map[string]any {
	return map[string]any{
		"chapter_title": asString(art["title"]), "synopsis": asString(art["synopsis"]), "mood": asString(art["mood"]),
		"style_anchor":     asString(art["style_anchor"]),
		"character_anchor": formatRoster(asMaps(art["characters"]), asString(art["style_anchor"]), "zh"),
		"characters":       characterDTOs(asMaps(art["characters"])), "scenes": sceneDTOs(asMaps(art["scenes"])),
		"scene_beats": sceneBeats(art), "scene_count": len(asMaps(art["beats"])),
		"shots": shots, "script_artifact": art, "llm_calls": calls, "parse_phases": phases,
		"quality_warnings": warnings, "quality_issues": []any{},
		"parse_run_id": parseRunID, "long_video_project_id": projectID,
	}
}

func characterDTOs(chars []map[string]any) []map[string]any {
	var out []map[string]any
	for _, c := range chars {
		var looks []any
		for _, lk := range asMaps(c["looks"]) {
			looks = append(looks, map[string]any{"id": lk["id"], "label": lk["label"], "body": lk["body"]})
		}
		row := map[string]any{
			"id": c["id"], "name": c["name"], "looks": looks, "default_look_id": c["default_look_id"],
		}
		for _, k := range []string{
			"role", "aliases", "age_range", "gender_presentation",
			"identity_baseline", "visual_identity", "voice_manner", "performance_notes", "identity_notes",
		} {
			if v, ok := c[k]; ok && v != nil {
				row[k] = v
			}
		}
		out = append(out, row)
	}
	return out
}

func sceneDTOs(scenes []map[string]any) []map[string]any {
	var out []map[string]any
	for _, s := range scenes {
		var looks []any
		for _, lk := range asMaps(s["looks"]) {
			looks = append(looks, map[string]any{"id": lk["id"], "label": lk["label"], "body": lk["body"]})
		}
		layout := s["spatial_layout_json"]
		if layout == nil {
			layout = map[string]any{}
		}
		out = append(out, map[string]any{
			"id": s["id"], "name": s["name"], "looks": looks, "default_look_id": s["default_look_id"],
			"spatial_layout_json":         layout,
			"grounding_panorama_asset_id": asString(s["grounding_panorama_asset_id"]),
			"grounding_depth_asset_id":    asString(s["grounding_depth_asset_id"]),
		})
	}
	return out
}
