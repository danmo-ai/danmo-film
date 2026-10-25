package ingest

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
	"github.com/danmo-ai/danmo-film/core/store/sqlite"
)

var singleAssetKeys = []string{
	"source_asset_id", "control_asset_id", "mask_asset_id",
	"start_asset_id", "end_asset_id", "tail_asset_id",
	"last_frame_asset_id", "end_frame_asset_id",
	"keyframe_asset_id", "segment_asset_id", "reference_asset_id",
	"grounding_panorama_asset_id", "grounding_depth_asset_id", "final_asset_id",
}

func isSingleAssetKey(key string) bool {
	for _, k := range singleAssetKeys {
		if k == key {
			return true
		}
	}
	return strings.HasSuffix(key, "_asset_id")
}

func GuessKind(mimeType, fallback string) string {
	mt := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(mt, "image/"):
		return "image"
	case strings.HasPrefix(mt, "audio/"):
		return "audio"
	case strings.HasPrefix(mt, "video/"):
		return "video"
	default:
		if fallback == "" {
			return "video"
		}
		return fallback
	}
}

func IngestProviderAsset(ctx context.Context, gen port.GenerationProvider, store port.AssetStore, providerAssetID, kind, mimeType, sourceTaskID string, metadata map[string]any) (*port.FilmAsset, error) {
	ref := strings.TrimSpace(providerAssetID)
	if ref == "" {
		return nil, fmt.Errorf("provider_asset_id is required")
	}
	if sqlite.IsFilmAssetID(ref) {
		existing, err := store.Get(ref)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, fmt.Errorf("Film asset not found: %s", ref)
		}
		return existing, nil
	}
	cached, err := store.FindByProviderRef(gen.ID(), ref)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}
	suffix := ".mp4"
	if kind == "image" {
		suffix = ".png"
	}
	tmp, err := os.CreateTemp("", "film_ingest_*"+suffix)
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if err := gen.DownloadResult(ctx, ref, tmp.Name()); err != nil {
		return nil, err
	}
	mt := mimeType
	if mt == "" {
		mt = mime.TypeByExtension(filepath.Ext(tmp.Name()))
	}
	if mt == "" {
		if kind == "image" {
			mt = "image/png"
		} else {
			mt = "video/mp4"
		}
	}
	return store.CreateFromFile(tmp.Name(), port.CreateAssetOpts{
		Kind: kind, MimeType: mt, SourceProvider: gen.ID(), ProviderRef: ref,
		SourceTaskID: sourceTaskID, Metadata: metadata,
	})
}

func IngestTaskAssetIDs(ctx context.Context, gen port.GenerationProvider, store port.AssetStore, task map[string]any, defaultKind string) map[string]any {
	out := cloneMap(task)
	status := strings.ToLower(str(out["status"]))
	rawIDs, _ := out["asset_ids"].([]any)
	if len(rawIDs) == 0 {
		if ss, ok := out["asset_ids"].([]string); ok {
			for _, s := range ss {
				rawIDs = append(rawIDs, s)
			}
		}
	}
	if len(rawIDs) == 0 {
		return out
	}
	switch status {
	case "queued", "pending", "running", "cancelled", "canceled", "failed", "error":
		return out
	}
	kind := defaultKind
	if kind == "" {
		kind = "video"
	}
	tk := strings.ToLower(str(out["kind"]) + str(out["task_kind"]))
	if strings.Contains(tk, "image") {
		kind = "image"
	} else if strings.Contains(tk, "audio") {
		kind = "audio"
	}
	tid := str(out["id"])
	if tid == "" {
		tid = str(out["task_id"])
	}
	var filmIDs []string
	var providerIDs []string
	for _, rid := range rawIDs {
		aid := strings.TrimSpace(fmt.Sprint(rid))
		if aid == "" {
			continue
		}
		row, err := IngestProviderAsset(ctx, gen, store, aid, kind, "", tid, nil)
		if err != nil {
			out["ingest_error"] = err.Error()
			return out
		}
		filmIDs = append(filmIDs, row.ID)
		providerIDs = append(providerIDs, aid)
	}
	ids := make([]any, len(filmIDs))
	for i, id := range filmIDs {
		ids[i] = id
	}
	out["asset_ids"] = ids
	out["provider_asset_ids"] = providerIDs
	return out
}

