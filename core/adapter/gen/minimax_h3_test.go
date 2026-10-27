package gen

import (
	"strings"
	"testing"

	"github.com/danmo-ai/danmo-film/core/port"
)

func TestBuildMiniMaxH3VideoPayloadModes(t *testing.T) {
	resolve := func(s string) string { return "data:image/png;base64," + s }

	t2v, err := buildMiniMaxH3VideoPayload("MiniMax-H3", port.VideoRequest{
		Mode: port.VideoModeT2V, Prompt: "dance",
	}, resolve)
	if err != nil {
		t.Fatal(err)
	}
	c := t2v["content"].([]map[string]any)
	if len(c) != 1 || c[0]["type"] != "text" {
		t.Fatalf("%+v", c)
	}

	i2v, err := buildMiniMaxH3VideoPayload("MiniMax-H3", port.VideoRequest{
		Mode: port.VideoModeI2V, Prompt: "p", FirstFrame: "f1",
	}, resolve)
	if err != nil {
		t.Fatal(err)
	}
	ic := i2v["content"].([]map[string]any)
	if len(ic) != 2 || ic[1]["role"] != "first_frame" {
		t.Fatalf("%+v", ic)
	}

	r2v, err := buildMiniMaxH3VideoPayload("MiniMax-H3", port.VideoRequest{
		Mode: port.VideoModeR2V, Prompt: "p", ReferenceImages: []string{"a", "b"},
	}, resolve)
	if err != nil {
		t.Fatal(err)
	}
	rc := r2v["content"].([]map[string]any)
	if len(rc) != 3 || rc[1]["role"] != "reference_image" {
		t.Fatalf("%+v", rc)
	}

	v2v, err := buildMiniMaxH3VideoPayload("MiniMax-H3", port.VideoRequest{
		Mode: port.VideoModeV2V, Prompt: "p", ReferenceVideos: []string{"vid"},
	}, resolve)
	if err != nil {
		t.Fatal(err)
	}
	vc := v2v["content"].([]map[string]any)
	if vc[1]["role"] != "reference_video" {
		t.Fatalf("%+v", vc)
	}
}

func TestAssertVideoModeSupported(t *testing.T) {
	cfg := map[string]any{"modes": []string{"t2v", "i2v", "flf", "r2v", "v2v"}}
	if err := AssertVideoModeSupported(cfg, port.VideoModeR2V); err != nil {
		t.Fatal(err)
	}
	err := AssertVideoModeSupported(map[string]any{"modes": []string{"t2v", "i2v"}}, port.VideoModeR2V)
	if err == nil || !strings.Contains(err.Error(), "r2v") {
		t.Fatalf("got %v", err)
	}
}

func TestParseImageRequestModes(t *testing.T) {
	t2i, err := ParseImageRequest(map[string]any{"mode": "t2i", "prompt": "cat"})
	if err != nil || t2i.Mode != port.ImageModeT2I {
		t.Fatalf("%+v %v", t2i, err)
	}
	i2i, err := ParseImageRequest(map[string]any{
		"mode": "i2i", "prompt": "p", "reference_asset_ids": []any{"r1"},
	})
	if err != nil || i2i.Mode != port.ImageModeI2I {
		t.Fatalf("%+v %v", i2i, err)
	}
	// explicit t2i drops stray refs
	clean, err := ParseImageRequest(map[string]any{
		"mode": "t2i", "prompt": "p", "reference_asset_ids": []any{"r1"},
	})
	if err != nil || len(clean.ReferenceImages) != 0 {
		t.Fatalf("%+v %v", clean, err)
	}
}
