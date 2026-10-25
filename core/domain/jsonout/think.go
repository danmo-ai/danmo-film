package jsonout

import (
	"regexp"
	"strings"
)

var (
	thinkBlock = regexp.MustCompile(`(?is)<(?:think|redacted_reasoning)>\s*(.*?)\s*</(?:think|redacted_reasoning)>`)
	thinkEnd   = regexp.MustCompile(`(?i)</(?:think|redacted_reasoning)>`)
	thinkStart = regexp.MustCompile(`(?i)<(?:think|redacted_reasoning)>\s*`)
	imEnd      = regexp.MustCompile(`<\|im_end\|>\s*$`)
)

func ExtractFinal(text string) string {
	thinking, answer := parseThinking(text)
	if answer != "" {
		return answer
	}
	if thinking != "" {
		return ""
	}
	return stripSpecial(text)
}

func parseThinking(text string) (string, string) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return "", ""
	}
	locs := thinkEnd.FindAllStringIndex(raw, -1)
	if len(locs) > 0 {
		last := locs[len(locs)-1]
		answer := stripSpecial(raw[last[1]:])
		var parts []string
		for _, m := range thinkBlock.FindAllStringSubmatch(raw, -1) {
			if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
				parts = append(parts, strings.TrimSpace(m[1]))
			}
		}
		return strings.Join(parts, "\n\n"), answer
	}
	if thinkStart.MatchString(raw) {
		return stripSpecial(thinkStart.ReplaceAllString(raw, "")), ""
	}
	return "", stripSpecial(raw)
}

func stripSpecial(s string) string {
	return strings.TrimSpace(imEnd.ReplaceAllString(strings.TrimSpace(s), ""))
}
