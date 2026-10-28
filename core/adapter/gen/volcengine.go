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

// Volcengine Ark Seedance — POST /api/v3/contents/generations/tasks
type VolcengineProvider struct {
	base   string
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	staged map[string]stagedFile
	urls   map[string]string // taskID -> result URL
}

type stagedFile struct {
	Path string
	MIME string
	URI  string
}

func NewVolcengine(base, apiKey, model string) *VolcengineProvider {
	if base == "" {
		base = "https://ark.cn-beijing.volces.com"
	}
	if model == "" {
		model = "doubao-seedance-1-0-pro-250528"
	}
	return &VolcengineProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
		staged: map[string]stagedFile{},
		urls:   map[string]string{},
	}
}

func (p *VolcengineProvider) ID() string { return "volcengine" }

func (p *VolcengineProvider) Health(ctx context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "missing Ark API key"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "volcengine-seedance"}, nil
}

func (p *VolcengineProvider) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("volcengine API key not configured")
	}
	m, err := p.getJSON(ctx, "/api/v3/models")
	if err != nil {
		return nil, fmt.Errorf("volcengine list models: %w", err)
	}
	items, err := parseOpenAIStyleModels(m, media)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("volcengine /api/v3/models returned no models")
	}
	return ensureConfiguredModel(items, p.model, media, p.model), nil
}

func (p *VolcengineProvider) GetRegistry(ctx context.Context) (map[string]any, error) {
	items, _ := p.ListModels(ctx, "")
	return map[string]any{"models": items}, nil
}

func (p *VolcengineProvider) ListAdapters(_ context.Context, _ string) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (p *VolcengineProvider) StageInput(_ context.Context, path, mime string) (map[string]any, error) {
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

func (p *VolcengineProvider) resolveStage(id string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.staged[id]; ok {
		return s.URI
	}
	return id
}

func (p *VolcengineProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	req, err := ParseImageRequest(body)
	if err != nil {
		return nil, err
	}
	if req.Mode != port.ImageModeT2I {
		return nil, fmt.Errorf("volcengine image only supports mode t2i (got %s)", req.Mode)
	}
	model := first(req.Model, "doubao-seedream-3-0-t2i-250415")
	payload := map[string]any{
		"model":           model,
		"prompt":          req.Prompt,
		"size":            first(req.Size, "1024x1024"),
		"n":               1,
		"response_format": "url",
	}
	m, err := p.postJSON(ctx, "/api/v3/images/generations", payload)
	if err != nil {
		return nil, err
	}
	url := firstImageURL(m)
	if url == "" {
		return nil, fmt.Errorf("volcengine image response missing url: %v", m)
	}
	tid := "img_" + nestedOrFlatID(m)
	if tid == "img_" {
		tid = "img_" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	p.mu.Lock()
	p.urls[tid] = url
	p.mu.Unlock()
	return taskEnvelope(tid, "completed", "image", map[string]any{"asset_ids": []string{url}}), nil
}

func (p *VolcengineProvider) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body)
}

func (p *VolcengineProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body)
}

func (p *VolcengineProvider) submitVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("volcengine API key not configured")
	}
	req, err := ParseVideoRequest(body)
	if err != nil {
		return nil, err
	}
	model := first(req.Model, p.model)
	content := []map[string]any{}
	if req.Prompt != "" {
		content = append(content, map[string]any{"type": "text", "text": req.Prompt})
	}
	switch DeriveVideoMode(req) {
	case port.VideoModeT2V:
		// text only
	case port.VideoModeI2V:
		content = append(content, map[string]any{
			"type": "image_url", "image_url": map[string]any{"url": p.resolveStage(req.FirstFrame)}, "role": "first_frame",
		})
	case port.VideoModeFLF:
		content = append(content,
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": p.resolveStage(req.FirstFrame)}, "role": "first_frame"},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": p.resolveStage(req.LastFrame)}, "role": "last_frame"},
		)
	case port.VideoModeR2V:
		for _, id := range filterNonEmptyURLs(req.ReferenceImages) {
			content = append(content, map[string]any{
				"type": "image_url", "image_url": map[string]any{"url": p.resolveStage(id)}, "role": "reference_image",
			})
		}
		for _, id := range filterNonEmptyURLs(req.ReferenceVideos) {
			content = append(content, map[string]any{
				"type": "video_url", "video_url": map[string]any{"url": p.resolveStage(id)}, "role": "reference_video",
			})
		}
	case port.VideoModeV2V:
		nVideo := 0
		for _, id := range filterNonEmptyURLs(req.ReferenceVideos) {
			content = append(content, map[string]any{
				"type": "video_url", "video_url": map[string]any{"url": p.resolveStage(id)}, "role": "reference_video",
			})
			nVideo++
		}
		for _, id := range filterNonEmptyURLs(req.ReferenceImages) {
			content = append(content, map[string]any{
				"type": "image_url", "image_url": map[string]any{"url": p.resolveStage(id)}, "role": "reference_image",
			})
		}
		if nVideo == 0 {
			return nil, fmt.Errorf("volcengine v2v requires reference video")
		}
	default:
		return nil, fmt.Errorf("volcengine unsupported media slot combination")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("volcengine video content empty")
	}
	payload := map[string]any{"model": model, "content": content}
	m, err := p.postJSON(ctx, "/api/v3/contents/generations/tasks", payload)
	if err != nil {
		return nil, err
	}
	id := nestedOrFlatID(m)
	if id == "" {
		return nil, fmt.Errorf("volcengine task id missing: %v", m)
	}
	return taskEnvelope(id, "queued", "video", nil), nil
}

