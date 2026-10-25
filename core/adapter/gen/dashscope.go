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
	base   string // native AIGC root (no /compatible-mode, no /api/v1)
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	staged map[string]stagedFile
	urls   map[string]string
}

func NewDashScope(base, apiKey, model string) *DashScopeProvider {
	if model == "" {
		model = "wan2.2-t2v-plus"
	}
	return &DashScopeProvider{
		base:   normalizeDashScopeBase(base),
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
		staged: map[string]stagedFile{},
		urls:   map[string]string{},
	}
}

// normalizeDashScopeBase maps LLM-style OpenAI-compatible URLs onto the
// native DashScope host used by /api/v1/services/aigc/* and /api/v1/tasks/*.
func normalizeDashScopeBase(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "https://dashscope.aliyuncs.com"
	}
	low := strings.ToLower(base)
	for _, suf := range []string{
		"/compatible-mode/v1",
		"/compatible-mode",
		"/api/v1",
	} {
		if strings.HasSuffix(low, suf) {
			base = base[:len(base)-len(suf)]
			base = strings.TrimRight(base, "/")
			low = strings.ToLower(base)
		}
	}
	if base == "" {
		return "https://dashscope.aliyuncs.com"
	}
	return base
}

func (p *DashScopeProvider) ID() string { return "dashscope" }

func (p *DashScopeProvider) Health(_ context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "missing DashScope API key"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "dashscope-wan"}, nil
}

func (p *DashScopeProvider) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("dashscope API key not configured")
	}
	var errs []string
	// 1) OpenAI-compat list — Token Plan returns the full subscription set here
	// (qwen-image-*, wan2.7-image*, wan3.0-video, chat models, …). No local catalog.
	if items, err := p.listCompatModels(ctx, media); err == nil && len(items) > 0 {
		return ensureConfiguredModel(items, p.model, media, p.model), nil
	} else if err != nil {
		errs = append(errs, err.Error())
	}
	// 2) Native Model Studio list (workspace / intl open-platform hosts).
	if items, err := p.listNativeModels(ctx, media); err == nil && len(items) > 0 {
		return ensureConfiguredModel(items, p.model, media, p.model), nil
	} else if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) == 0 {
		errs = append(errs, "empty model list")
	}
	return nil, fmt.Errorf("dashscope list models failed: %s", strings.Join(errs, "; "))
}

func (p *DashScopeProvider) listCompatModels(ctx context.Context, media string) ([]map[string]any, error) {
	m, err := p.getURL(ctx, p.base+"/compatible-mode/v1/models")
	if err != nil {
		return nil, err
	}
	return parseOpenAIStyleModels(m, media)
}

func (p *DashScopeProvider) listNativeModels(ctx context.Context, media string) ([]map[string]any, error) {
	var out []map[string]any
	seen := map[string]bool{}
	pageSize := 100
	for page := 1; page <= 50; page++ {
		path := "/api/v1/models?" + dashscopeModelsQuery(media, page, pageSize)
		m, err := p.getURL(ctx, p.base+path)
		if err != nil {
			return nil, err
		}
		items, total, err := parseDashScopeModelsPage(m, media)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			id := strVal(it["id"])
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, it)
		}
		if total > 0 && len(seen) >= total {
			break
		}
		if len(items) < pageSize {
			break
		}
	}
	return out, nil
}

func (p *DashScopeProvider) getURL(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")
	return p.do(req)
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
	model := first(strVal(body["model"]), "qwen-image-3.0-pro")
	prompt := strVal(body["prompt"])
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	size := dashImageSize(strVal(body["size"]))
	if dashUsesMultimodalImage(model) {
		return p.createImageMultimodal(ctx, model, prompt, size)
	}
	payload := map[string]any{
		"model": model,
		"input": map[string]any{"prompt": prompt},
		"parameters": map[string]any{"size": size, "n": 1},
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

// dashUsesMultimodalImage: Qwen-Image / Wan 2.6+ image / Wan 2.7-image / Z-Image
// use /aigc/multimodal-generation/generation (sync). Works on both open-platform
// (dashscope.aliyuncs.com) and Token Plan (token-plan.*.maas.aliyuncs.com) hosts.
// Older wanx* stay on text2image async (open-platform only).
func dashUsesMultimodalImage(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "qwen-image"):
		return true
	case strings.HasPrefix(m, "wan2.6-image"), strings.HasPrefix(m, "wan2.7-image"),
		strings.HasPrefix(m, "wan2.6-t2i"), strings.HasPrefix(m, "wan2.5-t2i"):
		return true
	case strings.HasPrefix(m, "wan2.6"), strings.HasPrefix(m, "wan2.7"):
		// wan2.6-t2v / wan2.7-i2v are video — not multimodal image.
		if strings.Contains(m, "-t2v") || strings.Contains(m, "-i2v") || strings.Contains(m, "-kf2v") || strings.Contains(m, "video") {
			return false
		}
		return strings.Contains(m, "image") || strings.Contains(m, "t2i")
	case strings.HasPrefix(m, "z-image"):
		return true
	default:
		return false
	}
}

func dashImageSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return "1024*1024"
	}
	return strings.ReplaceAll(size, "x", "*")
}

func (p *DashScopeProvider) createImageMultimodal(ctx context.Context, model, prompt, size string) (map[string]any, error) {
	params := map[string]any{"n": 1}
	if size != "" {
		params["size"] = size
	}
	payload := map[string]any{
		"model": model,
		"input": map[string]any{
			"messages": []any{
				map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{"text": prompt},
					},
				},
			},
		},
		"parameters": params,
	}
	m, err := p.postJSON(ctx, "/api/v1/services/aigc/multimodal-generation/generation", payload, false)
	if err != nil {
		return nil, err
	}
	url := dashMultimodalImageURL(m)
	if url == "" {
		// Some workspace/model combos still return async task ids.
		tid := ""
		if out, ok := m["output"].(map[string]any); ok {
			tid = strVal(out["task_id"])
		}
		if tid == "" {
			tid = nestedOrFlatID(m)
		}
		if tid != "" {
			return taskEnvelope(tid, "queued", "image", nil), nil
		}
		return nil, fmt.Errorf("dashscope multimodal image missing url: %v", m)
	}
	tid := "img_" + first(strVal(m["request_id"]), fmt.Sprintf("%d", time.Now().UnixNano()))
	p.mu.Lock()
	p.urls[tid] = url
	p.mu.Unlock()
	return taskEnvelope(tid, "completed", "image", map[string]any{"asset_ids": []string{url}}), nil
}

func dashMultimodalImageURL(m map[string]any) string {
	out, _ := m["output"].(map[string]any)
	if out == nil {
		return ""
	}
	if u := dashResultsURL(out); u != "" {
		return u
	}
	choices, _ := out["choices"].([]any)
	for _, c := range choices {
		cm, _ := c.(map[string]any)
		msg, _ := cm["message"].(map[string]any)
		content, _ := msg["content"].([]any)
		for _, part := range content {
			pm, _ := part.(map[string]any)
			if u := first(strVal(pm["image"]), strVal(pm["url"])); u != "" {
				return u
			}
		}
	}
	return ""
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
	// Sync multimodal image tasks are cached locally (no /tasks poll).
	p.mu.Lock()
	if u := p.urls[taskID]; u != "" && strings.HasPrefix(taskID, "img_") {
		p.mu.Unlock()
		return taskEnvelope(taskID, "completed", "image", map[string]any{"asset_ids": []string{u}}), nil
	}
	p.mu.Unlock()

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
	url := first(strVal(outp["video_url"]), dashResultsURL(outp))
	kind := "video"
	if url == "" {
		if img := first(strVal(outp["image_url"]), dashResultsURL(outp)); img != "" {
			url = img
			kind = "image"
		}
	} else if strings.Contains(strings.ToLower(url), ".png") || strings.Contains(strings.ToLower(url), ".jpg") {
		kind = "image"
	}
	out := taskEnvelope(taskID, status, kind, nil)
	if status == "completed" {
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
		msg := truncate(string(b), 400)
		if strings.Contains(string(b), "Unpurchased") || strings.Contains(string(b), "Model not exist") {
			return nil, fmt.Errorf("dashscope %s failed (%d): model not in this Key/plan — Token Plan 请选套餐列表中的 id（如 qwen-image-3.0-pro、wan2.7-image），并保持 Base URL 为 token-plan.*.maas…；开放平台用 sk- + dashscope.aliyuncs.com。raw: %s", req.URL.Path, resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("dashscope %s failed (%d): %s", req.URL.Path, resp.StatusCode, msg)
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
