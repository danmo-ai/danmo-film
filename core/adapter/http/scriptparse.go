package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/domain/activity"
	"github.com/danmo-ai/danmo-film/core/service/scriptparse"
)

func (a *App) registerScriptParse(r *gin.Engine) {
	r.POST("/api/script-parse/decompose", a.decompose)
	r.POST("/api/script-parse/decompose/stream", a.decomposeStream)
	r.POST("/api/script-parse/expand", a.expand)
	r.POST("/api/script-parse/expand/stream", a.expandStream)
	r.POST("/api/script-parse/expand/beat", a.expandBeat)
	r.POST("/api/script-parse/expand/beat/stream", a.expandBeatStream)

	// Multi-stage pipeline (R0–R7)
	r.POST("/api/script-parse/versions", a.scriptParseVersion)
	r.POST("/api/script-parse/process", a.scriptParseProcess)
	r.POST("/api/script-parse/process/stream", a.scriptParseProcessStream)
	r.POST("/api/script-parse/rerun/:stage", a.scriptParseRerun)
	r.POST("/api/script-parse/approve/facts", a.scriptParseApproveFacts)
	r.POST("/api/script-parse/approve/shots", a.scriptParseApproveShots)
	r.POST("/api/script-parse/repair-preview", a.scriptParseRepairPreview)
	r.POST("/api/script-parse/apply-patches", a.scriptParseApplyPatches)
}

func (a *App) requireLLM(c *gin.Context) bool {
	llm := a.llm()
	if llm == nil || !llm.Available() {
		c.JSON(503, gin.H{"detail": "LLM provider not configured. Set FILM_LLM_BASE_URL / FILM_LLM_MODEL (or ~/.danmo-film/config/settings.json)."})
		return false
	}
	return true
}

type decomposeBody struct {
	ScriptText         string `json:"script_text"`
	Title              string `json:"title"`
	Locale             string `json:"locale"`
	LongVideoProjectID string `json:"long_video_project_id"`
	Model              string `json:"model"`
}

type expandBody struct {
	ScriptArtifact     map[string]any   `json:"script_artifact"`
	Locale             string           `json:"locale"`
	TargetDurationSec  float64          `json:"target_duration_sec"`
	SegmentDurationSec float64          `json:"segment_duration_sec"`
	MaxClipSec         float64          `json:"max_clip_sec"`
	LongVideoProjectID string           `json:"long_video_project_id"`
	Model              string           `json:"model"`
	BeatIndices        []int            `json:"beat_indices"`
	BeatIndex          int              `json:"beat_index"`
	ExistingShots      []map[string]any `json:"existing_shots"`
}

func (a *App) decompose(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body decomposeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	rec := activity.NewRecorder(a.Activity, body.LongVideoProjectID)
	if rec.Active() {
		rec.Started(body.Title, 0)
	}
	var phases []map[string]any
	on := func(phase, msg string) {
		phases = append(phases, map[string]any{"phase": phase, "message": msg})
		if rec.Active() {
			rec.Phase(phase, msg)
		}
	}
	art, calls, err := scriptparse.RunDecompose(c.Request.Context(), a.llm(), body.Model, body.ScriptText, body.Title, localeFrom(c, body.Locale), on)
	if err != nil {
		if rec.Active() {
			rec.Failed(err.Error())
		}
		fail(c, httpStatus(err), err)
		return
	}
	resp := scriptparse.DecomposeResponse(art, phases, calls, rec.ParseRunID, body.LongVideoProjectID, localeFrom(c, body.Locale))
	if rec.Active() {
		rec.Completed(resp)
	}
	c.JSON(http.StatusOK, resp)
}

func (a *App) expand(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body expandBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	a.runExpand(c, body, false)
}

func (a *App) expandBeat(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body expandBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	a.runExpand(c, body, true)
}

func (a *App) runExpand(c *gin.Context, body expandBody, oneBeat bool) {
	rec := activity.NewRecorder(a.Activity, body.LongVideoProjectID)
	if rec.Active() {
		rec.Started("", body.TargetDurationSec)
	}
	var phases []map[string]any
	on := func(phase, msg string) {
		phases = append(phases, map[string]any{"phase": phase, "message": msg})
		if rec.Active() {
			rec.Phase(phase, msg)
		}
	}
	indices := body.BeatIndices
	if oneBeat {
		indices = []int{body.BeatIndex}
	}
	shots, calls, warnings, err := scriptparse.RunExpand(c.Request.Context(), a.llm(), body.Model, body.ScriptArtifact, body.TargetDurationSec, body.SegmentDurationSec, body.MaxClipSec, localeFrom(c, body.Locale), indices, on)
	if err != nil {
		if rec.Active() {
			rec.Failed(err.Error())
		}
		fail(c, httpStatus(err), err)
		return
	}
	if oneBeat && len(body.ExistingShots) > 0 {
		shots = scriptparse.MergeBeatShots(body.ExistingShots, shots, body.BeatIndex)
	}
	on("done", "done")
	resp := scriptparse.ExpandResponse(body.ScriptArtifact, shots, phases, calls, rec.ParseRunID, body.LongVideoProjectID, warnings)
	if rec.Active() {
		rec.Completed(resp)
	}
	c.JSON(http.StatusOK, resp)
}

