package gen

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

const taskIDSep = ":"

// Router dispatches image/video jobs to the profile that owns the requested model.
// Video refs use generation_profiles; image refs use image_profiles.
// Model refs are profile_id/model_id (same shape as LLM). Task ids are
// rewritten as profile_id:native_task_id so GetTask / SSE can route.
type Router struct {
	mu           sync.RWMutex
	settings     config.Settings
	videoClients map[string]port.GenerationProvider
	imageClients map[string]port.GenerationProvider
}

func NewRouter(settings config.Settings) (*Router, error) {
	r := &Router{
		settings:     config.Clone(settings),
		videoClients: map[string]port.GenerationProvider{},
		imageClients: map[string]port.GenerationProvider{},
	}
	if err := r.rebuildLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Router) ID() string { return "gen-router" }

func (r *Router) Update(settings config.Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = config.Clone(settings)
	r.videoClients = map[string]port.GenerationProvider{}
	r.imageClients = map[string]port.GenerationProvider{}
	return r.rebuildLocked()
}

func (r *Router) rebuildLocked() error {
	var firstErr error
	load := func(profiles []config.GenProfile, dest map[string]port.GenerationProvider) {
		for _, p := range profiles {
			if !p.Enabled {
				continue
			}
			c, err := clientForProfile(p)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			dest[p.ID] = c
		}
	}
	load(config.GenerationProfiles(r.settings), r.videoClients)
	load(config.ImageProfiles(r.settings), r.imageClients)
	if len(r.videoClients) == 0 && len(r.imageClients) == 0 {
		// Legacy single-provider env / settings without profiles yet.
		if c, err := resolveSingle(r.settings); err == nil && c != nil {
			r.videoClients[c.ID()] = c
			r.imageClients[c.ID()] = c
			return nil
		}
		// Empty provider lists are valid (settings UI before user adds any).
		return nil
	}
	if firstErr != nil && len(r.videoClients) == 0 && len(r.imageClients) == 0 {
		return firstErr
	}
	return nil
}

func clientForProfile(p config.GenProfile) (port.GenerationProvider, error) {
	return clientForVendor(config.GenVendor{
		ID:      p.ID,
		Vendor:  config.NormalizeGenVendorID(p.Vendor),
		Enabled: p.Enabled,
		BaseURL: p.BaseURL,
		APIKey:  p.APIKey,
		Models:  p.Models,
	})
}

func clientForVendor(v config.GenVendor) (port.GenerationProvider, error) {
	defaultModel := ""
	for _, m := range v.Models {
		if m.Enabled && m.ID != "" {
			defaultModel = m.ID
			break
		}
	}
	kind := config.NormalizeImageVendorID(v.Vendor)
	if kind == "" || (!config.IsKnownImageVendor(kind) && !config.IsKnownGenVendor(kind)) {
		kind = config.NormalizeGenVendorID(v.Vendor)
	}
	if kind == "" {
		kind = config.NormalizeGenVendorID(v.ID)
	}
	switch kind {
	case "danmo-make":
		return NewMake(
			config.FirstNonEmpty(os.Getenv("FILM_MAKE_BASE_URL"), v.BaseURL, "http://127.0.0.1:7800"),
			config.FirstNonEmpty(os.Getenv("FILM_MAKE_API_KEY"), v.APIKey),
		), nil
	case "volcengine":
		return NewVolcengine(
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_BASE_URL"), v.BaseURL),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_API_KEY"), v.APIKey),
			config.FirstNonEmpty(os.Getenv("FILM_VOLC_MODEL"), defaultModel),
		), nil
	case "dashscope":
		return NewDashScope(
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_BASE_URL"), v.BaseURL),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_API_KEY"), v.APIKey),
			config.FirstNonEmpty(os.Getenv("FILM_DASHSCOPE_MODEL"), defaultModel),
		), nil
	case "minimax":
		return NewMiniMax(
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_BASE_URL"), v.BaseURL),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_API_KEY"), v.APIKey),
			config.FirstNonEmpty(os.Getenv("FILM_MINIMAX_MODEL"), defaultModel),
		), nil
	case "openai":
		return NewOpenAIImage(
			config.FirstNonEmpty(os.Getenv("FILM_OPENAI_IMAGE_BASE_URL"), v.BaseURL, "https://api.openai.com/v1"),
			config.FirstNonEmpty(os.Getenv("FILM_OPENAI_IMAGE_API_KEY"), v.APIKey),
			config.FirstNonEmpty(os.Getenv("FILM_OPENAI_IMAGE_MODEL"), defaultModel),
		), nil
	default:
		return nil, fmt.Errorf("unknown generation provider %q", kind)
	}
}

