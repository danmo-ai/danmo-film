package httpapi

import "testing"

func TestPrefixGenTaskIDFromNestedMakeBody(t *testing.T) {
	out := map[string]any{
		"task": map[string]any{
			"id":     "tsk_abc",
			"status": "queued",
		},
	}
	got := prefixGenTaskID(out, "danmo-make-local")
	if got["id"] != "danmo-make-local:tsk_abc" {
		t.Fatalf("top id=%v", got["id"])
	}
	task, _ := got["task"].(map[string]any)
	if task["id"] != "danmo-make-local:tsk_abc" {
		t.Fatalf("task.id=%v", task["id"])
	}
}

func TestPrefixGenTaskIDIgnoresNilTopID(t *testing.T) {
	out := map[string]any{
		"id": nil,
		"task": map[string]any{
			"id": "tsk_xyz",
		},
	}
	got := prefixGenTaskID(out, "p1")
	if got["id"] != "p1:tsk_xyz" {
		t.Fatalf("id=%v", got["id"])
	}
}
