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
	req, err := ParseImageRequest(body)
	if err != nil {
		return nil, err
	}
	model := first(req.Model, "qwen-image-3.0-pro")
	prompt := req.Prompt
	size := dashImageSize(req.Size)
	refs := filterNonEmptyURLs(req.ReferenceImages)
	if req.SourceID != "" {
		// Ensure source participates as a reference for i2i/edit.
		found := false
		for _, r := range refs {
			if r == req.SourceID {
				found = true
				break
			}
		}
		if !found {
			refs = append([]string{req.SourceID}, refs...)
		}
	}
	switch req.Mode {
	case port.ImageModeT2I:
		if dashUsesMultimodalImage(model) {
			return p.createImageMultimodal(ctx, model, prompt, size, nil)
		}
	case port.ImageModeI2I, port.ImageModeEdit:
		if len(refs) == 0 {
			return nil, fmt.Errorf("dashscope %s requires reference/source image", req.Mode)
		}
		return p.createImageMultimodal(ctx, model, prompt, size, refs)
	case port.ImageModeInpaint:
		return nil, fmt.Errorf("dashscope image inpaint is not supported via this adapter yet")
	default:
		return nil, fmt.Errorf("dashscope unsupported image mode %q", req.Mode)
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

func bodyStringIDs(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, raw := range t {
			s := strings.TrimSpace(fmt.Sprint(raw))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (p *DashScopeProvider) createImageMultimodal(ctx context.Context, model, prompt, size string, refs []string) (map[string]any, error) {
	params := map[string]any{"n": 1}
	if size != "" {
		params["size"] = size
	}
	content := make([]any, 0, len(refs)+1)
	for _, id := range refs {
		uri := p.resolveStage(id)
		if uri == "" {
			return nil, fmt.Errorf("dashscope reference image %q not staged", id)
		}
		content = append(content, map[string]any{"image": uri})
	}
	content = append(content, map[string]any{"text": prompt})
	payload := map[string]any{
		"model": model,
		"input": map[string]any{
			"messages": []any{
				map[string]any{
					"role":    "user",
					"content": content,
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
	return p.submitVideo(ctx, body)
}

func (p *DashScopeProvider) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return p.submitVideo(ctx, body)
}

func dashVideoResolution(res string) string {
	r := strings.ToUpper(strings.TrimSpace(res))
	if r == "" {
		return ""
	}
	switch {
	case strings.Contains(r, "480"), r == "480P":
		return "480P"
	case strings.Contains(r, "720"), r == "720P":
		return "720P"
	case strings.Contains(r, "1080"), r == "1080P":
		return "1080P"
	default:
		return res
	}
}

func filterNonEmptyURLs(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func (p *DashScopeProvider) resolveVideoURLs(req *port.VideoRequest) {
	if req.FirstFrame != "" {
		req.FirstFrame = p.resolveStage(req.FirstFrame)
	}
	if req.LastFrame != "" {
		req.LastFrame = p.resolveStage(req.LastFrame)
	}
	if req.SourceVideo != "" {
		req.SourceVideo = p.resolveStage(req.SourceVideo)
	}
	for i, id := range req.ReferenceImages {
		req.ReferenceImages[i] = p.resolveStage(id)
	}
	for i, id := range req.ReferenceVideos {
		req.ReferenceVideos[i] = p.resolveStage(id)
	}
}

func buildDashV2VMedia(videos, images []string, v2vSlot string) []map[string]any {
	media := make([]map[string]any, 0, len(videos)+len(images))
	if strings.EqualFold(strings.TrimSpace(v2vSlot), "first_clip") {
		media = append(media, map[string]any{"type": "first_clip", "url": videos[0]})
		return media
	}
	for _, u := range videos {
		media = append(media, map[string]any{"type": "reference_video", "url": u})
	}
	for _, u := range images {
		media = append(media, map[string]any{"type": "reference_image", "url": u})
	}
	return media
}

// buildDashVideoInput maps a normalized VideoRequest onto DashScope input.
// protocol / v2vSlot come from providers.video_model_params (not model-id heuristics).
// Task kind is derived from filled media slots.
func buildDashVideoInput(model string, req port.VideoRequest, protocol, v2vSlot string) (map[string]any, error) {
	_ = model
	input := map[string]any{}
	if req.Prompt != "" {
		input["prompt"] = req.Prompt
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		input["negative_prompt"] = np
	}
	kind := DeriveVideoMode(req)

	switch protocol {
	case "", "dashscope_img_url":
		switch kind {
		case port.VideoModeT2V:
			return input, nil
		case port.VideoModeI2V, port.VideoModeFLF:
			if req.FirstFrame == "" {
				return nil, fmt.Errorf("dashscope i2v requires source_asset_id")
			}
			input["img_url"] = req.FirstFrame
			return input, nil
		default:
			return nil, fmt.Errorf("dashscope protocol %q does not support slot combination %s (configure video_model_params.protocol=dashscope_media)", protocol, kind)
		}

	case "dashscope_media":
		switch kind {
		case port.VideoModeT2V:
			return input, nil
		case port.VideoModeI2V:
			input["media"] = []map[string]any{{"type": "first_frame", "url": req.FirstFrame}}
			return input, nil
		case port.VideoModeFLF:
			input["media"] = []map[string]any{
				{"type": "first_frame", "url": req.FirstFrame},
				{"type": "last_frame", "url": req.LastFrame},
			}
			return input, nil
		case port.VideoModeR2V:
			refs := filterNonEmptyURLs(req.ReferenceImages)
			media := make([]map[string]any, 0, len(refs)+len(req.ReferenceVideos))
			for _, u := range refs {
				media = append(media, map[string]any{"type": "reference_image", "url": u})
			}
			for _, u := range filterNonEmptyURLs(req.ReferenceVideos) {
				media = append(media, map[string]any{"type": "reference_video", "url": u})
			}
			if len(media) == 0 {
				return nil, fmt.Errorf("dashscope r2v requires reference_asset_ids")
			}
			input["media"] = media
			return input, nil
		case port.VideoModeV2V:
			videos := filterNonEmptyURLs(req.ReferenceVideos)
			if req.SourceVideo != "" {
				found := false
				for _, v := range videos {
					if v == req.SourceVideo {
						found = true
						break
					}
				}
				if !found {
					videos = append([]string{req.SourceVideo}, videos...)
				}
			}
			if len(videos) == 0 {
				return nil, fmt.Errorf("dashscope v2v requires source_video_asset_id or reference_video_asset_ids")
			}
			input["media"] = buildDashV2VMedia(videos, filterNonEmptyURLs(req.ReferenceImages), v2vSlot)
			return input, nil
		default:
			return nil, fmt.Errorf("dashscope unsupported media slot combination %q", kind)
		}

	default:
		return nil, fmt.Errorf("dashscope unknown film_protocol %q", protocol)
	}
}

func (p *DashScopeProvider) submitVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("dashscope API key not configured")
	}
	req, err := ParseVideoRequest(body)
	if err != nil {
		return nil, err
	}
	if req.Model == "" {
		req.Model = p.model
	}
	protocol := filmProtocolFromBody(body)
	if protocol == "" {
		return nil, fmt.Errorf("dashscope model %q missing film_protocol (set providers.video_model_params)", req.Model)
	}
	p.resolveVideoURLs(&req)
	input, err := buildDashVideoInput(req.Model, req, protocol, strVal(body["film_v2v_slot"]))
	if err != nil {
		return nil, err
	}
	params := map[string]any{}
	if req.DurationSec > 0 {
		params["duration"] = int(req.DurationSec)
	}
	if res := dashVideoResolution(first(req.Resolution, req.Size)); res != "" {
		params["resolution"] = res
	}
	if req.Ratio != "" {
		params["ratio"] = req.Ratio
	}
	if req.Seed != 0 {
		params["seed"] = req.Seed
	}
	payload := map[string]any{"model": req.Model, "input": input, "parameters": params}
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