// ResolveModel picks the video profile client for a profile_id/model_id (or default).
func (r *Router) ResolveModel(ref string) (port.GenerationProvider, string, error) {
	c, bare, _, err := r.ResolveVideoModel(ref)
	return c, bare, err
}

// ResolveVideoModel returns the video-profile client, bare model id, and profile id.
func (r *Router) ResolveVideoModel(ref string) (port.GenerationProvider, string, string, error) {
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	vendor, bare, err := config.FindGenVendorForModel(settings, ref)
	if err != nil {
		return nil, "", "", err
	}
	r.mu.RLock()
	c := r.videoClients[vendor.ID]
	r.mu.RUnlock()
	if c == nil {
		return nil, "", "", fmt.Errorf("generation provider %q is not loaded", vendor.ID)
	}
	return c, bare, vendor.ID, nil
}

// ResolveImageModel returns the image-profile client, bare model id, and profile id.
func (r *Router) ResolveImageModel(ref string) (port.GenerationProvider, string, string, error) {
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	vendor, bare, err := config.FindImageVendorForModel(settings, ref)
	if err != nil {
		return nil, "", "", err
	}
	r.mu.RLock()
	c := r.imageClients[vendor.ID]
	r.mu.RUnlock()
	if c == nil {
		return nil, "", "", fmt.Errorf("image provider %q is not loaded", vendor.ID)
	}
	return c, bare, vendor.ID, nil
}

func (r *Router) clientByID(id string) (port.GenerationProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c := r.videoClients[id]; c != nil {
		return c, nil
	}
	if c := r.imageClients[id]; c != nil {
		return c, nil
	}
	return nil, fmt.Errorf("generation provider %q is not loaded", id)
}

func (r *Router) Health(ctx context.Context) (map[string]any, error) {
	r.mu.RLock()
	clients := make(map[string]port.GenerationProvider)
	for k, v := range r.videoClients {
		clients["video:"+k] = v
	}
	for k, v := range r.imageClients {
		clients["image:"+k] = v
	}
	r.mu.RUnlock()
	out := map[string]any{"ok": true, "providers": map[string]any{}}
	providers := out["providers"].(map[string]any)
	allOK := true
	for id, c := range clients {
		h, err := c.Health(ctx)
		if err != nil {
			allOK = false
			providers[id] = map[string]any{"ok": false, "error": err.Error()}
			continue
		}
		providers[id] = h
		if ok, _ := h["ok"].(bool); !ok {
			allOK = false
		}
	}
	out["ok"] = allOK
	return out, nil
}

