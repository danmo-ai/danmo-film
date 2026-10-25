package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// RemoteModel is one id returned by a provider models list endpoint.
type RemoteModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// FetchRemoteModels lists chat models from an OpenAI-compatible or Anthropic endpoint.
func FetchRemoteModels(ctx context.Context, protocol, baseURL, apiKey string) ([]RemoteModel, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	protocol = strings.TrimSpace(protocol)
	if protocol == "" {
		protocol = "openai"
	}

	client := &http.Client{Timeout: 30 * time.Second}
	url := modelsListURL(protocol, baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	switch protocol {
	case "anthropic":
		if apiKey != "" {
			req.Header.Set("x-api-key", apiKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 400 {
			msg = msg[:400] + "…"
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("models list HTTP %d: %s", resp.StatusCode, msg)
	}

	items, err := parseModelsPayload(body)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("provider returned no models")
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].ID) < strings.ToLower(items[j].ID)
	})
	return items, nil
}

func modelsListURL(protocol, baseURL string) string {
	if protocol == "anthropic" {
		if strings.HasSuffix(baseURL, "/v1") {
			return baseURL + "/models"
		}
		return baseURL + "/v1/models"
	}
	// openai + openai_responses
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/models"
	}
	return baseURL + "/v1/models"
}

func parseModelsPayload(raw []byte) ([]RemoteModel, error) {
	var envelope struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Name        string `json:"name"`
		} `json:"data"`
		Models []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"models"`
		Items []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Name  string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode models list: %w", err)
	}

	seen := map[string]struct{}{}
	out := make([]RemoteModel, 0, 32)
	add := func(id, label string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(label) == "" {
			label = id
		}
		out = append(out, RemoteModel{ID: id, Label: label})
	}

	for _, m := range envelope.Data {
		label := firstNonEmpty(m.DisplayName, m.Name, m.ID)
		add(m.ID, label)
	}
	for _, m := range envelope.Models {
		add(firstNonEmpty(m.ID, m.Name), firstNonEmpty(m.Name, m.ID))
	}
	for _, m := range envelope.Items {
		add(firstNonEmpty(m.ID, m.Name), firstNonEmpty(m.Label, m.Name, m.ID))
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
