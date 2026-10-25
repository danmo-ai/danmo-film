package llm

import (
	"testing"

	"github.com/danmo-ai/danmo-film/core/config"
)

func TestApplyReasoningDialectRequest_DeepSeek(t *testing.T) {
	body := map[string]any{}
	applyReasoningDialectRequest(body, config.DialectDeepSeek, "max", nil)
	th, _ := body["thinking"].(map[string]any)
	if th["type"] != "enabled" || body["reasoning_effort"] != "max" {
		t.Fatalf("%+v", body)
	}
	bodyOff := map[string]any{}
	applyReasoningDialectRequest(bodyOff, config.DialectDeepSeek, "off", nil)
	thOff, _ := bodyOff["thinking"].(map[string]any)
	if thOff["type"] != "disabled" {
		t.Fatalf("%+v", bodyOff)
	}
}

func TestApplyReasoningDialectRequest_Qwen(t *testing.T) {
	body := map[string]any{}
	applyReasoningDialectRequest(body, config.DialectQwen, "high", nil)
	if body["enable_thinking"] != true {
		t.Fatalf("%+v", body)
	}
}

func TestResolveEffort(t *testing.T) {
	if resolveEffort(nil) != "off" {
		t.Fatal("nil")
	}
}