func (r *Router) listEnabledRows(ctx context.Context, enabled []config.EnabledGenModel, media string) ([]map[string]any, error) {
	var out []map[string]any
	for _, m := range enabled {
		c, err := r.clientByID(m.ProviderID)
		if err != nil {
			continue
		}
		items, err := c.ListModels(ctx, media)
		if err != nil {
			continue
		}
		for _, row := range items {
			bare := strings.TrimSpace(fmt.Sprint(row["id"]))
			if bare != "" && bare != m.ID {
				continue
			}
			cp := map[string]any{}
			for k, v := range row {
				cp[k] = v
			}
			cp["id"] = m.Ref
			cp["provider_id"] = m.ProviderID
			cp["provider_label"] = m.ProviderLabel
			if m.Vendor != "" {
				cp["vendor"] = m.Vendor
			}
			if media != "" {
				cp["media"] = media
			}
			if strings.TrimSpace(fmt.Sprint(cp["label"])) == "" || fmt.Sprint(cp["label"]) == bare {
				cp["label"] = m.Label
			}
			out = append(out, cp)
		}
		found := false
		for _, row := range out {
			if fmt.Sprint(row["id"]) == m.Ref {
				found = true
				break
			}
		}
		if !found && (media == "" || media == "video" || media == "image") {
			med := media
			if med == "" {
				med = "video"
			}
			row := map[string]any{
				"id":             m.Ref,
				"media":          med,
				"label":          m.Label,
				"provider_id":    m.ProviderID,
				"provider_label": m.ProviderLabel,
			}
			if m.Vendor != "" {
				row["vendor"] = m.Vendor
			}
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *Router) ListModels(ctx context.Context, media string) ([]map[string]any, error) {
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	switch media {
	case "image":
		return r.listEnabledRows(ctx, config.ListEnabledImageModels(settings), "image")
	case "video":
		return r.listEnabledRows(ctx, config.ListEnabledGenModels(settings), "video")
	default:
		vid, _ := r.listEnabledRows(ctx, config.ListEnabledGenModels(settings), "video")
		img, _ := r.listEnabledRows(ctx, config.ListEnabledImageModels(settings), "image")
		return append(vid, img...), nil
	}
}

// ListModelsFor lists models for one profile (settings refresh). Ids are bare.
// media may be empty — then the provider returns its full list (no type filter).
func (r *Router) ListModelsFor(ctx context.Context, profileID, media string) ([]map[string]any, error) {
	profileID = strings.TrimSpace(profileID)
	c, err := r.clientByID(profileID)
	if err != nil {
		r.mu.RLock()
		settings := r.settings
		r.mu.RUnlock()
		var p config.GenProfile
		var ok bool
		// Prefer the profile registry that actually owns this id (image or video).
		if media == "image" || media == "" {
			p, ok = config.FindImageProfile(settings, profileID)
		}
		if !ok {
			p, ok = config.FindGenProfile(settings, profileID)
		}
		if !ok && media != "image" {
			p, ok = config.FindImageProfile(settings, profileID)
		}
		if !ok {
			vendor := config.NormalizeImageVendorID(profileID)
			if media != "image" {
				vendor = config.NormalizeGenVendorID(profileID)
			}
			if config.IsKnownImageVendor(vendor) || config.IsKnownGenVendor(vendor) {
				p = config.GenProfile{
					ID: profileID, Vendor: vendor,
					BaseURL: config.FirstNonEmpty(
						config.ImageVendorDefaultBaseURL(vendor),
						config.GenVendorDefaultBaseURL(vendor),
					),
					Enabled: true,
				}
				ok = true
			}
		}
		if !ok {
			return nil, err
		}
		c2, err2 := clientForProfile(p)
		if err2 != nil {
			return nil, err
		}
		c = c2
	}
	return c.ListModels(ctx, media)
}

func (r *Router) GetRegistry(ctx context.Context) (map[string]any, error) {
	items, err := r.ListModels(ctx, "")
	if err != nil {
		return map[string]any{"models": map[string]any{}}, nil
	}
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	vidParams := config.VideoModelParamsMap(settings)
	imgParams := config.ImageModelParamsMap(settings)
	models := map[string]any{}
	for _, row := range items {
		id := strings.TrimSpace(fmt.Sprint(row["id"]))
		if id == "" {
			continue
		}
		// Strip profile prefix from registry keys that use profile/model.
		bare := id
		if i := strings.LastIndex(id, "/"); i >= 0 && i+1 < len(id) {
			bare = id[i+1:]
		}
		media := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["media"])))
		out := row
		if vp, ok := vidParams[bare]; ok && (media == "" || media == "video") {
			out = config.ApplyVideoParamsToModelRow(out, vp)
		}
		if ip, ok := imgParams[bare]; ok && (media == "" || media == "image") {
			out = config.ApplyImageParamsToModelRow(out, ip)
		}
		models[id] = out
	}
	// Also expose catalog-only models (configured but not returned by live list).
	for id, vp := range vidParams {
		ref := id
		if _, exists := models[ref]; exists {
			continue
		}
		models[ref] = config.ApplyVideoParamsToModelRow(map[string]any{
			"id": id, "label": id, "media": "video",
		}, vp)
	}
	for id, ip := range imgParams {
		if _, exists := models[id]; exists {
			continue
		}
		models[id] = config.ApplyImageParamsToModelRow(map[string]any{
			"id": id, "label": id, "media": "image",
		}, ip)
	}
	return map[string]any{"models": models}, nil
}

