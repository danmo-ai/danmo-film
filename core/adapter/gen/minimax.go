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

// MiniMaxProvider — Hailuo / H3 video + image.
// Video: only MiniMax-H3 and later via POST /v2/video_generation (content[]).
type MiniMaxProvider struct {
	base   string
	apiKey string
	model  string
	client *http.Client
	mu     sync.Mutex
	files  map[string]string // task/file id -> download url
	staged map[string]stagedFile
	h3Task map[string]bool // task ids created on v2/H3 API
}

func NewMiniMax(base, apiKey, model string) *MiniMaxProvider {
	if base == "" {
		base = "https://api.minimax.chat"
	}
	if model == "" {
		model = "MiniMax-H3"
	}
	return &MiniMaxProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
		files:  map[string]string{},
		staged: map[string]stagedFile{},
		h3Task: map[string]bool{},
	}
}

func (p *MiniMaxProvider) ID() string { return "minimax" }

func (p *MiniMaxProvider) Health(_ context.Context) (map[string]any, error) {
	if p.apiKey == "" {
		return map[string]any{"ok": false, "configured": false, "error": "missing MiniMax API key"}, nil
	}
	return map[string]any{"ok": true, "configured": true, "vendor": "minimax-h3"}, nil
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
	// Capabilities come from providers.video_model_params / image_model_params via Router registry merge.
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
	// Keep a local stage entry so H3 content[] can use data URIs (preferred over file_id).
	uri, err := fileDataURI(path, mime)
	if err != nil {
		return nil, err
	}
	id := "stg_" + filepathBase(path)
	p.mu.Lock()
	p.staged[id] = stagedFile{Path: path, MIME: mime, URI: uri}
	p.mu.Unlock()

	// Also try vendor upload for callers that still expect a file_id.
	f, err := os.Open(path)
	if err != nil {
		return map[string]any{"id": id}, nil
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("purpose", "prompt")
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return map[string]any{"id": id}, nil
	}
	if _, err := io.Copy(fw, f); err != nil {
		return map[string]any{"id": id}, nil
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/v1/files/upload", &buf)
	if err != nil {
		return map[string]any{"id": id}, nil
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())
	m, err := p.do(req)
	if err != nil {
		return map[string]any{"id": id}, nil
	}
	fid := first(strVal(m["file_id"]), nestedOrFlatID(m))
	if file, ok := m["file"].(map[string]any); ok && fid == "" {
		fid = strVal(file["file_id"])
	}
	if fid != "" {
		p.mu.Lock()
		p.staged[fid] = stagedFile{Path: path, MIME: mime, URI: uri}
		p.mu.Unlock()
		return map[string]any{"id": fid}, nil
	}
	return map[string]any{"id": id}, nil
}

func (p *MiniMaxProvider) resolveMediaURL(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") || strings.HasPrefix(id, "data:") {
		return id
	}
	p.mu.Lock()
	s, ok := p.staged[id]
	p.mu.Unlock()
	if ok {
		if s.URI != "" {
			return s.URI
		}
		if s.Path != "" {
			if uri, err := fileDataURI(s.Path, s.MIME); err == nil {
				return uri
			}
		}
	}
	return id
}

func (p *MiniMaxProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	req, err := ParseImageRequest(body)
	if err != nil {
		return nil, err
	}
	model := first(req.Model, "image-01")
	// MiniMax image-01 family: t2i only for now.
	if req.Mode != port.ImageModeT2I {
		return nil, fmt.Errorf("minimax image model %s does not support mode %s (only t2i)", model, req.Mode)
	}
	payload := map[string]any{
		"model":  model,
		"prompt": req.Prompt,
	}
	if req.Size != "" {
		payload["size"] = req.Size
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
	return p.submitVideo(ctx, body)
}

func (p *MiniMaxProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body)
}

func (p *MiniMaxProvider) submitVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("minimax API key not configured")
	}
	req, err := ParseVideoRequest(body)
	if err != nil {
		return nil, err
	}
	model := first(req.Model, p.model)
	protocol := filmProtocolFromBody(body)
	if protocol == "" {
		return nil, fmt.Errorf("minimax video model %q missing film_protocol (set providers.video_model_params)", model)
	}
	if protocol != "minimax_h3_content" {
		return nil, fmt.Errorf("minimax adapter only supports protocol minimax_h3_content (got %q for %s)", protocol, model)
	}
	if modes, ok := body["film_modes"].([]string); ok && len(modes) > 0 {
		if err := AssertVideoModeSupported(map[string]any{"modes": modes}, req.Mode); err != nil {
			return nil, err
		}
	}
	payload, err := buildMiniMaxH3VideoPayload(model, req, p.resolveMediaURL)
	if err != nil {
		return nil, err
	}
	m, err := p.postJSON(ctx, "/v2/video_generation", payload)
	if err != nil {
		return nil, err
	}
	id := first(strVal(m["task_id"]), nestedOrFlatID(m))
	if id == "" {
		return nil, fmt.Errorf("minimax video task_id missing")
	}
	p.mu.Lock()
	p.h3Task[id] = true
	p.mu.Unlock()
	return taskEnvelope(id, "queued", "video", nil), nil
}

