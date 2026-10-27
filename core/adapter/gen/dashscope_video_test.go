package gen

import (
	"strings"
	"testing"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

func TestBuildDashVideoInputModes(t *testing.T) {
	media := config.VideoProtocolDashMedia
	legacy := config.VideoProtocolDashImgURL

	i2v, err := buildDashVideoInput("wan3.0-video", port.VideoRequest{
		Mode: port.VideoModeI2V, Prompt: "p", FirstFrame: "first",
	}, media, "")
	if err != nil {
		t.Fatal(err)
	}
	m := i2v["media"].([]map[string]any)
	if len(m) != 1 || m[0]["type"] != "first_frame" {
		t.Fatalf("%+v", m)
	}

	flf, err := buildDashVideoInput("wan3.0-video", port.VideoRequest{
		Mode: port.VideoModeFLF, FirstFrame: "a", LastFrame: "b",
	}, media, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(flf["media"].([]map[string]any)) != 2 {
		t.Fatalf("%+v", flf["media"])
	}

	r2v, err := buildDashVideoInput("wan3.0-video", port.VideoRequest{
		Mode: port.VideoModeR2V, ReferenceImages: []string{"r1", "r2"},
	}, media, "")
	if err != nil {
		t.Fatal(err)
	}
	rm := r2v["media"].([]map[string]any)
	if len(rm) != 2 || rm[0]["type"] != "reference_image" {
		t.Fatalf("%+v", rm)
	}

	v2v, err := buildDashVideoInput("wan3.0-video", port.VideoRequest{
		Mode: port.VideoModeV2V, ReferenceVideos: []string{"vid"},
	}, media, "")
	if err != nil {
		t.Fatal(err)
	}
	vm := v2v["media"].([]map[string]any)
	if len(vm) != 1 || vm[0]["type"] != "reference_video" {
		t.Fatalf("%+v", vm)
	}

	clip, err := buildDashVideoInput("wan2.7-i2v", port.VideoRequest{
		Mode: port.VideoModeV2V, ReferenceVideos: []string{"vid"},
	}, media, "first_clip")
	if err != nil {
		t.Fatal(err)
	}
	cm := clip["media"].([]map[string]any)
	if cm[0]["type"] != "first_clip" {
		t.Fatalf("wan2.7 v2v want first_clip, got %+v", cm)
	}

	imgURL, err := buildDashVideoInput("wan2.6-i2v", port.VideoRequest{
		Mode: port.VideoModeI2V, FirstFrame: "https://x/a.png",
	}, legacy, "")
	if err != nil {
		t.Fatal(err)
	}
	if imgURL["img_url"] != "https://x/a.png" {
		t.Fatalf("%+v", imgURL)
	}

	_, err = buildDashVideoInput("wan2.6-i2v", port.VideoRequest{
		Mode: port.VideoModeR2V, ReferenceImages: []string{"r"},
	}, legacy, "")
	if err == nil || !strings.Contains(err.Error(), "dashscope_media") {
		t.Fatalf("expected protocol error, got %v", err)
	}
}

func TestDashVideoResolution(t *testing.T) {
	if got := dashVideoResolution("1280x720"); got != "720P" {
		t.Fatalf("got %q", got)
	}
}