func (r *Router) ListAdapters(ctx context.Context, forModel string) (any, error) {
	c, bare, err := r.ResolveModel(forModel)
	if err != nil {
		return map[string]any{"items": []any{}}, nil
	}
	return c.ListAdapters(ctx, bare)
}

func (r *Router) CreateImage(ctx context.Context, body map[string]any) (map[string]any, error) {
	ref := strings.TrimSpace(fmt.Sprint(body["model"]))
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	vendor, bare, err := config.FindImageVendorForModel(settings, ref)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	c := r.imageClients[vendor.ID]
	r.mu.RUnlock()
	if c == nil {
		return nil, fmt.Errorf("image provider %q is not loaded", vendor.ID)
	}
	cp := cloneMap(body)
	cp["model"] = bare
	imgParams := config.ResolveImageModelParams(settings, bare)
	req, perr := ParseImageRequest(cp)
	if perr == nil {
		if err := imgParams.AssertSupportsImageMode(string(req.Mode)); err != nil {
			return nil, err
		}
	}
	if imgParams.Protocol != "" {
		cp["film_protocol"] = imgParams.Protocol
	}
	if len(imgParams.Modes) > 0 {
		cp["film_modes"] = imgParams.Modes
	}
	out, err := c.CreateImage(ctx, cp)
	if err != nil {
		return nil, err
	}
	return prefixTaskID(out, vendor.ID), nil
}

func (r *Router) CreateVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return r.dispatchVideo(ctx, body, func(c port.GenerationProvider, b map[string]any) (map[string]any, error) {
		return c.CreateVideo(ctx, b)
	})
}

func (r *Router) EditVideo(ctx context.Context, body map[string]any) (map[string]any, error) {
	return r.dispatchVideo(ctx, body, func(c port.GenerationProvider, b map[string]any) (map[string]any, error) {
		return c.EditVideo(ctx, b)
	})
}

func (r *Router) dispatchVideo(
	ctx context.Context,
	body map[string]any,
	fn func(port.GenerationProvider, map[string]any) (map[string]any, error),
) (map[string]any, error) {
	ref := strings.TrimSpace(fmt.Sprint(body["model"]))
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()
	vendor, bare, err := config.FindGenVendorForModel(settings, ref)
	if err != nil {
		return nil, err
	}
	r.mu.RLock()
	c := r.videoClients[vendor.ID]
	r.mu.RUnlock()
	if c == nil {
		return nil, fmt.Errorf("generation provider %q is not loaded", vendor.ID)
	}
	cp := cloneMap(body)
	cp["model"] = bare
	vidParams := config.ResolveVideoModelParams(settings, bare)
	req, perr := ParseVideoRequest(cp)
	if perr == nil {
		if err := vidParams.AssertSupportsVideoMode(string(req.Mode)); err != nil {
			return nil, err
		}
	}
	switch {
	case strings.TrimSpace(vidParams.Protocol) != "":
		cp["film_protocol"] = vidParams.Protocol
	case config.NormalizeGenVendorID(vendor.Vendor) == "danmo-make":
		cp["film_protocol"] = config.VideoProtocolMakeNative
	case !config.HasConfiguredVideoParams(settings, bare):
		return nil, fmt.Errorf("video model %q has no providers.video_model_params entry; add modes/protocol in settings", bare)
	}
	if len(vidParams.Modes) > 0 {
		cp["film_modes"] = vidParams.Modes
	}
	if vidParams.V2VSlot != "" {
		cp["film_v2v_slot"] = vidParams.V2VSlot
	}
	out, err := fn(c, cp)
	if err != nil {
		return nil, err
	}
	return prefixTaskID(out, vendor.ID), nil
}

