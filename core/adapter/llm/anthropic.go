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

type AnthropicClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	id      string
	model   string
}

func NewAnthropicClient(baseURL, apiKey, providerID, defaultModel string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if providerID == "" {
		providerID = "anthropic"
	}
	return &AnthropicClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: defaultChatTimeout},
		id:      providerID,
		model:   defaultModel,
	}
}

func (c *AnthropicClient) ID() string      { return c.id }
func (c *AnthropicClient) Available() bool { return c.apiKey != "" && c.baseURL != "" }

func (c *AnthropicClient) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	model := req.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		return port.LLMChatResponse{}, fmt.Errorf("LLM model id is required")
	}
	var system string
	var msgs []map[string]any
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
			continue
		}
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	maxTok := 4096
	temp := 0.35
	if req.GenParams != nil {
		if req.GenParams.MaxTokens > 0 {
			maxTok = req.GenParams.MaxTokens
		}
		if req.GenParams.HasTemperature || req.GenParams.Temperature != 0 {
			temp = req.GenParams.Temperature
		}
	}
	body := map[string]any{"model": model, "max_tokens": maxTok, "temperature": temp, "messages": msgs}
	if req.GenParams != nil && req.GenParams.HasTopP && req.GenParams.TopP > 0 {
		body["top_p"] = req.GenParams.TopP
	}
	if system != "" {
		body["system"] = system
	}
	raw, _ := json.Marshal(body)
	url := c.baseURL + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return port.LLMChatResponse{}, fmt.Errorf("Anthropic chat failed (%d): %s", resp.StatusCode, truncate(string(b), 500))
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return port.LLMChatResponse{}, err
	}
	content := anthropicText(data)
	if content == "" {
		return port.LLMChatResponse{}, fmt.Errorf("unexpected Anthropic response shape")
	}
	return port.LLMChatResponse{Content: content, Done: true}, nil
}

func anthropicText(data map[string]any) string {
	arr, _ := data["content"].([]any)
	var b strings.Builder
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if m["type"] == "text" {
			if t, ok := m["text"].(string); ok {
				b.WriteString(t)
			}
		}
	}
	return b.String()
}
