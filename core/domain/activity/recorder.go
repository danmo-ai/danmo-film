package activity

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

type Recorder struct {
	Store      port.ActivityStore
	ProjectID  string
	ParseRunID string
}

func NewRecorder(store port.ActivityStore, projectID string) *Recorder {
	pid := strings.TrimSpace(projectID)
	r := &Recorder{Store: store, ProjectID: pid}
	if pid != "" {
		b := make([]byte, 12)
		_, _ = rand.Read(b)
		r.ParseRunID = "prun_" + hex.EncodeToString(b)
	}
	return r
}

func (r *Recorder) Active() bool { return r != nil && r.ProjectID != "" && r.ParseRunID != "" }

func (r *Recorder) Started(title string, duration float64) {
	if !r.Active() {
		return
	}
	sum := "Script parse started"
	if title != "" {
		sum += ": " + title
	}
	_, _ = r.Store.Append(port.ActivityEvent{
		ProjectID: r.ProjectID, Category: "script_parse", EventType: "parse_started",
		Phase: "script_parse", ParseRunID: r.ParseRunID, Summary: sum,
		Detail: map[string]any{"chapter_title": title, "target_duration_sec": duration},
	})
}

func (r *Recorder) Phase(phase, message string) {
	if !r.Active() {
		return
	}
	_, _ = r.Store.Append(port.ActivityEvent{
		ProjectID: r.ProjectID, Category: "script_parse", EventType: "parse_phase",
		Phase: phase, ParseRunID: r.ParseRunID, Summary: "Parse phase: " + phase,
		Detail: map[string]any{"message": message},
	})
}

func (r *Recorder) Completed(resp map[string]any) {
	if !r.Active() {
		return
	}
	shots, _ := resp["shots"].([]any)
	_, _ = r.Store.Append(port.ActivityEvent{
		ProjectID: r.ProjectID, Category: "script_parse", EventType: "parse_completed",
		Phase: "script_parse", Status: "completed", ParseRunID: r.ParseRunID,
		Summary: fmt.Sprintf("Script parse completed · %d shot(s)", len(shots)),
		Detail:  map[string]any{"chapter_title": resp["chapter_title"], "scene_count": resp["scene_count"], "shot_count": len(shots), "llm_calls": resp["llm_calls"]},
	})
}

func (r *Recorder) Failed(err string) {
	if !r.Active() {
		return
	}
	if len(err) > 2000 {
		err = err[:2000]
	}
	_, _ = r.Store.Append(port.ActivityEvent{
		ProjectID: r.ProjectID, Category: "script_parse", EventType: "parse_failed",
		Phase: "script_parse", Status: "failed", ParseRunID: r.ParseRunID,
		Summary: "Script parse failed", Detail: map[string]any{"error": err},
	})
}
