package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

type OpenAIResponsesClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	id      string
	model   string
}

func NewOpenAIResponsesClient(baseURL, apiKey, providerID, defaultModel string) *OpenAIResponsesClient {
	if providerID == "" {
		providerID = "openai_responses"
	}
	return &OpenAIResponsesClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: defaultChatTimeout},
		id:      providerID,
		model:   defaultModel,
	}
}

func (c *OpenAIResponsesClient) ID() string      { return c.id }
func (c *OpenAIResponsesClient) Available() bool { return c.baseURL != "" }

func (c *OpenAIResponsesClient) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return port.LLMChatResponse{}, fmt.Errorf("LLM model id is required")
	}
	input := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		input = append(input, map[string]any{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{"model": model, "input": input}
	if req.GenParams != nil {
		if req.GenParams.MaxTokens > 0 {
			body["max_output_tokens"] = req.GenParams.MaxTokens
		}
		if req.GenParams.HasTemperature || req.GenParams.Temperature != 0 {
			body["temperature"] = req.GenParams.Temperature
		}
		if req.GenParams.HasTopP && req.GenParams.TopP > 0 {
			body["top_p"] = req.GenParams.TopP
		}
	}
	raw, _ := json.Marshal(body)
	url := strings.TrimRight(c.baseURL, "/") + "/responses"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
		return port.LLMChatResponse{}, fmt.Errorf("LLM responses failed (%d): %s", resp.StatusCode, truncate(string(b), 500))
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return port.LLMChatResponse{}, err
	}
	text := extractResponsesText(data)
	if text == "" {
		return port.LLMChatResponse{}, fmt.Errorf("unexpected Responses API shape")
	}
	return port.LLMChatResponse{Content: text, Done: true}, nil
}

func extractResponsesText(data map[string]any) string {
	if s, ok := data["output_text"].(string); ok && s != "" {
		return s
	}
	out, _ := data["output"].([]any)
	var b strings.Builder
	for _, item := range out {
		m, _ := item.(map[string]any)
		content, _ := m["content"].([]any)
		for _, c := range content {
			cm, _ := c.(map[string]any)
			if t, ok := cm["text"].(string); ok {
				b.WriteString(t)
			}
		}
	}
	return b.String()
}