func (p *VolcengineProvider) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	p.mu.Lock()
	if u, ok := p.urls[taskID]; ok {
		p.mu.Unlock()
		return taskEnvelope(taskID, "completed", "image", map[string]any{"asset_ids": []string{u}}), nil
	}
	p.mu.Unlock()
	m, err := p.getJSON(ctx, "/api/v3/contents/generations/tasks/"+taskID)
	if err != nil {
		return nil, err
	}
	status := mapArkStatus(strVal(m["status"]))
	out := taskEnvelope(taskID, status, "video", nil)
	if status == "completed" {
		url := arkVideoURL(m)
		if url == "" {
			return nil, fmt.Errorf("volcengine completed task missing video url")
		}
		p.mu.Lock()
		p.urls[taskID] = url
		p.mu.Unlock()
		out["asset_ids"] = []string{url}
	}
	if status == "failed" {
		out["error"] = first(strVal(m["error"]), fmt.Sprintf("%v", m["error"]))
	}
	return out, nil
}

func (p *VolcengineProvider) DownloadResult(ctx context.Context, providerRef, dest string) error {
	p.mu.Lock()
	u := p.urls[providerRef]
	p.mu.Unlock()
	if u == "" {
		if strings.HasPrefix(providerRef, "http") || strings.HasPrefix(providerRef, "data:") {
			u = providerRef
		}
	}
	if u == "" {
		return fmt.Errorf("no result url for %s", providerRef)
	}
	return downloadURL(p.client, u, dest, map[string]string{"Authorization": "Bearer " + p.apiKey})
}

func (p *VolcengineProvider) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	return pollToSSE(ctx, func() (map[string]any, error) { return p.GetTask(ctx, taskID) })
}

func (p *VolcengineProvider) postJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return p.do(req)
}

func (p *VolcengineProvider) getJSON(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return p.do(req)
}

func (p *VolcengineProvider) do(req *http.Request) (map[string]any, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("volcengine %s failed (%d): %s", req.URL.Path, resp.StatusCode, truncate(string(b), 400))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func mapArkStatus(s string) string {
	switch strings.ToLower(s) {
	case "succeeded", "success", "completed":
		return "completed"
	case "failed", "error":
		return "failed"
	case "running", "processing":
		return "running"
	default:
		if s == "" {
			return "queued"
		}
		return strings.ToLower(s)
	}
}

func arkVideoURL(m map[string]any) string {
	if c, ok := m["content"].(map[string]any); ok {
		if u := strVal(c["video_url"]); u != "" {
			return u
		}
	}
	if u := strVal(m["video_url"]); u != "" {
		return u
	}
	if arr, ok := m["content"].([]any); ok {
		for _, it := range arr {
			if cm, ok := it.(map[string]any); ok {
				if u := strVal(cm["video_url"]); u != "" {
					return u
				}
			}
		}
	}
	return ""
}

func firstImageURL(m map[string]any) string {
	if data, ok := m["data"].([]any); ok && len(data) > 0 {
		if d, ok := data[0].(map[string]any); ok {
			return strVal(d["url"])
		}
	}
	return strVal(m["url"])
}
