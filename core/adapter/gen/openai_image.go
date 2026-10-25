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

// OpenAIImageProvider speaks OpenAI Images API (gpt-image / DALL·E).
// Video methods fail loud — use a video vendor for CreateVideo.
type OpenAIImageProvider struct {
	base   string
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	urls   map[string]string
}

func NewOpenAIImage(base, apiKey, model string) *OpenAIImageProvider {
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAIImageProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: strings.TrimSpace(apiKey),
		model:  first(strings.TrimSpace(model), "gpt-image-1"),
		client: &http.Client{Timeout: 120 * time.Second},
		urls:   map[string]string{},
	}
}

func (p *OpenAIImageProvider) ID() string { return "openai" }

func (p *OpenAIImageProvider) Health(ctx context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "api key missing"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "openai-images"}, nil
}

func (p *OpenAIImageProvider) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}
	m, err := p.getJSON(ctx, "/models")
	if err != nil {
		return nil, fmt.Errorf("openai list models: %w", err)
	}
	items, err := parseOpenAIStyleModels(m, media)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("openai /models returned no models")
	}
	return ensureConfiguredModel(items, p.model, media, p.model), nil
}

func (p *OpenAIImageProvider) getJSON(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, truncate(string(b), 400))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *OpenAIImageProvider) GetRegistry(ctx context.Context) (map[string]any, error) {
	items, _ := p.ListModels(ctx, "image")
	return map[string]any{"models": items}, nil
}

func (p *OpenAIImageProvider) ListAdapters(_ context.Context, _ string) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (p *OpenAIImageProvider) StageInput(_ context.Context, path, mime string) (map[string]any, error) {
	return map[string]any{"id": path, "mime": mime}, nil
}

func (p *OpenAIImageProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}
	model := first(strVal(body["model"]), p.model, "gpt-image-1")
	prompt := strVal(body["prompt"])
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
		"n":      1,
		"size":   first(strVal(body["size"]), "1024x1024"),
	}
	m, err := p.postJSON(ctx, "/images/generations", payload)
	if err != nil {
		return nil, err
	}
	url := firstOpenAIImageURL(m)
	if url == "" {
		return nil, fmt.Errorf("openai image response missing url/b64: %v", m)
	}
	tid := "img_" + fmt.Sprintf("%d", time.Now().UnixNano())
	p.mu.Lock()
	p.urls[tid] = url
	p.mu.Unlock()
	return taskEnvelope(tid, "completed", "image", map[string]any{"asset_ids": []string{url}}), nil
}

func (p *OpenAIImageProvider) CreateVideo(_ context.Context, _ map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("openai image provider does not support CreateVideo")
}

func (p *OpenAIImageProvider) EditVideo(_ context.Context, _ map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("openai image provider does not support EditVideo")
}

func (p *OpenAIImageProvider) GetTask(_ context.Context, taskID string) (map[string]any, error) {
	p.mu.Lock()
	url := p.urls[taskID]
	p.mu.Unlock()
	if url == "" {
		return nil, fmt.Errorf("unknown openai image task %q", taskID)
	}
	return taskEnvelope(taskID, "completed", "image", map[string]any{"asset_ids": []string{url}}), nil
}

func (p *OpenAIImageProvider) DownloadResult(ctx context.Context, providerRef, dest string) error {
	p.mu.Lock()
	u := p.urls[providerRef]
	p.mu.Unlock()
	if u == "" {
		u = providerRef
	}
	if strings.HasPrefix(u, "data:") {
		return writeDataURIFile(u, dest)
	}
	return downloadURL(p.client, u, dest, nil)
}

func (p *OpenAIImageProvider) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	return pollToSSE(ctx, func() (map[string]any, error) { return p.GetTask(ctx, taskID) })
}

func (p *OpenAIImageProvider) postJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai HTTP %d: %s", resp.StatusCode, truncate(string(b), 400))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func firstOpenAIImageURL(m map[string]any) string {
	data, _ := m["data"].([]any)
	if len(data) == 0 {
		return ""
	}
	row, _ := data[0].(map[string]any)
	if u := strVal(row["url"]); u != "" {
		return u
	}
	if b64 := strVal(row["b64_json"]); b64 != "" {
		return "data:image/png;base64," + b64
	}
	return ""
}

var _ port.GenerationProvider = (*OpenAIImageProvider)(nil)
