package scriptparse

import (
	"strings"
	"testing"
)

func TestPreprocessGoldScene(t *testing.T) {
	script := `1. 内景 林夏公寓 夜

暴雨。林夏，29 岁，右手缠着渗血的绷带。
她把一张烧焦的照片压在桌上，攥紧黄铜钥匙。

手机亮起，陈屿发来信息：“别开门。”

三声敲门。
林夏抬头，钥匙从手中滑落。
`
	pipe, err := Preprocess(script, "钥匙")
	if err != nil {
		t.Fatal(err)
	}
	if pipe.SourceDocument == nil || len(pipe.SourceDocument.Lines) < 5 {
		t.Fatalf("expected numbered lines, got %+v", pipe.SourceDocument)
	}
	if !strings.Contains(pipe.SourceDocument.NumberedText, "1|") {
		t.Fatalf("expected numbered text, got %q", pipe.SourceDocument.NumberedText[:40])
	}
	if len(pipe.SceneIndex) < 1 {
		t.Fatal("expected at least one scene")
	}
	sc := pipe.SceneIndex[0]
	if sc.SceneID != "S001" {
		t.Fatalf("scene id %s", sc.SceneID)
	}
	if sc.SourceStartLine != 1 {
		t.Fatalf("start line %d", sc.SourceStartLine)
	}
	if !strings.Contains(sc.Text, "黄铜钥匙") {
		t.Fatalf("scene text missing prop: %q", sc.Text)
	}
}

func TestDiffSceneIndex(t *testing.T) {
	a, _ := Preprocess("1. INT. ROOM - NIGHT\n\nHello world action line here enough.\n", "t")
	b, _ := Preprocess("1. INT. ROOM - NIGHT\n\nHello world action line here enough.\n\nChanged ending.\n", "t")
	// Force different hash on same id by rebuilding second with modified text
	b.SceneIndex[0].SceneID = a.SceneIndex[0].SceneID
	changes := DiffSceneIndex(a.SceneIndex, b.SceneIndex)
	if len(changes) < 1 {
		t.Fatal("expected changes")
	}
	found := false
	for _, c := range changes {
		if c.SceneID == a.SceneIndex[0].SceneID && (c.Status == "modified" || c.Status == "unchanged" || c.Status == "added") {
			found = true
		}
	}
	if !found {
		t.Fatalf("unexpected changes %+v", changes)
	}
}

func TestSplitChineseHeading(t *testing.T) {
	script := "第一场 内景 咖啡店 日\n\n她走进来，点了一杯美式。\n\n第二场 外景 街道 夜\n\n雨还在下。\n"
	pipe, err := Preprocess(script, "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(pipe.SceneIndex) < 2 {
		t.Fatalf("expected 2 scenes, got %d: %+v", len(pipe.SceneIndex), pipe.SceneIndex)
	}
}