func StageFilmAsset(ctx context.Context, gen port.GenerationProvider, store port.AssetStore, filmAssetID string) (string, error) {
	path, err := store.ResolvePath(filmAssetID)
	if err != nil {
		return "", err
	}
	row, err := store.Get(filmAssetID)
	if err != nil {
		return "", err
	}
	mimeType := "application/octet-stream"
	if row != nil && row.MimeType != "" {
		mimeType = row.MimeType
	}
	staged, err := gen.StageInput(ctx, path, mimeType)
	if err != nil {
		return "", err
	}
	pid := str(staged["id"])
	if pid == "" {
		pid = str(staged["asset_id"])
	}
	if pid == "" {
		return "", fmt.Errorf("provider stage_input_file returned no id: %v", staged)
	}
	return pid, nil
}

func RewriteBodyAssetRefs(ctx context.Context, body map[string]any, gen port.GenerationProvider, store port.AssetStore) (map[string]any, error) {
	out := cloneMap(body)
	staged, err := stageAssetRefsDeep(ctx, out, gen, store)
	if err != nil {
		return nil, err
	}
	return staged.(map[string]any), nil
}

func stageAssetRefsDeep(ctx context.Context, v any, gen port.GenerationProvider, store port.AssetStore) (any, error) {
	switch x := v.(type) {
	case map[string]any:
		out := cloneMap(x)
		for key, val := range out {
			if isSingleAssetKey(key) {
				s := strings.TrimSpace(fmt.Sprint(val))
				if sqlite.IsFilmAssetID(s) {
					pid, err := StageFilmAsset(ctx, gen, store, s)
					if err != nil {
						return nil, err
					}
					out[key] = pid
					continue
				}
			}
			if key == "reference_asset_ids" {
				if rewritten, err := stageReferenceAssetIDs(ctx, val, gen, store); err != nil {
					return nil, err
				} else {
					out[key] = rewritten
					continue
				}
			}
			next, err := stageAssetRefsDeep(ctx, val, gen, store)
			if err != nil {
				return nil, err
			}
			out[key] = next
		}
		return out, nil
	case []any:
		arr := make([]any, len(x))
		for i, item := range x {
			next, err := stageAssetRefsDeep(ctx, item, gen, store)
			if err != nil {
				return nil, err
			}
			arr[i] = next
		}
		return arr, nil
	default:
		return v, nil
	}
}

func stageReferenceAssetIDs(ctx context.Context, val any, gen port.GenerationProvider, store port.AssetStore) ([]any, error) {
	raw, ok := val.([]any)
	if !ok {
		if ss, ok := val.([]string); ok {
			for _, s := range ss {
				raw = append(raw, s)
			}
		}
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var rewritten []any
	for _, item := range raw {
		s := strings.TrimSpace(fmt.Sprint(item))
		if s == "" {
			continue
		}
		if sqlite.IsFilmAssetID(s) {
			pid, err := StageFilmAsset(ctx, gen, store, s)
			if err != nil {
				return nil, err
			}
			rewritten = append(rewritten, pid)
		} else {
			rewritten = append(rewritten, s)
		}
	}
	return rewritten, nil
}

func MaterializeSegmentPath(ctx context.Context, assetID, dest string, gen port.GenerationProvider, store port.AssetStore) error {
	aid := strings.TrimSpace(assetID)
	if aid == "" {
		return fmt.Errorf("empty segment asset id")
	}
	if sqlite.IsFilmAssetID(aid) {
		src, err := store.ResolvePath(aid)
		if err != nil {
			return err
		}
		return copyFile(src, dest)
	}
	return gen.DownloadResult(ctx, aid, dest)
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
