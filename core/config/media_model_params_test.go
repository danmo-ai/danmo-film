package config

import (
	"testing"
)

func TestSeedVideoModelParamsRoundTrip(t *testing.T) {
	seed := SeedVideoModelParamsFromPresets()
	wan27 := seed["wan2.7-i2v"]
	if wan27.Protocol != VideoProtocolDashMedia || wan27.V2VSlot != "first_clip" {
		t.Fatalf("wan2.7-i2v: %+v", wan27)
	}
	wan3 := seed["wan3.0-video"]
	if wan3.Protocol != VideoProtocolDashMedia || !wan3.SupportsVideoMode("r2v") {
		t.Fatalf("wan3.0-video: %+v", wan3)
	}
	h3 := seed["MiniMax-H3"]
	if h3.Protocol != VideoProtocolMiniMaxH3 || !h3.SupportsVideoMode("v2v") {
		t.Fatalf("MiniMax-H3: %+v", h3)
	}

	anyMap := VideoModelParamsToAny(seed)
	parsed := parseVideoModelParamsMap(anyMap)
	got := parsed["wan2.7-i2v"]
	if got.V2VSlot != "first_clip" || got.Protocol != VideoProtocolDashMedia {
		t.Fatalf("round-trip wan2.7: %+v", got)
	}
}

func TestHydrateSeedsMediaModelParams(t *testing.T) {
	s := Hydrate(Settings{
		"providers": map[string]any{},
	})
	vid := VideoModelParamsMap(s)
	if vid["wan3.0-video"].Protocol == "" {
		t.Fatal("expected wan3.0-video seed")
	}
	img := ImageModelParamsMap(s)
	if len(img["qwen-image"].Modes) == 0 {
		t.Fatal("expected qwen-image seed")
	}
}

func TestAssertSupportsVideoModeFailLoud(t *testing.T) {
	p := VideoModelParams{Modes: []string{"t2v", "i2v"}}
	if err := p.AssertSupportsVideoMode("r2v"); err == nil {
		t.Fatal("expected error")
	}
	if err := p.AssertSupportsVideoMode("i2v"); err != nil {
		t.Fatal(err)
	}
}
