package sqlite

import (
	"fmt"
	"math"
)

// RevisionConflictError is returned when optimistic revision does not match stored state.
type RevisionConflictError struct {
	ServerRevision int
	ClientRevision int
	ResourceKind   string
	ResourceID     string
}

func (e *RevisionConflictError) Error() string {
	return fmt.Sprintf("%s revision conflict for %s: server has %d, client sent %d",
		e.ResourceKind, e.ResourceID, e.ServerRevision, e.ClientRevision)
}

// stateRevision reads only the optimistic-lock `revision` field.
// Never use schema `version` (project/series document version 1|2) — that caused
// every second save to 409 after bumpRevision corrupted version.
func stateRevision(state map[string]any) int {
	if state == nil {
		return 0
	}
	if v, ok := state["revision"]; ok && v != nil {
		return intNumber(v, 0)
	}
	return 0
}

func intNumber(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return def
		}
		return int(t)
	case float32:
		return int(t)
	default:
		return def
	}
}

// clientSentRevision is true only when the client explicitly sent `revision`.
// Schema `version` must not participate in optimistic locking.
func clientSentRevision(state map[string]any) (int, bool) {
	if state == nil {
		return 0, false
	}
	if v, ok := state["revision"]; ok && v != nil {
		return intNumber(v, 0), true
	}
	return 0, false
}

func bumpRevision(state map[string]any) {
	if state == nil {
		return
	}
	state["revision"] = stateRevision(state) + 1
}
