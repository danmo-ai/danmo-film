package jsonout

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ExtractObject(text string) (map[string]any, error) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return nil, fmt.Errorf("LLM response is empty; expected a JSON object")
	}
	cleaned := stripFence(raw)
	if i := strings.Index(cleaned, "{"); i > 0 {
		cleaned = cleaned[i:]
	}
	var last error
	for _, cand := range candidates(cleaned) {
		var parsed any
		if err := json.Unmarshal([]byte(cand), &parsed); err != nil {
			last = err
			continue
		}
		m, ok := parsed.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("LLM JSON root must be an object, got %T", parsed)
		}
		return m, nil
	}
	snippet := strings.ReplaceAll(cleaned, "\n", " ")
	if len(snippet) > 280 {
		snippet = snippet[:280]
	}
	detail := ""
	if last != nil {
		detail = ": " + last.Error()
	}
	return nil, fmt.Errorf("LLM response is not valid JSON%s: %s", detail, snippet)
}

func stripFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSpace(t)
	if i := strings.Index(t, "\n"); i >= 0 {
		first := strings.ToLower(strings.TrimSpace(t[:i]))
		if first == "json" || first == "" {
			t = t[i+1:]
		}
	}
	if i := strings.LastIndex(t, "```"); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

func candidates(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(c string) {
		c = strings.TrimSpace(c)
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	add(text)
	add(balanced(text))
	add(braced(text))
	return out
}

func balanced(raw string) string {
	start := strings.Index(raw, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(raw); i++ {
		ch := raw[i]
		if inStr {
			if esc {
				esc = false
			} else if ch == '\\' {
				esc = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		if ch == '"' {
			inStr = true
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return raw[start : i+1]
			}
		}
	}
	return ""
}

func braced(text string) string {
	i := strings.Index(text, "{")
	j := strings.LastIndex(text, "}")
	if i >= 0 && j > i {
		return text[i : j+1]
	}
	return ""
}