func buildMiniMaxH3VideoPayload(model string, req port.VideoRequest, resolve func(string) string) (map[string]any, error) {
	content := make([]map[string]any, 0, 8)
	if req.Prompt != "" {
		content = append(content, map[string]any{"type": "text", "text": req.Prompt})
	}
	switch req.Mode {
	case port.VideoModeT2V:
		// text only
	case port.VideoModeI2V:
		u := resolve(req.FirstFrame)
		if u == "" {
			return nil, fmt.Errorf("minimax i2v requires first_frame url")
		}
		content = append(content, map[string]any{
			"type": "image_url", "image_url": map[string]any{"url": u}, "role": "first_frame",
		})
	case port.VideoModeFLF:
		fu, lu := resolve(req.FirstFrame), resolve(req.LastFrame)
		content = append(content,
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": fu}, "role": "first_frame"},
			map[string]any{"type": "image_url", "image_url": map[string]any{"url": lu}, "role": "last_frame"},
		)
	case port.VideoModeR2V:
		for _, id := range filterNonEmptyURLs(req.ReferenceImages) {
			content = append(content, map[string]any{
				"type": "image_url", "image_url": map[string]any{"url": resolve(id)}, "role": "reference_image",
			})
		}
		for _, id := range filterNonEmptyURLs(req.ReferenceVideos) {
			content = append(content, map[string]any{
				"type": "video_url", "video_url": map[string]any{"url": resolve(id)}, "role": "reference_video",
			})
		}
		if len(content) < 2 { // text + at least one ref
			return nil, fmt.Errorf("minimax r2v requires reference images/videos")
		}
	case port.VideoModeV2V:
		nVid := 0
		for _, id := range filterNonEmptyURLs(req.ReferenceVideos) {
			content = append(content, map[string]any{
				"type": "video_url", "video_url": map[string]any{"url": resolve(id)}, "role": "reference_video",
			})
			nVid++
		}
		for _, id := range filterNonEmptyURLs(req.ReferenceImages) {
			content = append(content, map[string]any{
				"type": "image_url", "image_url": map[string]any{"url": resolve(id)}, "role": "reference_image",
			})
		}
		if nVid == 0 {
			return nil, fmt.Errorf("minimax v2v requires reference video")
		}
	default:
		return nil, fmt.Errorf("minimax unsupported video mode %q", req.Mode)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("minimax video content empty")
	}
	payload := map[string]any{
		"model":   model,
		"content": content,
	}
	if req.DurationSec > 0 {
		payload["duration"] = int(req.DurationSec)
	}
	if res := first(req.Resolution, dashVideoResolution(req.Size)); res != "" {
		payload["resolution"] = res
	}
	if req.Ratio != "" {
		payload["ratio"] = req.Ratio
	}
	return payload, nil
}

func (p *MiniMaxProvider) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	p.mu.Lock()
	if u, ok := p.files[taskID]; ok {
		p.mu.Unlock()
		return taskEnvelope(taskID, "completed", "image", map[string]any{"asset_ids": []string{u}}), nil
	}
	isH3 := p.h3Task[taskID]
	p.mu.Unlock()

	if isH3 {
		return p.getH3Task(ctx, taskID)
	}
	// Try H3 query first (in case process restarted); fall back to v1.
	if out, err := p.getH3Task(ctx, taskID); err == nil {
		return out, nil
	}
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

func (p *MiniMaxProvider) getH3Task(ctx context.Context, taskID string) (map[string]any, error) {
	m, err := p.getJSON(ctx, "/v2/query/video_generation/"+taskID)
	if err != nil {
		return nil, err
	}
	task, _ := m["task"].(map[string]any)
	if task == nil {
		task = m
	}
	status := mapMiniMaxH3Status(strVal(task["status"]))
	out := taskEnvelope(taskID, status, "video", nil)
	if status == "completed" {
		url := ""
		if content, ok := task["content"].(map[string]any); ok {
			url = strVal(content["url"])
		}
		if url == "" {
			url = strVal(task["download_url"])
		}
		if url == "" {
			return nil, fmt.Errorf("minimax H3 completed task missing content.url")
		}
		p.mu.Lock()
		p.files[taskID] = url
		p.h3Task[taskID] = true
		p.mu.Unlock()
		out["asset_ids"] = []string{url}
	}
	if status == "failed" {
		if errObj, ok := task["error"].(map[string]any); ok {
			out["error"] = first(strVal(errObj["message"]), strVal(errObj["code"]), fmt.Sprint(errObj))
		} else if e := strVal(task["error"]); e != "" {
			out["error"] = e
		}
	}
	return out, nil
}

func mapMiniMaxH3Status(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "succeeded", "success", "completed":
		return "completed"
	case "failed", "cancelled", "canceled", "error":
		return "failed"
	case "queueing", "queued", "pending":
		return "queued"
	case "preparing", "processing", "running":
		return "running"
	default:
		return mapArkStatus(s)
	}
}

func (p *MiniMaxProvider) DownloadResult(ctx context.Context, providerRef, dest string) error {
	if strings.HasPrefix(providerRef, "http") {
		return downloadURL(p.client, providerRef, dest, map[string]string{"Authorization": "Bearer " + p.apiKey})
	}
	p.mu.Lock()
	if u := p.files[providerRef]; strings.HasPrefix(u, "http") {
		p.mu.Unlock()
		return downloadURL(p.client, u, dest, nil)
	}
	p.mu.Unlock()
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
