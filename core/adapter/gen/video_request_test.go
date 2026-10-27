package gen

import (
	"strings"
	"testing"

	"github.com/danmo-ai/danmo-film/core/port"
)

func TestChainModeToVideoMode(t *testing.T) {
	cases := map[string]port.VideoMode{
		"keyframe_only":   port.VideoModeI2V,
		"last_frame":      port.VideoModeI2V,
		"first_last":      port.VideoModeFLF,
		"reference_r2v":   port.VideoModeR2V,
		"reference_v2v":   port.VideoModeV2V,
		"v2v":             port.VideoModeV2V,
		"video_continue":  port.VideoModeV2V,
		"t2v":             port.VideoModeT2V,
		"":                "",
	}
	for in, want := range cases {
		if got := ChainModeToVideoMode(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestParseVideoRequestExplicitR2VDropsFirstFrame(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{
		"mode":                "r2v",
		"prompt":              "p",
		"source_asset_id":     "first",
		"reference_asset_ids": []any{"ref-a", "ref-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Mode != port.VideoModeR2V {
		t.Fatalf("mode %s", req.Mode)
	}
	if req.FirstFrame != "" {
		t.Fatalf("first frame should be dropped for explicit r2v, got %q", req.FirstFrame)
	}
	if len(req.ReferenceImages) != 2 {
		t.Fatalf("refs %+v", req.ReferenceImages)
	}
}

func TestParseVideoRequestFromChainMetadata(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{
		"prompt":          "p",
		"source_asset_id": "kf",
		"metadata": map[string]any{
			"long_video_chain_mode": "keyframe_only",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Mode != port.VideoModeI2V || req.FirstFrame != "kf" {
		t.Fatalf("%+v", req)
	}
}

func TestParseVideoRequestV2V(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{
		"mode":                     "v2v",
		"prompt":                   "continue",
		"source_video_asset_id":    "vid1",
		"reference_video_asset_ids": []any{"vid2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Mode != port.VideoModeV2V {
		t.Fatalf("mode %s", req.Mode)
	}
	if len(req.ReferenceVideos) < 1 || req.ReferenceVideos[0] != "vid1" {
		t.Fatalf("videos %+v", req.ReferenceVideos)
	}
}

func TestParseVideoRequestT2VRejectsStrayMediaWhenInferred(t *testing.T) {
	// No explicit mode + no media → t2v
	req, err := ParseVideoRequest(map[string]any{"prompt": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Mode != port.VideoModeT2V {
		t.Fatalf("%s", req.Mode)
	}
}

func TestValidateVideoRequestMixFailWhenInferredAmbiguous(t *testing.T) {
	err := ValidateVideoRequest(port.VideoRequest{
		Mode:            port.VideoModeI2V,
		FirstFrame:      "a",
		ReferenceImages: []string{"r"},
	})
	if err == nil || !strings.Contains(err.Error(), "cannot mix") {
		t.Fatalf("expected mix error, got %v", err)
	}
}
