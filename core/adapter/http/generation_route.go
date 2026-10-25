package httpapi

import (
	"context"
	"fmt"
	"strings"

	genadapt "github.com/danmo-ai/danmo-film/core/adapter/gen"
	"github.com/danmo-ai/danmo-film/core/port"
)

// generationForBody picks the vendor client for body.model (profile_id/model_id)
// and rewrites model to the bare upstream id. Staging must use this same client.
// action is "image" | "video" | "video_edit" so image_profiles vs generation_profiles resolve correctly.
func (a *App) generationForBody(body map[string]any, action string) (port.GenerationProvider, map[string]any, error) {
	if body == nil {
		body = map[string]any{}
	}
	ref := strings.TrimSpace(fmt.Sprint(body["model"]))
	root := a.generation()
	if r, ok := root.(*genadapt.Router); ok {
		var client port.GenerationProvider
		var bare, profileID string
		var err error
		if action == "image" {
			client, bare, profileID, err = r.ResolveImageModel(ref)
		} else {
			client, bare, profileID, err = r.ResolveVideoModel(ref)
		}
		if err != nil {
			return nil, body, err
		}
		cp := map[string]any{}
		for k, v := range body {
			cp[k] = v
		}
		cp["model"] = bare
		return &taskPrefixed{inner: client, profileID: profileID}, cp, nil
	}
	return root, body, nil
}

// taskPrefixed wraps a vendor client so Create* responses get profile_id:task_id
// (matches Router.dispatch), so GetTask / SSE can route.
type taskPrefixed struct {
	inner     port.GenerationProvider
	profileID string
}

func (t *taskPrefixed) ID() string {
	if t.profileID != "" {
		return t.profileID
	}
	return t.inner.ID()
}

func (t *taskPrefixed) Health(ctx context.Context) (map[string]any, error) {
	return t.inner.Health(ctx)
}

func (t *taskPrefixed) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	return t.inner.ListModels(ctx, media)
}

func (t *taskPrefixed) GetRegistry(ctx context.Context) (map[string]any, error) {
	return t.inner.GetRegistry(ctx)
}

func (t *taskPrefixed) ListAdapters(ctx context.Context, forModel string) (any, error) {
	return t.inner.ListAdapters(ctx, forModel)
}

func (t *taskPrefixed) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	out, err := t.inner.CreateImage(ctx, body)
	if err != nil {
		return nil, err
	}
	return prefixGenTaskID(out, t.ID()), nil
}

func (t *taskPrefixed) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	out, err := t.inner.CreateVideo(ctx, body)
	if err != nil {
		return nil, err
	}
	return prefixGenTaskID(out, t.ID()), nil
}

func (t *taskPrefixed) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	out, err := t.inner.EditVideo(ctx, body)
	if err != nil {
		return nil, err
	}
	return prefixGenTaskID(out, t.ID()), nil
}

func (t *taskPrefixed) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	return t.inner.GetTask(ctx, taskID)
}

func (t *taskPrefixed) DownloadResult(ctx context.Context, providerRef, dest string) error {
	return t.inner.DownloadResult(ctx, providerRef, dest)
}

func (t *taskPrefixed) StageInput(ctx context.Context, path, mime string) (map[string]any, error) {
	return t.inner.StageInput(ctx, path, mime)
}

func (t *taskPrefixed) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	return t.inner.IterTaskSSE(ctx, taskID)
}

func prefixGenTaskID(out map[string]any, providerID string) map[string]any {
	if out == nil || strings.TrimSpace(providerID) == "" {
		return out
	}
	native := extractGenTaskID(out)
	if native == "" || strings.Contains(native, ":") {
		return out
	}
	prefixed := providerID + ":" + native
	cp := make(map[string]any, len(out)+1)
	for k, v := range out {
		cp[k] = v
	}
	cp["id"] = prefixed
	// Make-style bodies nest the id under task — keep both in sync so clients
	// that prefer task.id still get a routable profile_id:native_id.
	if task, ok := cp["task"].(map[string]any); ok {
		t2 := make(map[string]any, len(task)+1)
		for k, v := range task {
			t2[k] = v
		}
		t2["id"] = prefixed
		cp["task"] = t2
	}
	return cp
}

func extractGenTaskID(out map[string]any) string {
	if out == nil {
		return ""
	}
	if id := strings.TrimSpace(fmt.Sprint(out["id"])); id != "" && id != "<nil>" {
		return id
	}
	if task, ok := out["task"].(map[string]any); ok {
		if id := strings.TrimSpace(fmt.Sprint(task["id"])); id != "" && id != "<nil>" {
			return id
		}
	}
	if id := strings.TrimSpace(fmt.Sprint(out["task_id"])); id != "" && id != "<nil>" {
		return id
	}
	return ""
}