func (a *App) decomposeStream(c *gin.Context) {
	a.streamParse(c, true, false)
}

func (a *App) expandStream(c *gin.Context) {
	a.streamParse(c, false, false)
}

func (a *App) expandBeatStream(c *gin.Context) {
	a.streamParse(c, false, true)
}

func (a *App) streamParse(c *gin.Context, isDecompose, oneBeat bool) {
	if !a.requireLLM(c) {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	write := func(event string, v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b)
		c.Writer.Flush()
	}
	on := func(phase, msg string) {
		write("progress", map[string]any{"phase": phase, "message": msg})
	}
	if isDecompose {
		var body decomposeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			write("error", map[string]any{"error": err.Error()})
			return
		}
		rec := activity.NewRecorder(a.Activity, body.LongVideoProjectID)
		if rec.Active() {
			rec.Started(body.Title, 0)
		}
		var phases []map[string]any
		wrapped := func(phase, msg string) {
			phases = append(phases, map[string]any{"phase": phase, "message": msg})
			if rec.Active() {
				rec.Phase(phase, msg)
			}
			on(phase, msg)
		}
		art, calls, err := scriptparse.RunDecompose(c.Request.Context(), a.llm(), body.Model, body.ScriptText, body.Title, localeFrom(c, body.Locale), wrapped)
		if err != nil {
			if rec.Active() {
				rec.Failed(err.Error())
			}
			write("error", map[string]any{"error": err.Error()})
			return
		}
		resp := scriptparse.DecomposeResponse(art, phases, calls, rec.ParseRunID, body.LongVideoProjectID, localeFrom(c, body.Locale))
		if rec.Active() {
			rec.Completed(resp)
		}
		write("result", resp)
		return
	}
	var body expandBody
	if err := c.ShouldBindJSON(&body); err != nil {
		write("error", map[string]any{"error": err.Error()})
		return
	}
	rec := activity.NewRecorder(a.Activity, body.LongVideoProjectID)
	if rec.Active() {
		rec.Started("", body.TargetDurationSec)
	}
	var phases []map[string]any
	wrapped := func(phase, msg string) {
		phases = append(phases, map[string]any{"phase": phase, "message": msg})
		if rec.Active() {
			rec.Phase(phase, msg)
		}
		on(phase, msg)
	}
	indices := body.BeatIndices
	if oneBeat {
		indices = []int{body.BeatIndex}
	}
	shots, calls, warnings, err := scriptparse.RunExpand(c.Request.Context(), a.llm(), body.Model, body.ScriptArtifact, body.TargetDurationSec, body.SegmentDurationSec, body.MaxClipSec, localeFrom(c, body.Locale), indices, wrapped)
	if err != nil {
		if rec.Active() {
			rec.Failed(err.Error())
		}
		write("error", map[string]any{"error": err.Error()})
		return
	}
	if oneBeat && len(body.ExistingShots) > 0 {
		shots = scriptparse.MergeBeatShots(body.ExistingShots, shots, body.BeatIndex)
	}
	wrapped("done", "done")
	resp := scriptparse.ExpandResponse(body.ScriptArtifact, shots, phases, calls, rec.ParseRunID, body.LongVideoProjectID, warnings)
	if rec.Active() {
		rec.Completed(resp)
	}
	write("result", resp)
}

func httpStatus(err error) int {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not configured") || strings.Contains(msg, "not installed") {
		return 503
	}
	if strings.Contains(msg, "too short") || strings.Contains(msg, "invalid") || strings.Contains(msg, "required") || strings.Contains(msg, "validation") || strings.Contains(msg, "not approved") || strings.Contains(msg, "gate") {
		return 400
	}
	return 500
}

