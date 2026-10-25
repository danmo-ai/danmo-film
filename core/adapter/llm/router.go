package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
)

// Router dispatches Chat to the LLM profile that owns the requested model.
type Router struct {
	mu       sync.RWMutex
	settings config.Settings
	clients  map[string]port.LLMProvider // profile id → client
}

func NewRouter(settings config.Settings) (*Router, error) {
	r := &Router{
		settings: config.Clone(settings),
		clients:  map[string]port.LLMProvider{},
	}
	if err := r.rebuildLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Router) ID() string { return "llm-router" }

func (r *Router) Available() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range config.LLMProfiles(r.settings) {
		if p.Enabled && strings.TrimSpace(p.BaseURL) != "" {
			return true
		}
	}
	return len(r.clients) > 0
}

func (r *Router) Update(settings config.Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = config.Clone(settings)
	r.clients = map[string]port.LLMProvider{}
	return r.rebuildLocked()
}

func (r *Router) rebuildLocked() error {
	profiles := config.LLMProfiles(r.settings)
	if len(profiles) == 0 {
		return fmt.Errorf("no LLM profiles configured")
	}
	var firstErr error
	for _, p := range profiles {
		if !p.Enabled {
			continue
		}
		c, err := clientForProfile(p)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		r.clients[p.ID] = c
	}
	if len(r.clients) == 0 {
		if firstErr != nil {
			return firstErr
		}
		return fmt.Errorf("no enabled LLM profiles")
	}
	return nil
}

func (r *Router) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	r.mu.RLock()
	settings := r.settings
	r.mu.RUnlock()

	ref := strings.TrimSpace(req.Model)
	if ref == "" {
		ref = config.DefaultLLMModelID(settings)
	}
	profile, err := config.FindLLMProfileForModel(settings, ref)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	// Create-page ref → bare model id for upstream + param lookup.
	bare, err := config.BareLLMModelID(ref)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	req.Model = bare

	if req.GenParams == nil {
		req.GenParams = &port.ModelGenParams{}
	}
	resolved := config.ResolveModelParams(settings, bare)

	// Protocol dialect: provider × model (this profile's model row), not model id alone.
	proto := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(profile.Protocol), "-", "_"))
	if proto == "openai" || proto == "openai_compatible" || proto == "local" || proto == "" {
		if strings.TrimSpace(req.GenParams.ReasoningDialect) == "" {
			req.GenParams.ReasoningDialect = config.ResolveReasoningDialect(profile, bare)
		}
	}

	// Capacity + sampling + thinking level: bare model id → llm_model_params → defaults.
	// (Dialect only encodes thinking_mode; the value lives here.)
	if !req.GenParams.HasTemperature {
		req.GenParams.Temperature = resolved.Temperature
		req.GenParams.HasTemperature = true
	}
	if !req.GenParams.HasTopP && resolved.TopP > 0 {
		req.GenParams.TopP = resolved.TopP
		req.GenParams.HasTopP = true
	}
	if strings.TrimSpace(req.GenParams.ThinkingMode) == "" {
		req.GenParams.ThinkingMode = resolved.ThinkingMode
	}
	if resolved.ThinkingBudget > 0 && len(req.GenParams.EffortBudgetTokens) == 0 {
		effort := strings.TrimSpace(strings.ToLower(req.GenParams.ThinkingMode))
		if effort != "" && effort != "off" && effort != "disabled" && effort != "none" {
			if effort == "on" || effort == "enabled" || effort == "adaptive" {
				effort = "high"
			}
			req.GenParams.EffortBudgetTokens = map[string]int{effort: resolved.ThinkingBudget}
		}
	}
	req.GenParams.MaxTokens = config.ClampMaxTokens(req.GenParams.MaxTokens, resolved.MaxOutputTokens)

	r.mu.RLock()
	client := r.clients[profile.ID]
	r.mu.RUnlock()
	if client == nil {
		c, err := clientForProfile(profile)
		if err != nil {
			return port.LLMChatResponse{}, err
		}
		r.mu.Lock()
		r.clients[profile.ID] = c
		r.mu.Unlock()
		client = c
	}
	return client.Chat(ctx, req)
}

func clientForProfile(p config.LLMProfile) (port.LLMProvider, error) {
	protocol := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p.Protocol), "-", "_"))
	defaultModel := ""
	for _, m := range p.Models {
		if m.Enabled {
			defaultModel = m.ID
			break
		}
	}
	switch protocol {
	case "openai", "openai_compatible", "local", "":
		return NewOpenAIChatClient(p.BaseURL, p.APIKey, p.ID, defaultModel), nil
	case "openai_responses", "responses":
		return NewOpenAIResponsesClient(p.BaseURL, p.APIKey, p.ID, defaultModel), nil
	case "anthropic":
		return NewAnthropicClient(p.BaseURL, p.APIKey, p.ID, defaultModel), nil
	case "mock":
		return &MockClient{}, nil
	default:
		if strings.Contains(protocol, "openai") {
			return NewOpenAIChatClient(p.BaseURL, p.APIKey, p.ID, defaultModel), nil
		}
		return nil, fmt.Errorf("unknown LLM protocol %q on profile %q", p.Protocol, p.ID)
	}
}
