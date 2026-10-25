package config

import (
	"fmt"
	"strings"
)

// LLM model refs used by the create page and llm_default_model:
//   provider_id/model_id   e.g. "openai-chat/gpt-5-mini"
//
// provider_id + model_id → LLM client (profile) + protocol dialect (that profile's model row).
// model_id alone         → llm_model_params / llm_default_* (capacity + sampling).
// Upstream request body still uses bare model_id.

// FormatLLMModelRef builds provider_id/model_id. Both parts are required.
func FormatLLMModelRef(providerID, modelID string) string {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	if providerID == "" || modelID == "" {
		return ""
	}
	return providerID + "/" + modelID
}

// ParseLLMModelRef splits provider_id/model_id. Bare ids are rejected.
func ParseLLMModelRef(ref string) (providerID, modelID string, err error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", fmt.Errorf("empty LLM model ref")
	}
	i := strings.Index(ref, "/")
	if i <= 0 || i >= len(ref)-1 {
		return "", "", fmt.Errorf("LLM model ref must be provider_id/model_id, got %q", ref)
	}
	providerID = strings.TrimSpace(ref[:i])
	modelID = strings.TrimSpace(ref[i+1:])
	if providerID == "" || modelID == "" {
		return "", "", fmt.Errorf("LLM model ref must be provider_id/model_id, got %q", ref)
	}
	return providerID, modelID, nil
}

// BareLLMModelID returns the upstream model id from a provider_id/model_id ref.
func BareLLMModelID(ref string) (string, error) {
	_, modelID, err := ParseLLMModelRef(ref)
	return modelID, err
}