func (r *Router) GetTask(ctx context.Context, taskID string) (map[string]any, error) {
	providerID, native, err := splitTaskID(taskID)
	if err != nil {
		return nil, err
	}
	c, err := r.clientByID(providerID)
	if err != nil {
		return nil, err
	}
	out, err := c.GetTask(ctx, native)
	if err != nil {
		return nil, err
	}
	return prefixTaskID(out, providerID), nil
}

func (r *Router) DownloadResult(ctx context.Context, providerRef, dest string) error {
	r.mu.RLock()
	clients := make([]port.GenerationProvider, 0, len(r.videoClients)+len(r.imageClients))
	for _, c := range r.videoClients {
		clients = append(clients, c)
	}
	for _, c := range r.imageClients {
		clients = append(clients, c)
	}
	r.mu.RUnlock()
	var last error
	for _, c := range clients {
		if err := c.DownloadResult(ctx, providerRef, dest); err == nil {
			return nil
		} else {
			last = err
		}
	}
	if last != nil {
		return last
	}
	return fmt.Errorf("download failed for %q", providerRef)
}

func (r *Router) StageInput(ctx context.Context, path, mime string) (map[string]any, error) {
	// Prefer video default, then image default — staging before create uses the same vendor.
	r.mu.RLock()
	videoRef := config.DefaultGenerationModelID(r.settings)
	imageRef := config.DefaultImageModelID(r.settings)
	r.mu.RUnlock()
	if c, _, _, err := r.ResolveVideoModel(videoRef); err == nil {
		return c.StageInput(ctx, path, mime)
	}
	if c, _, _, err := r.ResolveImageModel(imageRef); err == nil {
		return c.StageInput(ctx, path, mime)
	}
	return nil, fmt.Errorf("stage input: no generation/image provider loaded (pass provider_id/model_id on create)")
}

func (r *Router) IterTaskSSE(ctx context.Context, taskID string) (<-chan port.SSEEvent, error) {
	providerID, native, err := splitTaskID(taskID)
	if err != nil {
		return nil, err
	}
	c, err := r.clientByID(providerID)
	if err != nil {
		return nil, err
	}
	ch, err := c.IterTaskSSE(ctx, native)
	if err != nil {
		return nil, err
	}
	out := make(chan port.SSEEvent)
	go func() {
		defer close(out)
		for ev := range ch {
			out <- ev
		}
	}()
	return out, nil
}

func prefixTaskID(out map[string]any, providerID string) map[string]any {
	if out == nil || strings.TrimSpace(providerID) == "" {
		return out
	}
	native := extractTaskNativeID(out)
	if native == "" || strings.Contains(native, taskIDSep) {
		return out
	}
	prefixed := providerID + taskIDSep + native
	cp := cloneMap(out)
	cp["id"] = prefixed
	if task, ok := cp["task"].(map[string]any); ok {
		t2 := cloneMap(task)
		t2["id"] = prefixed
		cp["task"] = t2
	}
	return cp
}

func extractTaskNativeID(out map[string]any) string {
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

func splitTaskID(taskID string) (providerID, native string, err error) {
	taskID = strings.TrimSpace(taskID)
	i := strings.Index(taskID, taskIDSep)
	if i <= 0 || i >= len(taskID)-1 {
		return "", "", fmt.Errorf("task id must be provider_id:native_id, got %q", taskID)
	}
	// Left side is opaque profile id — do not normalize to vendor kind.
	providerID = strings.TrimSpace(taskID[:i])
	native = taskID[i+1:]
	if providerID == "" || native == "" {
		return "", "", fmt.Errorf("task id must be provider_id:native_id, got %q", taskID)
	}
	return providerID, native, nil
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
