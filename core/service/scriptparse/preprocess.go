package scriptparse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	sceneHeadingRe = regexp.MustCompile(`(?i)^(?:\s*)((?:\d+[A-Za-z]?\.)?\s*)?(INT\.?|EXT\.?|内景|外景|内\/外|I\/E\.?)[\s.·／/]*(.+)$`)
	simpleNumScene = regexp.MustCompile(`^\s*(\d+[A-Za-z]?)\s*[.、．]\s*(.+)$`)
	chineseScene   = regexp.MustCompile(`^\s*(第?[一二三四五六七八九十百千\d]+场)\s*[.、．:\s]*(.+)$`)
)

// Preprocess builds a numbered source document and scene index without LLM.
func Preprocess(scriptText, title string) (*ScriptPipeline, error) {
	text := strings.ReplaceAll(scriptText, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	if len([]rune(text)) < 20 {
		return nil, fmt.Errorf("script_text is too short")
	}
	lines := strings.Split(text, "\n")
	numbered := make([]string, len(lines))
	for i, ln := range lines {
		numbered[i] = fmt.Sprintf("%d|%s", i+1, ln)
	}
	hash := hashText(text)
	scriptID := stableID("script_", hash)
	versionID := stableID("ver_", hash+fmt.Sprint(time.Now().UnixNano()))

	scenes := splitScenes(lines)
	if len(scenes) == 0 {
		// Whole script as one scene.
		scenes = []SceneIndexEntry{{
			SceneID:         "S001",
			Heading:         orUntitled(title),
			SourceStartLine: 1,
			SourceEndLine:   len(lines),
			SourceHash:      hashText(text),
			Text:            text,
			Synopsis:        "",
		}}
	}

	pipe := &ScriptPipeline{
		SchemaVersion: PipelineSchemaVersion,
		ScriptID:      scriptID,
		VersionID:     versionID,
		SourceHash:    hash,
		Title:         orUntitled(title),
		SourceDocument: &SourceDocument{
			Text:         text,
			NumberedText: strings.Join(numbered, "\n"),
			Lines:        lines,
			UploadedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		SceneIndex:     scenes,
		SceneArtifacts: map[string]*SceneArtifact{},
		Workflow: WorkflowState{
			Stage:  StageR0,
			Status: "completed",
		},
	}
	for _, sc := range scenes {
		pipe.SceneArtifacts[sc.SceneID] = &SceneArtifact{
			SceneID:      sc.SceneID,
			ReviewStatus: ReviewPending,
		}
	}
	return pipe, nil
}

func splitScenes(lines []string) []SceneIndexEntry {
	type marker struct {
		line    int
		heading string
		num     string
		intExt  string
		loc     string
		tod     string
	}
	var markers []marker
	for i, raw := range lines {
		ln := strings.TrimSpace(raw)
		if ln == "" {
			continue
		}
		if m := sceneHeadingRe.FindStringSubmatch(ln); m != nil {
			num := strings.TrimSpace(strings.Trim(m[1], ". "))
			intExt := normalizeIntExt(m[2])
			rest := strings.TrimSpace(m[3])
			loc, tod := splitLocTime(rest)
			markers = append(markers, marker{line: i + 1, heading: ln, num: num, intExt: intExt, loc: loc, tod: tod})
			continue
		}
		if m := chineseScene.FindStringSubmatch(ln); m != nil {
			markers = append(markers, marker{line: i + 1, heading: ln, num: m[1], loc: strings.TrimSpace(m[2])})
			continue
		}
		if m := simpleNumScene.FindStringSubmatch(ln); m != nil {
			rest := strings.TrimSpace(m[2])
			// Only treat as scene if rest looks like a heading (short, or has INT/EXT/内景).
			if looksLikeHeading(rest) {
				intExt, loc, tod := "", rest, ""
				if mm := sceneHeadingRe.FindStringSubmatch(rest); mm != nil {
					intExt = normalizeIntExt(mm[2])
					loc, tod = splitLocTime(strings.TrimSpace(mm[3]))
				}
				markers = append(markers, marker{line: i + 1, heading: ln, num: m[1], intExt: intExt, loc: loc, tod: tod})
			}
		}
	}
	if len(markers) == 0 {
		return nil
	}
	out := make([]SceneIndexEntry, 0, len(markers))
	for i, mk := range markers {
		end := len(lines)
		if i+1 < len(markers) {
			end = markers[i+1].line - 1
		}
		if end < mk.line {
			end = mk.line
		}
		bodyLines := lines[mk.line-1 : end]
		body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		sid := fmt.Sprintf("S%03d", i+1)
		out = append(out, SceneIndexEntry{
			SceneID:           sid,
			SourceSceneNumber: mk.num,
			Heading:           mk.heading,
			IntExt:            mk.intExt,
			LocationText:      mk.loc,
			StoryTime:         mk.tod,
			SourceStartLine:   mk.line,
			SourceEndLine:     end,
			SourceHash:        hashText(body),
			Text:              body,
		})
	}
	return out
}

func looksLikeHeading(s string) bool {
	if len([]rune(s)) > 80 {
		return false
	}
	u := strings.ToUpper(s)
	if strings.Contains(u, "INT") || strings.Contains(u, "EXT") ||
		strings.Contains(s, "内景") || strings.Contains(s, "外景") ||
		strings.Contains(s, "日") || strings.Contains(s, "夜") {
		return true
	}
	// All-caps short line often a slugline.
	letters := 0
	upper := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	return letters > 0 && upper*100/letters >= 70
}

func normalizeIntExt(raw string) string {
	u := strings.ToUpper(strings.TrimSpace(raw))
	u = strings.Trim(u, ".")
	switch {
	case strings.Contains(u, "I/E") || strings.Contains(raw, "内/外"):
		return "I/E"
	case strings.HasPrefix(u, "INT") || strings.Contains(raw, "内景"):
		return "INT"
	case strings.HasPrefix(u, "EXT") || strings.Contains(raw, "外景"):
		return "EXT"
	default:
		return u
	}
}

func splitLocTime(rest string) (loc, tod string) {
	parts := regexp.MustCompile(`[\s·\-—]+`).Split(rest, -1)
	if len(parts) == 0 {
		return rest, ""
	}
	last := strings.ToUpper(parts[len(parts)-1])
	timeWords := map[string]bool{
		"DAY": true, "NIGHT": true, "DAWN": true, "DUSK": true, "EVENING": true, "MORNING": true,
		"日": true, "夜": true, "晨": true, "晚": true, "黄昏": true, "清晨": true,
	}
	if timeWords[last] || timeWords[parts[len(parts)-1]] {
		return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " ")), parts[len(parts)-1]
	}
	return rest, ""
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

// DiffSceneIndex compares previous and current scene indexes for incremental reruns.
func DiffSceneIndex(prev, curr []SceneIndexEntry) []SceneChange {
	prevByHash := map[string]SceneIndexEntry{}
	prevByID := map[string]SceneIndexEntry{}
	for _, p := range prev {
		prevByHash[p.SourceHash] = p
		prevByID[p.SceneID] = p
	}
	usedPrev := map[string]bool{}
	var changes []SceneChange
	for _, c := range curr {
		if p, ok := prevByID[c.SceneID]; ok && p.SourceHash == c.SourceHash {
			changes = append(changes, SceneChange{SceneID: c.SceneID, Status: "unchanged"})
			usedPrev[p.SceneID] = true
			continue
		}
		if p, ok := prevByHash[c.SourceHash]; ok && !usedPrev[p.SceneID] {
			changes = append(changes, SceneChange{SceneID: c.SceneID, Status: "moved"})
			usedPrev[p.SceneID] = true
			continue
		}
		if _, ok := prevByID[c.SceneID]; ok {
			changes = append(changes, SceneChange{SceneID: c.SceneID, Status: "modified"})
			usedPrev[c.SceneID] = true
			continue
		}
		changes = append(changes, SceneChange{SceneID: c.SceneID, Status: "added"})
	}
	for _, p := range prev {
		if !usedPrev[p.SceneID] {
			changes = append(changes, SceneChange{SceneID: p.SceneID, Status: "removed"})
		}
	}
	return changes
}
