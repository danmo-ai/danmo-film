package scriptparse

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
)

func stableID(prefix, key string) string {
	h := sha1.Sum([]byte(key))
	return prefix + "_" + hex.EncodeToString(h[:])[:10]
}

func assignIDs(art map[string]any) {
	chars := asMaps(art["characters"])
	for i, ch := range chars {
		name := asString(ch["name"])
		if asString(ch["id"]) == "" {
			ch["id"] = stableID("char", name)
		}
		looks := asMaps(ch["looks"])
		for j, lk := range looks {
			if asString(lk["id"]) == "" {
				lk["id"] = stableID("look", fmt.Sprintf("%s|%s|%d", name, asString(lk["label"]), j))
			}
			looks[j] = lk
		}
		ch["looks"] = toAny(looks)
		if asString(ch["default_look_id"]) == "" && len(looks) > 0 {
			ch["default_look_id"] = looks[0]["id"]
		}
		chars[i] = ch
	}
	art["characters"] = toAny(chars)

	scenes := asMaps(art["scenes"])
	for i, sc := range scenes {
		name := asString(sc["name"])
		if asString(sc["id"]) == "" {
			sc["id"] = stableID("scene", name)
		}
		looks := asMaps(sc["looks"])
		for j, lk := range looks {
			if asString(lk["id"]) == "" {
				lk["id"] = stableID("scene_look", fmt.Sprintf("%s|%s|%d", name, asString(lk["label"]), j))
			}
			looks[j] = lk
		}
		sc["looks"] = toAny(looks)
		if asString(sc["default_look_id"]) == "" && len(looks) > 0 {
			sc["default_look_id"] = looks[0]["id"]
		}
		scenes[i] = sc
	}
	art["scenes"] = toAny(scenes)

	beats := asMaps(art["beats"])
	for i, b := range beats {
		b["index"] = i
		beats[i] = b
	}
	art["beats"] = toAny(beats)
}

func toAny(ms []map[string]any) []any {
	out := make([]any, len(ms))
	for i, m := range ms {
		out[i] = m
	}
	return out
}

func splitDurationParts(total, cap float64) []float64 {
	if total <= cap {
		return []float64{math.Round(total*10) / 10}
	}
	n := int(math.Ceil(total / cap))
	base := math.Round((total/float64(n))*10) / 10
	parts := make([]float64, n)
	for i := 0; i < n-1; i++ {
		parts[i] = base
	}
	parts[n-1] = math.Round((total-base*float64(n-1))*10) / 10
	for i := range parts {
		if parts[i] < 2 {
			parts[i] = 2
		}
	}
	return parts
}

func sanitizeRole(raw string) string {
	t := strings.TrimSpace(raw)
	valid := map[string]bool{"establishing": true, "pre_anchor": true, "face_anchor": true, "post_anchor": true, "keyframe": true, "tail_continuation": true}
	if valid[t] {
		return t
	}
	aliases := map[string]string{"wide": "keyframe", "action": "keyframe", "anchor": "face_anchor", "face": "face_anchor", "establish": "establishing", "continuation": "tail_continuation"}
	if v, ok := aliases[t]; ok {
		return v
	}
	return "keyframe"
}

// sanitizeCastRole normalizes story-map character roles (not segment roles).
func sanitizeCastRole(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	switch t {
	case "protagonist", "lead", "main", "hero", "主角":
		return "protagonist"
	case "supporting", "support", "secondary", "配角":
		return "supporting"
	case "extra", "bit", "background", "龙套", "群演":
		return "extra"
	case "antagonist", "villain", "反派":
		return "supporting"
	}
	if t == "" {
		return "supporting"
	}
	return "supporting"
}

func sanitizeVis(raw, def string) string {
	t := strings.TrimSpace(raw)
	valid := map[string]bool{"invisible": true, "silhouette": true, "partial": true, "full_face": true}
	if valid[t] {
		return t
	}
	aliases := map[string]string{"visible": "full_face", "full": "full_face", "face": "full_face", "hidden": "invisible", "none": "invisible"}
	if v, ok := aliases[t]; ok {
		return v
	}
	return def
}

func startFrameMode(role, prev string) string {
	if role == "tail_continuation" {
		return "prev_segment_tail"
	}
	if role == "post_anchor" && prev == "face_anchor" {
		return "anchor_link"
	}
	return "keyframe"
}

func formatRoster(chars []map[string]any, style, locale string) string {
	var b strings.Builder
	for _, ch := range chars {
		name := asString(ch["name"])
		looks := asMaps(ch["looks"])
		if len(looks) == 0 {
			if locale == "en" {
				b.WriteString(fmt.Sprintf("[Character: %s]\n---\n", name))
			} else {
				b.WriteString(fmt.Sprintf("【角色·%s】\n---\n", name))
			}
			continue
		}
		for _, lk := range looks {
			label := asString(lk["label"])
			if label == "" {
				label = "默认"
			}
			lookID := asString(lk["id"])
			body := asString(lk["body"])
			if locale == "en" {
				if lookID != "" {
					b.WriteString(fmt.Sprintf("[Character: %s | look %s | %s] %s\n---\n", name, lookID, label, body))
				} else {
					b.WriteString(fmt.Sprintf("[Character: %s | %s] %s\n---\n", name, label, body))
				}
			} else {
				if lookID != "" {
					b.WriteString(fmt.Sprintf("【角色·%s·造型 %s·%s】%s\n---\n", name, lookID, label, body))
				} else {
					b.WriteString(fmt.Sprintf("【角色·%s·%s】%s\n---\n", name, label, body))
				}
			}
		}
	}
	if style != "" {
		if locale == "en" {
			b.WriteString("[Style] " + style)
		} else {
			b.WriteString("【画风】" + style)
		}
	}
	return strings.TrimSpace(b.String())
}

