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
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

type MakeProvider struct {
	base   string
	apiKey string
	client *http.Client
}

func NewMake(base, apiKey string) *MakeProvider {
	if base == "" {
		base = "http://127.0.0.1:7800"
	}
	return &MakeProvider{
		base:   strings.TrimRight(base, "/"),
		apiKey: strings.TrimSpace(apiKey),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *MakeProvider) ID() string { return "danmo-make" }

func (p *MakeProvider) headers() http.Header {
	h := http.Header{"Accept": []string{"application/json"}}
	if p.apiKey != "" {
		h.Set("Authorization", "Bearer "+p.apiKey)
		h.Set("X-API-Key", p.apiKey)
	}
	return h
}

func (p *MakeProvider) BaseURL() string { return p.base }

func (p *MakeProvider) Forward(ctx context.Context, method, path string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.base+path, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header = p.headers()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b, nil
}

func (p *MakeProvider) doJSON(ctx context.Context, method, path string, body any) (map[string]any, []byte, int, error) {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, nil, 0, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base+path, rdr)
	if err != nil {
		return nil, nil, 0, err
	}
	req.Header = p.headers()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, b, resp.StatusCode, fmt.Errorf("Make %s %s failed (%d): %s", method, path, resp.StatusCode, truncate(string(b), 300))
	}
	if len(b) == 0 {
		return map[string]any{}, b, resp.StatusCode, nil
	}
	var obj any
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, b, resp.StatusCode, err
	}
	if m, ok := obj.(map[string]any); ok {
		return m, b, resp.StatusCode, nil
	}
	return map[string]any{"value": obj}, b, resp.StatusCode, nil
}

func (p *MakeProvider) Health(ctx context.Context) (map[string]any, error) {
	m, _, _, err := p.doJSON(ctx, http.MethodGet, "/api/system/health", nil)
	return m, err
}

func (p *MakeProvider) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	// Danmo Make (DanQing-Studio): GET /api/models?installed=true[&media=]
	// installed=true → only models with at least one ready (downloaded) version.
	q := []string{"installed=true"}
	if media != "" {
		q = append(q, "media="+media)
	}
	path := "/api/models?" + strings.Join(q, "&")
	m, raw, _, err := p.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	items, err := parseMakeModelsPayload(m, raw, media)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func parseMakeModelsPayload(m map[string]any, raw []byte, media string) ([]map[string]any, error) {
	if items, ok := m["items"].([]any); ok {
		return asMaps(items), nil
	}
	if items, ok := m["models"].([]any); ok {
		return asMaps(items), nil
	}
	// Current Make returns {"models": { "<id>": { media, family, actions, installed, … } }}
	if models, ok := m["models"].(map[string]any); ok {
		out := make([]map[string]any, 0, len(models))
		for id, rawRow := range models {
			row := map[string]any{"id": id, "label": id}
			if mm, ok := rawRow.(map[string]any); ok {
				for k, v := range mm {
					row[k] = v
				}
				if lbl := makeModelLabel(mm, id); lbl != "" {
					row["label"] = lbl
				}
				if media == "" {
					if med := strVal(mm["media"]); med != "" {
						row["media"] = med
					}
				} else {
					row["media"] = media
				}
			}
			out = append(out, row)
		}
		return out, nil
	}
	var arr []any
	if json.Unmarshal(raw, &arr) == nil {
		return asMaps(arr), nil
	}
	return nil, fmt.Errorf("unexpected /api/models payload from Make")
}

func makeModelLabel(m map[string]any, fallback string) string {
	if s := strVal(m["label"]); s != "" {
		return s
	}
	switch n := m["name"].(type) {
	case string:
		if strings.TrimSpace(n) != "" {
			return strings.TrimSpace(n)
		}
	case map[string]any:
		if zh := strVal(n["zh"]); zh != "" {
			return zh
		}
		if en := strVal(n["en"]); en != "" {
			return en
		}
	}
	if fam := strVal(m["family"]); fam != "" {
		return fam
	}
	return fallback
}

func (p *MakeProvider) GetRegistry(ctx context.Context) (map[string]any, error) {
	m, _, _, err := p.doJSON(ctx, http.MethodGet, "/api/registry", nil)
	return m, err
}

func (p *MakeProvider) ListAdapters(ctx context.Context, forModel string) (any, error) {
	path := "/api/adapters"
	if forModel != "" {
		path += "?for_model=" + forModel
	}
	m, raw, _, err := p.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var v any
	_ = json.Unmarshal(raw, &v)
	if v != nil {
		return v, nil
	}
	return m, nil
}

func (p *MakeProvider) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	normalized, req, err := NormalizeImageBody(body)
	if err != nil {
		return nil, err
	}
	path := "/api/images/generations"
	if req.Mode == port.ImageModeEdit || req.Mode == port.ImageModeInpaint || req.Mode == port.ImageModeI2I {
		// Make uses edits for conditioned image jobs when available.
		path = "/api/images/edits"
	}
	m, _, status, err := p.doJSON(ctx, http.MethodPost, path, normalized)
	if err != nil && path == "/api/images/edits" && status == 404 {
		// Older Make builds may lack /edits — fall back to generations with mode set.
		m, _, _, err = p.doJSON(ctx, http.MethodPost, "/api/images/generations", normalized)
	}
	return m, err
}

func (p *MakeProvider) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	normalized, req, err := NormalizeVideoBody(body)
	if err != nil {
		return nil, err
	}
	// Make: generations for text-only; edits when first_frame / refs / video are present.
	path := "/api/videos/generations"
	if HasConditionedVideoMedia(req) {
		path = "/api/videos/edits"
	}
	m, _, _, err := p.doJSON(ctx, http.MethodPost, path, normalized)
	return m, err
}

func (p *MakeProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	normalized, _, err := NormalizeVideoBody(body)
	if err != nil {
		return nil, err
	}
	m, _, _, err := p.doJSON(ctx, http.MethodPost, "/api/videos/edits", normalized)
	return m, err
}

func (p *MakeProvider) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	m, _, _, err := p.doJSON(ctx, http.MethodGet, "/api/tasks/"+taskID, nil)
	return m, err
}

func (p *MakeProvider) DownloadResult(ctx context.Context, providerRef, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/api/assets/"+providerRef+"/file", nil)
	if err != nil {
		return err
	}
	req.Header = p.headers()
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Make result download failed (%d): %s", resp.StatusCode, truncate(string(b), 300))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func (p *MakeProvider) StageInput(ctx context.Context, path, mime string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("stage path missing: %s", path)
	}
	defer f.Close()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, err
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/api/assets", &buf)
	if err != nil {
		return nil, err
	}
	req.Header = p.headers()
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Make stage_input_file failed (%d): %s", resp.StatusCode, truncate(string(b), 300))
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (p *MakeProvider) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.base+"/api/tasks/"+taskID+"/stream", nil)
	if err != nil {
		return nil, err
	}
	req.Header = p.headers()
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("Make task stream failed (%d)", resp.StatusCode)
	}
	ch := make(chan port.SSEEvent)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		readSSE(resp.Body, ch)
	}()
	return ch, nil
}

func asMaps(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
