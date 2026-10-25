package llm

import (
	"context"
	"fmt"

	"github.com/danmo-ai/danmo-film/core/port"
)

type MockClient struct {
	Reply string
}

func (m *MockClient) ID() string      { return "mock" }
func (m *MockClient) Available() bool { return true }

func (m *MockClient) Chat(_ context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	if m.Reply == "" {
		return port.LLMChatResponse{}, fmt.Errorf("mock LLM has no reply")
	}
	_ = req
	return port.LLMChatResponse{Content: m.Reply, Done: true}, nil
}