func sceneBeats(art map[string]any) []map[string]any {
	var rows []map[string]any
	for _, beat := range asMaps(art["beats"]) {
		text := asString(beat["narrative"])
		if loc := asString(beat["location"]); loc != "" {
			text = loc + " — " + text
		}
		rows = append(rows, map[string]any{
			"order": asInt(beat["index"]) + 1,
			"title": asString(beat["title"]),
			"beat":  text,
		})
	}
	return rows
}

func MergeBeatShots(existing, newShots []map[string]any, beatIndex int) []map[string]any {
	var before, after []map[string]any
	for _, s := range existing {
		bi := asInt(s["narrative_beat_index"])
		switch {
		case bi < beatIndex:
			before = append(before, s)
		case bi > beatIndex:
			after = append(after, s)
		}
	}
	merged := append(append(before, newShots...), after...)
	for i := range merged {
		merged[i]["order"] = i
	}
	return merged
}

func splitChunks(text string, size int) []string {
	t := strings.TrimSpace(text)
	if len([]rune(t)) <= size {
		return []string{t}
	}
	runes := []rune(t)
	var out []string
	for len(runes) > 0 {
		n := size
		if n > len(runes) {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

func charOnScreen(ch map[string]any, on []string, narrative string) bool {
	name := asString(ch["name"])
	text := strings.ToLower(narrative)
	for _, n := range on {
		if n == name {
			return true
		}
	}
	for _, alias := range asStrings(ch["aliases"]) {
		for _, n := range on {
			if n == alias {
				return true
			}
		}
		if alias != "" && strings.Contains(text, strings.ToLower(alias)) {
			return true
		}
	}
	if name != "" && strings.Contains(text, strings.ToLower(name)) {
		return true
	}
	return false
}

func pickCharacterLook(ch map[string]any, on []string, narrative string) (charID, lookID string) {
	charID = asString(ch["id"])
	looks := asMaps(ch["looks"])
	if charID == "" || len(looks) == 0 {
		return "", ""
	}
	matchText := strings.ToLower(strings.Join(on, " ") + " " + narrative)
	bestScore := -1
	var bestLookID string
	for _, lk := range looks {
		lid := asString(lk["id"])
		if lid == "" {
			continue
		}
		label := strings.ToLower(asString(lk["label"]))
		score := 0
		if label != "" && strings.Contains(matchText, label) {
			score = 10 + len(label)
		}
		if score > bestScore {
			bestScore = score
			bestLookID = lid
		}
	}
	if bestLookID != "" {
		return charID, bestLookID
	}
	def := asString(ch["default_look_id"])
	if def == "" && len(looks) > 0 {
		def = asString(looks[0]["id"])
	}
	return charID, def
}

func pickSceneLook(sc map[string]any, loc string) (sceneID, lookID string) {
	sceneID = asString(sc["id"])
	if sceneID == "" {
		return "", ""
	}
	lookID = asString(sc["default_look_id"])
	looks := asMaps(sc["looks"])
	if lookID == "" && len(looks) > 0 {
		lookID = asString(looks[0]["id"])
	}
	locLower := strings.ToLower(loc)
	for _, lk := range looks {
		lid := asString(lk["id"])
		label := strings.ToLower(asString(lk["label"]))
		if label != "" && locLower != "" && strings.Contains(locLower, label) {
			return sceneID, lid
		}
	}
	return sceneID, lookID
}

func sceneMatches(sc map[string]any, loc string) bool {
	if loc == "" {
		return false
	}
	sid := asString(sc["id"])
	name := asString(sc["name"])
	if sid != "" && loc == sid {
		return true
	}
	if name != "" && (strings.Contains(name, loc) || strings.Contains(loc, name)) {
		return true
	}
	return false
}

func simpleCastLock(shots []map[string]any, chars, scenes []map[string]any) {
	for i, shot := range shots {
		on := asStrings(shot["characters_on_screen"])
		narrative := asString(shot["narrative"])
		var looks []any
		for _, ch := range chars {
			if !charOnScreen(ch, on, narrative) {
				continue
			}
			cid, lid := pickCharacterLook(ch, on, narrative)
			if cid == "" || lid == "" {
				continue
			}
			looks = append(looks, map[string]any{"character_id": cid, "look_id": lid})
		}
		shot["cast_looks"] = looks
		loc := asString(shot["location"])
		for _, sc := range scenes {
			if sceneMatches(sc, loc) {
				sid, lid := pickSceneLook(sc, loc)
				if sid != "" && lid != "" {
					shot["scene_look"] = map[string]any{"scene_id": sid, "look_id": lid}
				}
				break
			}
		}
		shots[i] = shot
	}
}
