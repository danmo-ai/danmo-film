package port

import "context"

type ChatContentPart struct {
	Type     string // text | image
	Text     string
	MimeType string
	Data     string
	Name     string
}

type ChatMessage struct {
	Role             string
	Content          string
	Parts            []ChatContentPart
	ToolCalls        []ChatToolCall
	ToolCallID       string
	Name             string
	ReasoningContent string
}

type ChatToolCall struct {
	ID           string
	Name         string
	Arguments    map[string]any
	RepairedFrom string
}

type ModelGenParams struct {
	MaxTokens          int
	Temperature        float64
	TopP               float64
	FrequencyPenalty   float64
	PresencePenalty    float64
	Stop               []string
	ThinkingMode       string
	EffortBudgetTokens map[string]int
	ReasoningDialect   string
	// HasTemperature / HasTopP: when false, Router fills from llm_model_params / defaults.
	HasTemperature bool
	HasTopP        bool
}

type LLMChatRequest struct {
	Model     string
	Messages  []ChatMessage
	Tools     []map[string]any
	ToolChoice string
	GenParams *ModelGenParams
}

type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type LLMChatResponse struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ChatToolCall
	Usage            *LLMUsage
	Done             bool
}

// LLMProvider is the only LLM surface domain code may call.
type LLMProvider interface {
	ID() string
	Available() bool
	Chat(ctx context.Context, req LLMChatRequest) (LLMChatResponse, error)
}