type processBody struct {
	ScriptText         string                      `json:"script_text"`
	Title              string                      `json:"title"`
	Locale             string                      `json:"locale"`
	Model              string                      `json:"model"`
	LongVideoProjectID string                      `json:"long_video_project_id"`
	Existing           *scriptparse.ScriptPipeline `json:"existing_pipeline"`
	StopAt             string                      `json:"stop_at"`
	UntilShots         bool                        `json:"until_shots"`
	TargetDurationSec  float64                     `json:"target_duration_sec"`
	SegmentDurationSec float64                     `json:"segment_duration_sec"`
	MaxClipSec         float64                     `json:"max_clip_sec"`
	OnlySceneIDs       []string                    `json:"only_scene_ids"`
	RerunStage         string                      `json:"rerun_stage"`
}

type approveBody struct {
	Pipeline *scriptparse.ScriptPipeline `json:"pipeline"`
}

type repairPreviewBody struct {
	Pipeline   *scriptparse.ScriptPipeline `json:"pipeline"`
	FindingIDs []string                    `json:"finding_ids"`
	UserNote   string                      `json:"user_note"`
	Model      string                      `json:"model"`
}

func (a *App) scriptParseVersion(c *gin.Context) {
	var body processBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	pipe, err := scriptparse.Preprocess(body.ScriptText, body.Title)
	if err != nil {
		fail(c, httpStatus(err), err)
		return
	}
	pipe.Locale = scriptparse.NormalizeLocale(localeFrom(c, body.Locale))
	if body.Existing != nil {
		pipe.SceneChangeSet = scriptparse.DiffSceneIndex(body.Existing.SceneIndex, pipe.SceneIndex)
	}
	c.JSON(http.StatusOK, gin.H{"script_pipeline": pipe})
}

func (a *App) scriptParseProcess(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body processBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	pipe, err := a.runProcess(c, body, nil)
	if err != nil {
		fail(c, httpStatus(err), err)
		return
	}
	c.JSON(http.StatusOK, a.pipelineResponse(pipe, body.LongVideoProjectID))
}

func (a *App) scriptParseProcessStream(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body processBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	write := func(event string, v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b)
		c.Writer.Flush()
	}
	on := func(ev scriptparse.ProgressEvent) {
		if ev.Checkpoint && ev.Pipeline != nil {
			write("checkpoint", a.pipelineResponse(ev.Pipeline, body.LongVideoProjectID))
			return
		}
		write("progress", map[string]any{
			"stage": ev.Stage, "scene_id": ev.SceneID, "message": ev.Message, "pct": ev.Pct,
		})
	}
	pipe, err := a.runProcess(c, body, on)
	if err != nil {
		payload := map[string]any{"error": err.Error(), "stage": pipeStage(pipe), "repairable": true}
		if pipe != nil {
			for k, v := range a.pipelineResponse(pipe, body.LongVideoProjectID) {
				payload[k] = v
			}
		}
		write("error", payload)
		return
	}
	write("result", a.pipelineResponse(pipe, body.LongVideoProjectID))
}

func (a *App) scriptParseRerun(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body processBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	body.RerunStage = c.Param("stage")
	if body.Existing == nil {
		fail(c, 400, fmt.Errorf("existing_pipeline required"))
		return
	}
	pipe, err := a.runProcess(c, body, nil)
	if err != nil {
		fail(c, httpStatus(err), err)
		return
	}
	c.JSON(http.StatusOK, a.pipelineResponse(pipe, body.LongVideoProjectID))
}

func (a *App) scriptParseApproveFacts(c *gin.Context) {
	var body approveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	if body.Pipeline == nil {
		fail(c, 400, fmt.Errorf("pipeline required"))
		return
	}
	if err := scriptparse.ApproveFacts(body.Pipeline); err != nil {
		fail(c, 400, err)
		return
	}
	scriptparse.LintPipeline(body.Pipeline)
	chars, scenes := scriptparse.RosterFromPipeline(body.Pipeline)
	c.JSON(http.StatusOK, gin.H{
		"script_pipeline": body.Pipeline,
		"characters":      chars,
		"scenes":          scenes,
		"scene_beats":     scriptparse.SceneBeatsFromPipeline(body.Pipeline),
	})
}

func (a *App) scriptParseApproveShots(c *gin.Context) {
	var body approveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	if body.Pipeline == nil {
		fail(c, 400, fmt.Errorf("pipeline required"))
		return
	}
	if err := scriptparse.ApproveShots(body.Pipeline); err != nil {
		fail(c, 400, err)
		return
	}
	scriptparse.LintPipeline(body.Pipeline)
	c.JSON(http.StatusOK, gin.H{
		"script_pipeline": body.Pipeline,
		"shots":           scriptparse.ShotsFromPipeline(body.Pipeline),
	})
}

