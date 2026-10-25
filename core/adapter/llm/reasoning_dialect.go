package llm

import (
	"regexp"
	"strings"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

var thinkTagRe = regexp.MustCompile(`(?is)<think>(.*?)</think>`)

func resolveEffort(gp *port.ModelGenParams) string {
	if gp == nil {
		return "off"
	}
	mode := strings.TrimSpace(strings.ToLower(gp.ThinkingMode))
	switch mode {
	case "", "off", "disabled", "none":
		return "off"
	case "enabled", "on", "adaptive":
		return "high"
	default:
		return mode // low|medium|high|max|…
	}
}

func resolveDialectForChat(gp *port.ModelGenParams, model string) string {
	if gp != nil && strings.TrimSpace(gp.ReasoningDialect) != "" {
		return strings.TrimSpace(gp.ReasoningDialect)
	}
	if d := config.InferReasoningDialect(model); d != "" {
		return d
	}
	return config.DialectOpenAI
}

// applyReasoningDialectRequest mutates Chat Completions body (danmo-work matrix).
func applyReasoningDialectRequest(body map[string]any, dialect, effort string, gp *port.ModelGenParams) {
	on := effort != "" && effort != "off"
	switch dialect {
	case config.DialectQwen:
		body["enable_thinking"] = on
		if on && gp != nil && len(gp.EffortBudgetTokens) > 0 {
			if budget := gp.EffortBudgetTokens[effort]; budget > 0 {
				body["thinking_budget"] = budget
			}
		}
	case config.DialectKimi:
		if on {
			body["thinking"] = map[string]any{"type": "enabled", "keep": "all"}
		} else {
			body["thinking"] = map[string]any{"type": "disabled"}
		}
	case config.DialectKimiCode:
		body["thinking"] = map[string]any{"type": "enabled", "keep": "all"}
	case config.DialectKimiK3:
		e := effort
		if !on {
			e = "low"
		}
		body["reasoning_effort"] = e
	case config.DialectGLM:
		if on {
			body["thinking"] = map[string]any{"type": "enabled", "clear_thinking": false}
			body["reasoning_effort"] = mapGLMEffort(effort)
		} else {
			body["thinking"] = map[string]any{"type": "disabled"}
		}
	case config.DialectMiniMax:
		body["reasoning_split"] = true
		if on {
			body["thinking"] = map[string]any{"type": "adaptive"}
		} else {
			body["thinking"] = map[string]any{"type": "disabled"}
		}
	case config.DialectGemini:
		if on {
			body["reasoning_effort"] = effort
		} else {
			body["reasoning_effort"] = "none"
		}
	case config.DialectDeepSeek:
		if on {
			body["thinking"] = map[string]any{"type": "enabled"}
			body["reasoning_effort"] = mapDeepSeekEffort(effort)
		} else {
			body["thinking"] = map[string]any{"type": "disabled"}
		}
	case config.DialectGrok:
		if effort == "" || effort == "off" {
			body["reasoning_effort"] = "low"
		} else {
			body["reasoning_effort"] = effort
		}
	default: // openai
		if on {
			body["reasoning_effort"] = effort
		}
	}
}

func mapDeepSeekEffort(effort string) string {
	switch strings.ToLower(effort) {
	case "minimal", "low":
		return "low"
	case "max", "ultra":
		return "max"
	default:
		return "high"
	}
}

func mapGLMEffort(effort string) string {
	switch strings.ToLower(effort) {
	case "max", "xhigh":
		return "max"
	case "minimal", "none", "low", "medium":
		return "high"
	default:
		return effort
	}
}

func normalizeChatCompletionsReasoning(content, reasoningContent, reasoning, dialect string) (answer, reasoningOut string) {
	reasoningOut = strings.TrimSpace(reasoningContent)
	if reasoningOut == "" {
		reasoningOut = strings.TrimSpace(reasoning)
	}
	answer = content
	if reasoningOut == "" && (dialect == config.DialectMiniMax || dialect == config.DialectDeepSeek) {
		if m := thinkTagRe.FindStringSubmatch(content); len(m) == 2 {
			reasoningOut = strings.TrimSpace(m[1])
			answer = strings.TrimSpace(thinkTagRe.ReplaceAllString(content, ""))
		}
	}
	return answer, reasoningOut
}
