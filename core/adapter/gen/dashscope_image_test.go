package gen

import "testing"

func TestBodyStringIDs(t *testing.T) {
	got := bodyStringIDs([]any{" a ", "", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected %v", got)
	}
}
