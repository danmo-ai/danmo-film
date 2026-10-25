package gen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

// MiniMax Hailuo official video_generation API.
type MiniMaxProvider struct {
	base   string
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	files  map[string]string // file_id or task -> retrieve url
	staged map[string]stagedFile
}

func NewMiniMax(base, apiKey, model string) *MiniMaxProvider {
	if base == "" {
		base = "https://api.minimax.chat"
	}
	if model == "" {
		model = "MiniMax-Hailuo-02"
	}
	return &MiniMaxProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
		files:  map[string]string{},
		staged: map[string]stagedFile{},
	}
}

func (p *MiniMaxProvider) ID() string { return "minimax" }

func (p *MiniMaxProvider) Health(_ context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "missing MiniMax API key"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "minimax-hailuo"}, nil
}

func (p *MiniMaxProvider) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	m, err := p.getJSON(ctx, "/v1/models")
	if err != nil {
		return nil, fmt.Errorf("minimax list models: %w", err)
	}
	items, err := parseMiniMaxModels(m, media)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("minimax /v1/models returned no models")
	}
	return ensureConfiguredModel(items, p.model, media, p.model), nil
}

func (p *MiniMaxProvider) GetRegistry(ctx context.Context) (map[string]any, error) {
	items, _ := p.ListModels(ctx, "")
	return map[string]any{"models": items}, nil
}

func (p *MiniMaxProvider) ListAdapters(_ context.Context, _ string) (any, error) {
	return map[string]any{"items": []any{}}, nil
}

func (p *MiniMaxProvider) StageInput(ctx context.Context, path, mime string) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("purpose", "prompt")
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, err
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/v1/files/upload", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	m, err := p.do(req)
	if err != nil {
		return nil, fmt.Errorf("minimax file upload: %w", err)
	}
	id := first(strVal(m["file_id"]), nestedOrFlatID(m))
	if file, ok := m["file"].(map[string]any); ok && id == "" {
		id = strVal(file["file_id"])
	}
	if id == "" {
		return nil, fmt.Errorf("minimax upload returned no file_id")
	}
	p.mu.Lock()
	p.staged[id] = stagedFile{Path: path, MIME: mime}
	p.mu.Unlock()
	return map[string]any{"id": id}, nil
}

func (p *MiniMaxProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	payload := map[string]any{
		"model":  first(strVal(body["model"]), "image-01"),
		"prompt": strVal(body["prompt"]),
	}
	m, err := p.postJSON(ctx, "/v1/image_generation", payload)
	if err != nil {
		return nil, err
	}
	tid := nestedOrFlatID(m)
	if tid == "" {
		tid = fmt.Sprintf("img_%d", time.Now().UnixNano())
	}
	url := minimaxImageURL(m)
	if url != "" {
		p.mu.Lock()
		p.files[tid] = url
		p.mu.Unlock()
		return taskEnvelope(tid, "completed", "image", map[string]any{"asset_ids": []string{url}}), nil
	}
	return taskEnvelope(tid, "queued", "image", nil), nil
}

func (p *MiniMaxProvider) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body, false)
}

func (p *MiniMaxProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body, true)
}

func (p *MiniMaxProvider) submitVideo(ctx context.Context, body map[string]any, isEdit bool) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	payload := map[string]any{
		"model":  first(strVal(body["model"]), p.model),
		"prompt": strVal(body["prompt"]),
	}
	firstFrame := first(strVal(body["source_asset_id"]), strVal(body["start_asset_id"]))
	lastFrame := first(strVal(body["tail_asset_id"]), strVal(body["end_asset_id"]))
	if isEdit && firstFrame == "" {
		return nil, fmt.Errorf("minimax i2v requires source_asset_id")
	}
	if firstFrame != "" {
		payload["first_frame_image"] = firstFrame
	}
	if lastFrame != "" {
		payload["last_frame_image"] = lastFrame
	}
	m, err := p.postJSON(ctx, "/v1/video_generation", payload)
	if err != nil {
		return nil, err
	}
	id := first(strVal(m["task_id"]), nestedOrFlatID(m))
	if id == "" {
		return nil, fmt.Errorf("minimax video task_id missing")
	}
	return taskEnvelope(id, "queued", "video", nil), nil
}

func (p *MiniMaxProvider) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	p.mu.Lock()
	if u, ok := p.files[taskID]; ok {
		p.mu.Unlock()
		return taskEnvelope(taskID, "completed", "image", map[string]any{"asset_ids": []string{u}}), nil
	}
	p.mu.Unlock()
	m, err := p.getJSON(ctx, "/v1/query/video_generation?task_id="+taskID)
	if err != nil {
		return nil, err
	}
	status := mapArkStatus(strVal(m["status"]))
	out := taskEnvelope(taskID, status, "video", nil)
	if status == "completed" {
		fileID := strVal(m["file_id"])
		if fileID == "" {
			return nil, fmt.Errorf("minimax completed task missing file_id")
		}
		p.mu.Lock()
		p.files[taskID] = fileID
		p.mu.Unlock()
		out["asset_ids"] = []string{fileID}
	}
	return out, nil
}

func (p *MiniMaxProvider) DownloadResult(ctx context.Context, providerRef, dest string) error {
	if strings.HasPrefix(providerRef, "http") {
		return downloadURL(p.client, providerRef, dest, map[string]string{"Authorization": "Bearer " + p.apiKey})
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/v1/files/retrieve?file_id="+providerRef, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	m, err := p.do(req)
	if err != nil {
		return err
	}
	url := strVal(m["download_url"])
	if file, ok := m["file"].(map[string]any); ok && url == "" {
		url = strVal(file["download_url"])
	}
	if url == "" {
		return fmt.Errorf("minimax retrieve missing download_url")
	}
	return downloadURL(p.client, url, dest, nil)
}

func (p *MiniMaxProvider) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	return pollToSSE(ctx, func() (map[string]any, error) { return p.GetTask(ctx, taskID) })
}

func (p *MiniMaxProvider) postJSON(ctx context.Context, path string, body any) (map[string]any, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return p.do(req)
}

func (p *MiniMaxProvider) getJSON(ctx context.Context, path string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return p.do(req)
}

func (p *MiniMaxProvider) do(req *http.Request) (map[string]any, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("minimax %s failed (%d): %s", req.URL.Path, resp.StatusCode, truncate(string(b), 400))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if bc, ok := m["base_resp"].(map[string]any); ok {
		if asFloat(bc["status_code"]) != 0 {
			return nil, fmt.Errorf("minimax error: %s", strVal(bc["status_msg"]))
		}
	}
	return m, nil
}

func minimaxImageURL(m map[string]any) string {
	if data, ok := m["data"].(map[string]any); ok {
		if img, ok := data["image_urls"].([]any); ok && len(img) > 0 {
			return strVal(img[0])
		}
	}
	return ""
}
