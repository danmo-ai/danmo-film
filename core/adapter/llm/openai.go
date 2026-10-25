package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

const defaultChatTimeout = 10 * time.Minute

type OpenAIChatClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	id      string
	model   string
}

func NewOpenAIChatClient(baseURL, apiKey, providerID, defaultModel string) *OpenAIChatClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if providerID == "" {
		providerID = "openai"
	}
	return &OpenAIChatClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: defaultChatTimeout},
		id:      providerID,
		model:   defaultModel,
	}
}

func (c *OpenAIChatClient) ID() string      { return c.id }
func (c *OpenAIChatClient) Available() bool { return c.baseURL != "" }

func (c *OpenAIChatClient) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return port.LLMChatResponse{}, fmt.Errorf("LLM model id is required (request.model or FILM_LLM_MODEL)")
	}
	messages := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{"model": model, "messages": messages, "stream": false}
	if req.GenParams != nil {
		if req.GenParams.MaxTokens > 0 {
			body["max_tokens"] = req.GenParams.MaxTokens
		}
		if req.GenParams.HasTemperature || req.GenParams.Temperature != 0 {
			body["temperature"] = req.GenParams.Temperature
		}
		if req.GenParams.HasTopP && req.GenParams.TopP > 0 {
			body["top_p"] = req.GenParams.TopP
		}
	} else {
		body["temperature"] = 0.35
		body["max_tokens"] = 4096
	}
	dialect := resolveDialectForChat(req.GenParams, model)
	effort := resolveEffort(req.GenParams)
	applyReasoningDialectRequest(body, dialect, effort, req.GenParams)
	raw, err := json.Marshal(body)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	url := c.baseURL + "/v1/chat/completions"
	if strings.HasSuffix(c.baseURL, "/v1") {
		url = c.baseURL + "/chat/completions"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return port.LLMChatResponse{}, fmt.Errorf("LLM chat failed (%d): %s", resp.StatusCode, truncate(string(b), 500))
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return port.LLMChatResponse{}, fmt.Errorf("unexpected LLM response: %w", err)
	}
	content, reasoning, err := openaiContent(data)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	answer, reasoningOut := normalizeChatCompletionsReasoning(content, reasoning, "", dialect)
	out := port.LLMChatResponse{Content: answer, ReasoningContent: reasoningOut, Done: true}
	if u, ok := data["usage"].(map[string]any); ok {
		out.Usage = &port.LLMUsage{
			PromptTokens:     asInt(u["prompt_tokens"]),
			CompletionTokens: asInt(u["completion_tokens"]),
			TotalTokens:      asInt(u["total_tokens"]),
		}
	}
	return out, nil
}

func openaiContent(data map[string]any) (content, reasoning string, err error) {
	choices, _ := data["choices"].([]any)
	if len(choices) == 0 {
		return "", "", fmt.Errorf("unexpected LLM response shape: no choices")
	}
	ch, _ := choices[0].(map[string]any)
	msg, _ := ch["message"].(map[string]any)
	content, ok := msg["content"].(string)
	if !ok {
		return "", "", fmt.Errorf("LLM response content is not a string")
	}
	if rc, ok := msg["reasoning_content"].(string); ok {
		reasoning = rc
	} else if r, ok := msg["reasoning"].(string); ok {
		reasoning = r
	}
	return content, reasoning, nil
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