func (a *App) scriptParseRepairPreview(c *gin.Context) {
	if !a.requireLLM(c) {
		return
	}
	var body repairPreviewBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	if body.Pipeline == nil {
		fail(c, 400, fmt.Errorf("pipeline required"))
		return
	}
	res, err := scriptparse.RepairPreview(c.Request.Context(), a.llm(), scriptparse.RepairPreviewOpts{
		Pipeline:   body.Pipeline,
		FindingIDs: body.FindingIDs,
		UserNote:   body.UserNote,
		Model:      body.Model,
	})
	if err != nil {
		fail(c, httpStatus(err), err)
		return
	}
	chars, scenes := scriptparse.RosterFromPipeline(res.ScriptPipeline)
	c.JSON(http.StatusOK, gin.H{
		"patches":         res.Patches,
		"script_pipeline": res.ScriptPipeline,
		"llm_calls":       res.LLMCalls,
		"characters":      chars,
		"scenes":          scenes,
		"scene_beats":     scriptparse.SceneBeatsFromPipeline(res.ScriptPipeline),
	})
}

func (a *App) scriptParseApplyPatches(c *gin.Context) {
	var body approveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	pipe, err := scriptparse.ApplyRepairPreview(body.Pipeline)
	if err != nil {
		fail(c, 400, err)
		return
	}
	chars, scenes := scriptparse.RosterFromPipeline(pipe)
	c.JSON(http.StatusOK, gin.H{
		"script_pipeline": pipe,
		"characters":      chars,
		"scenes":          scenes,
		"scene_beats":     scriptparse.SceneBeatsFromPipeline(pipe),
		"shots":           scriptparse.ShotsFromPipeline(pipe),
	})
}

func (a *App) runProcess(c *gin.Context, body processBody, on scriptparse.StageProgressFn) (*scriptparse.ScriptPipeline, error) {
	rec := activity.NewRecorder(a.Activity, body.LongVideoProjectID)
	if rec.Active() {
		rec.Started(body.Title, body.TargetDurationSec)
	}
	wrapped := on
	if rec.Active() {
		wrapped = func(ev scriptparse.ProgressEvent) {
			rec.Phase(ev.Stage, ev.Message)
			if on != nil {
				on(ev)
			}
		}
	}
	opts := scriptparse.ProcessOpts{
		ScriptText:         body.ScriptText,
		Title:              body.Title,
		Locale:             localeFrom(c, body.Locale),
		Model:              body.Model,
		Existing:           body.Existing,
		StopAt:             body.StopAt,
		UntilShots:         body.UntilShots,
		TargetDurationSec:  body.TargetDurationSec,
		SegmentDurationSec: body.SegmentDurationSec,
		MaxClipSec:         body.MaxClipSec,
		OnlySceneIDs:       body.OnlySceneIDs,
		RerunStage:         body.RerunStage,
	}
	pipe, err := scriptparse.RunProcess(c.Request.Context(), a.llm(), opts, wrapped)
	if err != nil {
		if rec.Active() {
			rec.Failed(err.Error())
		}
		return pipe, err
	}
	if pipe != nil {
		pipe.ParseRunID = rec.ParseRunID
	}
	if rec.Active() {
		rec.Completed(a.pipelineResponse(pipe, body.LongVideoProjectID))
	}
	return pipe, nil
}

func (a *App) pipelineResponse(pipe *scriptparse.ScriptPipeline, projectID string) map[string]any {
	chars, scenes := scriptparse.RosterFromPipeline(pipe)
	resp := map[string]any{
		"script_pipeline": pipe,
		"characters":      chars,
		"scenes":          scenes,
		"scene_beats":     scriptparse.SceneBeatsFromPipeline(pipe),
		"llm_calls":       pipe.LLMCalls,
		"parse_run_id":    pipe.ParseRunID,
		"project_id":      projectID,
		"synopsis":        "",
		"mood":            "",
		"style_anchor":    "",
	}
	if pipe.StoryMap != nil {
		resp["synopsis"] = pipe.StoryMap.Logline
		resp["mood"] = pipe.StoryMap.Mood
		resp["style_anchor"] = pipe.StoryMap.StyleAnchor
	}
	if pipe.Workflow.ShotsApproved || len(pipe.GenPackages) > 0 {
		resp["shots"] = scriptparse.ShotsFromPipeline(pipe)
	}
	return resp
}

func pipeStage(pipe *scriptparse.ScriptPipeline) string {
	if pipe == nil {
		return ""
	}
	return pipe.Workflow.Stage
}
