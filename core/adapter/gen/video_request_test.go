package gen

import (
	"strings"
	"testing"

	"github.com/danmo-ai/danmo-film/core/port"
)

func TestChainModeToVideoMode(t *testing.T) {
	cases := map[string]port.VideoMode{
		"keyframe_only":  port.VideoModeI2V,
		"last_frame":     port.VideoModeI2V,
		"first_last":     port.VideoModeFLF,
		"reference_r2v":  port.VideoModeR2V,
		"reference_v2v":  port.VideoModeV2V,
		"v2v":            port.VideoModeV2V,
		"video_continue": port.VideoModeV2V,
		"t2v":            port.VideoModeT2V,
		"":               "",
	}
	for in, want := range cases {
		if got := ChainModeToVideoMode(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestParseVideoRequestR2VRejectsFirstFrameMix(t *testing.T) {
	_, err := ParseVideoRequest(map[string]any{
		"prompt":              "p",
		"source_asset_id":     "first",
		"reference_asset_ids": []any{"ref-a", "ref-b"},
	})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected exclusive mix error, got %v", err)
	}
}

func TestParseVideoRequestR2VFromSlots(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{
		"prompt":              "p",
		"reference_asset_ids": []any{"ref-a", "ref-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if DeriveVideoMode(req) != port.VideoModeR2V {
		t.Fatalf("kind %s", DeriveVideoMode(req))
	}
	if len(req.ReferenceImages) != 2 {
		t.Fatalf("refs %+v", req.ReferenceImages)
	}
	if req.FirstFrame != "" {
		t.Fatalf("unexpected first frame %q", req.FirstFrame)
	}
}

func TestParseVideoRequestI2VFromSlots(t *testing.T) {
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
	if DeriveVideoMode(req) != port.VideoModeI2V || req.FirstFrame != "kf" {
		t.Fatalf("%+v", req)
	}
}

func TestParseVideoRequestIgnoresBodyMode(t *testing.T) {
	// body.mode must not drop slots — conflicting media fails loud instead.
	_, err := ParseVideoRequest(map[string]any{
		"mode":                "r2v",
		"prompt":              "p",
		"source_asset_id":     "first",
		"reference_asset_ids": []any{"ref-a"},
	})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected exclusive mix error, got %v", err)
	}
}

func TestParseVideoRequestV2V(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{
		"prompt":                    "continue",
		"source_video_asset_id":     "vid1",
		"reference_video_asset_ids": []any{"vid2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if DeriveVideoMode(req) != port.VideoModeV2V {
		t.Fatalf("kind %s", DeriveVideoMode(req))
	}
	if len(req.ReferenceVideos) < 1 || req.ReferenceVideos[0] != "vid1" {
		t.Fatalf("videos %+v", req.ReferenceVideos)
	}
}

func TestParseVideoRequestT2V(t *testing.T) {
	req, err := ParseVideoRequest(map[string]any{"prompt": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if DeriveVideoMode(req) != port.VideoModeT2V {
		t.Fatalf("%s", DeriveVideoMode(req))
	}
}

func TestValidateVideoRequestLastWithoutFirst(t *testing.T) {
	err := ValidateVideoRequest(port.VideoRequest{
		LastFrame: "tail",
	})
	if err == nil || !strings.Contains(err.Error(), "requires first_frame") {
		t.Fatalf("expected last-without-first error, got %v", err)
	}
}

func TestValidateVideoRequestMixFail(t *testing.T) {
	err := ValidateVideoRequest(port.VideoRequest{
		FirstFrame:      "a",
		ReferenceImages: []string{"r"},
	})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mix error, got %v", err)
	}
}

func TestNormalizeVideoBodyStripsMode(t *testing.T) {
	out, req, err := NormalizeVideoBody(map[string]any{
		"mode":   "i2v",
		"prompt": "p",
		"source_asset_id": "kf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["mode"]; ok {
		t.Fatalf("mode should be stripped: %+v", out)
	}
	if DeriveVideoMode(req) != port.VideoModeI2V {
		t.Fatalf("%s", DeriveVideoMode(req))
	}
}
