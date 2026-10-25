package scriptparse

import (
	"context"
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/domain/jsonout"
	"github.com/danmo-ai/danmo-film/core/port"
)

type ProgressFn func(phase, message string)

func invokeChat(ctx context.Context, llm port.LLMProvider, model, system, user string, maxTokens int) (string, error) {
	resp, err := llm.Chat(ctx, port.LLMChatRequest{
		Model: model,
		Messages: []port.ChatMessage{
			{Role: "system", Content: strings.TrimSpace(system)},
			{Role: "user", Content: strings.TrimSpace(user)},
		},
		// Sampling filled by Router from llm_model_params / llm_default_*.
		GenParams: &port.ModelGenParams{MaxTokens: maxTokens},
	})
	if err != nil {
		return "", err
	}
	return jsonout.ExtractFinal(resp.Content), nil
}

func invokeWithReview(ctx context.Context, llm port.LLMProvider, model, system, user string, maxTokens int, validate func(string) (bool, string), onProgress ProgressFn, passName string) (string, int, error) {
	pending := user
	calls := 0
	var last string
	for attempt := 1; attempt <= 2; attempt++ {
		if attempt > 1 && onProgress != nil && passName != "" {
			onProgress("review_retry", passName)
		}
		resp, err := invokeChat(ctx, llm, model, system, pending, maxTokens)
		calls++
		if err != nil {
			return "", calls, err
		}
		last = resp
		ok, fb := validate(resp)
		if ok {
			return last, calls, nil
		}
		if attempt < 2 && strings.TrimSpace(fb) != "" {
			pending = strings.TrimSpace(user) + "\n\n---\nPrevious output failed validation. Fix ONLY the issues below; return full corrected JSON.\n\nIssues:\n" + strings.TrimSpace(fb)
		}
	}
	fb := ""
	if validate != nil {
		_, fb = validate(last)
	}
	if strings.TrimSpace(fb) == "" {
		fb = "validation failed after retries"
	}
	return "", calls, fmt.Errorf("%s: %s", passName, fb)
}

func localeBlock(locale string) string {
	if NormalizeLocale(locale) == "zh" {
		return "\n\n## Output language\nWrite every JSON string value in Simplified Chinese (简体中文). Keep character names in the script's Chinese form."
	}
	return "\n\n## Output language\nWrite every JSON string value in English."
}

func NormalizeLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(l, "zh") {
		return "zh"
	}
	if l == "" {
		return "zh"
	}
	return "en"
}

func tokenBudget(n int) int {
	if n < 256 {
		return 256
	}
	return n
}

func asMaps(v any) []map[string]any {
	arr, _ := v.([]any)
	var out []map[string]any
	for _, it := range arr {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func asFloat(v any, def float64) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return def
	}
}

func asInt(v any) int { return int(asFloat(v, 0)) }

func asStrings(v any) []string {
	arr, _ := v.([]any)
	var out []string
	for _, it := range arr {
		s := strings.TrimSpace(fmt.Sprint(it))
		if s != "" && s != "<nil>" {
			out = append(out, s)
		}
	}
	return out
}
