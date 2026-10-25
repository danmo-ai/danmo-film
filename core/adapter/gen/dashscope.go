package gen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

// DashScope 通义万相 video-synthesis (wan2.x) — one official async API.
type DashScopeProvider struct {
	base   string
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	staged map[string]stagedFile
	urls   map[string]string
}

func NewDashScope(base, apiKey, model string) *DashScopeProvider {
	if base == "" {
		base = "https://dashscope.aliyuncs.com"
	}
	if model == "" {
		model = "wan2.2-t2v-plus"
	}
	return &DashScopeProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
		staged: map[string]stagedFile{},
		urls:   map[string]string{},
	}
}

func (p *DashScopeProvider) ID() string { return "dashscope" }

func (p *DashScopeProvider) Health(_ context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "missing DashScope API key"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "dashscope-wan"}, nil
}

func (p *DashScopeProvider) ListModels(_ context.Context, media string) ([]map[string]any, error) {
	models := []map[string]any{
		{"id": p.model, "media": "video", "label": "万相视频", "modes": []string{"t2v", "i2v"}, "max_duration_sec": 10},
		{"id": "wanx-v1", "media": "image", "label": "万相文生图", "modes": []string{"t2i"}},
	}
	if media == "" {
		return models, nil
	}
	var out []map[string]any
	for _, m := range models {
		if strVal(m["media"]) == media {
			out = append(out, m)
		}
	}
	return out, nil
}

func (p *DashScopeProvider) GetRegistry(ctx context.Context) (map[string]any, error) {
	items, _ := p.ListModels(ctx, "")
	return map[string]any{"models": items}, nil
}

func (p *DashScopeProvider) ListAdapters(_ context.Context, _ string) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (p *DashScopeProvider) StageInput(_ context.Context, path, mime string) (map[string]any, error) {
	uri, err := fileDataURI(path, mime)
	if err != nil {
		return nil, err
	}
	id := "stg_" + filepathBase(path)
	p.mu.Lock()
	p.staged[id] = stagedFile{Path: path, MIME: mime, URI: uri}
	p.mu.Unlock()
	return map[string]any{"id": id}, nil
}

func (p *DashScopeProvider) resolveStage(id string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.staged[id]; ok {
		return s.URI
	}
	return id
}

func (p *DashScopeProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("dashscope API key not configured")
	}
	model := first(strVal(body["model"]), "wanx-v1")
	payload := map[string]any{
		"model": model,
		"input": map[string]any{"prompt": strVal(body["prompt"])},
		"parameters": map[string]any{"size": first(strVal(body["size"]), "1024*1024"), "n": 1},
	}
	m, err := p.postJSON(ctx, "/api/v1/services/aigc/text2image/image-synthesis", payload, true)
	if err != nil {
		return nil, err
	}
	tid := nestedOrFlatID(m)
	if tid == "" {
		if out, ok := m["output"].(map[string]any); ok {
			tid = strVal(out["task_id"])
		}
	}
	if tid == "" {
		return nil, fmt.Errorf("dashscope image task id missing")
	}
	return taskEnvelope(tid, "queued", "image", nil), nil
}

func (p *DashScopeProvider) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body, false)
}

func (p *DashScopeProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body, true)
}

func (p *DashScopeProvider) submitVideo(ctx context.Context, body map[string]any, isEdit bool) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("dashscope API key not configured")
	}
	model := first(strVal(body["model"]), p.model)
	input := map[string]any{"prompt": strVal(body["prompt"])}
	img := first(strVal(body["source_asset_id"]), strVal(body["start_asset_id"]))
	if isEdit {
		if img == "" {
			return nil, fmt.Errorf("dashscope i2v requires source_asset_id")
		}
		input["img_url"] = p.resolveStage(img)
	}
	payload := map[string]any{"model": model, "input": input, "parameters": map[string]any{}}
	if dur := asFloat(body["duration"]); dur > 0 {
		payload["parameters"].(map[string]any)["duration"] = dur
	}
	m, err := p.postJSON(ctx, "/api/v1/services/aigc/video-generation/video-synthesis", payload, true)
	if err != nil {
		return nil, err
	}
	tid := ""
	if out, ok := m["output"].(map[string]any); ok {
		tid = strVal(out["task_id"])
	}
	if tid == "" {
		tid = nestedOrFlatID(m)
	}
	if tid == "" {
		return nil, fmt.Errorf("dashscope video task id missing")
	}
	return taskEnvelope(tid, "queued", "video", nil), nil
}

func (p *DashScopeProvider) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	m, err := p.do(req)
	if err != nil {
		return nil, err
	}
	outp, _ := m["output"].(map[string]any)
	status := mapArkStatus(strVal(outp["task_status"]))
	if status == "queued" {
		status = mapArkStatus(strVal(m["task_status"]))
	}
	kind := "video"
	out := taskEnvelope(taskID, status, kind, nil)
	if status == "completed" {
		url := first(strVal(outp["video_url"]), dashResultsURL(outp))
		if url == "" {
			return nil, fmt.Errorf("dashscope completed task missing video/image url")
		}
		p.mu.Lock()
		p.urls[taskID] = url
		p.mu.Unlock()
		out["asset_ids"] = []string{url}
	}
	return out, nil
}

func (p *DashScopeProvider) DownloadResult(_ context.Context, providerRef, dest string) error {
	p.mu.Lock()
	u := p.urls[providerRef]
	p.mu.Unlock()
	if u == "" {
		u = providerRef
	}
	return downloadURL(p.client, u, dest, nil)
}

func (p *DashScopeProvider) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	return pollToSSE(ctx, func() (map[string]any, error) { return p.GetTask(ctx, taskID) })
}

func (p *DashScopeProvider) postJSON(ctx context.Context, path string, body any, async bool) (map[string]any, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if async {
		req.Header.Set("X-DashScope-Async", "enable")
	}
	return p.do(req)
}

func (p *DashScopeProvider) do(req *http.Request) (map[string]any, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("dashscope %s failed (%d): %s", req.URL.Path, resp.StatusCode, truncate(string(b), 400))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func dashResultsURL(out map[string]any) string {
	if res, ok := out["results"].([]any); ok && len(res) > 0 {
		if r, ok := res[0].(map[string]any); ok {
			return first(strVal(r["url"]), strVal(r["video_url"]))
		}
	}
	return strVal(out["output_video_url"])
}
