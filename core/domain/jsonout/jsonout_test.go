package jsonout

import "testing"

func TestExtractObjectFence(t *testing.T) {
	m, err := ExtractObject("```json\n{\"a\": 1}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if m["a"].(float64) != 1 {
		t.Fatalf("got %#v", m)
	}
}

func TestExtractFinalThink(t *testing.T) {
	got := ExtractFinal("<think>plan</think>\n{\"ok\":true}")
	if got != "{\"ok\":true}" {
		t.Fatalf("got %q", got)
	}
}
