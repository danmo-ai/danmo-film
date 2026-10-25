package httpapi

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"

	genadapt "github.com/danmo-ai/danmo-film/core/adapter/gen"
	llmadapt "github.com/danmo-ai/danmo-film/core/adapter/llm"
	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/runtimepaths"
)

func (a *App) registerSettings(r *gin.Engine) {
	g := r.Group("/api/settings")
	g.GET("", a.getSettings)
	g.PUT("", a.putSettings)
	g.GET("/system", a.getSettingsSystem)
	g.GET("/workspace-paths", a.getWorkspacePaths)
	g.GET("/llm/models", a.listLLMModels)
	g.GET("/llm/presets", a.listLLMPresets)
	g.POST("/llm/fetch-models", a.fetchLLMModels)
}

func (a *App) getSettings(c *gin.Context) {
	a.mu.RLock()
	s := config.PublicView(a.Settings)
	profiles := config.LLMProfiles(a.Settings)
	defaultModel := config.DefaultLLMModelID(a.Settings)
	defCtx := config.DefaultLLMContextWindow(a.Settings)
	defMax := config.DefaultLLMMaxOutputTokens(a.Settings)
	defTemp := config.DefaultLLMTemperature(a.Settings)
	defTopP := config.DefaultLLMTopP(a.Settings)
	defThink := config.DefaultLLMThinkingMode(a.Settings)
	modelParams := config.ModelParamsToAny(config.LLMModelParamsMap(a.Settings))
	a.mu.RUnlock()
	providers := s.Map("providers")
	providers["llm_profiles"] = config.ProfilesToAny(maskProfiles(profiles))
	providers["llm_default_model"] = defaultModel
	providers["llm_default_context_window"] = defCtx
	providers["llm_default_max_output_tokens"] = defMax
	providers["llm_default_temperature"] = defTemp
	providers["llm_default_top_p"] = defTopP
	providers["llm_default_thinking_mode"] = defThink
	providers["llm_model_params"] = modelParams
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"paths": gin.H{
			"settings_file": runtimepaths.SettingsPath(),
			"user_data_dir": runtimepaths.UserDataDir(),
			"workspace_dir": runtimepaths.WorkspaceDir(),
			"assets_root":   runtimepaths.FilmAssetsRoot(),
			"film_db":       runtimepaths.FilmDBPath(),
		},
		"catalog": providerCatalog(),
	})
}

func (a *App) listLLMPresets(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items":              config.LLMProviderPresets(),
		"reasoning_dialects": config.ReasoningDialectOptions(),
	})
}

func (a *App) listLLMModels(c *gin.Context) {
	a.mu.RLock()
	s := a.Settings
	a.mu.RUnlock()
	items := config.ListEnabledLLMModels(s)
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"default_model": config.DefaultLLMModelID(s),
	})
}

type fetchLLMModelsReq struct {
	ProfileID string `json:"profile_id"`
	Protocol  string `json:"protocol"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
}

func (a *App) fetchLLMModels(c *gin.Context) {
	var req fetchLLMModelsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err)
		return
	}
	baseURL := strings.TrimSpace(req.BaseURL)
	protocol := strings.TrimSpace(req.Protocol)
	apiKey := strings.TrimSpace(req.APIKey)
	needSavedKey := apiKey == "" || apiKey == config.SecretSentinel
	if needSavedKey || baseURL == "" || protocol == "" {
		a.mu.RLock()
		profiles := config.LLMProfiles(a.Settings)
		a.mu.RUnlock()
		pid := strings.TrimSpace(req.ProfileID)
		for _, p := range profiles {
			match := pid != "" && p.ID == pid
			if !match && pid == "" && baseURL != "" && strings.TrimSpace(p.BaseURL) == baseURL {
				match = true
			}
			if !match {
				continue
			}
			if baseURL == "" {
				baseURL = p.BaseURL
			}
			if protocol == "" {
				protocol = p.Protocol
			}
			if needSavedKey {
				apiKey = p.APIKey
			}
			break
		}
	}
	if baseURL == "" {
		fail(c, 400, fmt.Errorf("base_url is required"))
		return
	}
	items, err := llmadapt.FetchRemoteModels(c.Request.Context(), protocol, baseURL, apiKey)
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func maskProfiles(profiles []config.LLMProfile) []config.LLMProfile {
	out := make([]config.LLMProfile, len(profiles))
	copy(out, profiles)
	for i := range out {
		if strings.TrimSpace(out[i].APIKey) != "" {
			out[i].APIKey = config.SecretSentinel
		}
		out[i].Models = append([]config.LLMModel(nil), out[i].Models...)
	}
	return out
}

func (a *App) putSettings(c *gin.Context) {
	var incoming config.Settings
	if err := c.ShouldBindJSON(&incoming); err != nil {
		fail(c, 400, err)
		return
	}
	a.mu.RLock()
	existing := config.Clone(a.Settings)
	a.mu.RUnlock()
	merged := config.MergeIncoming(existing, incoming)

	if err := config.ValidateLLMProfiles(config.LLMProfiles(merged)); err != nil {
		fail(c, 400, err)
		return
	}

	llm, err := llmadapt.Resolve(merged)
	if err != nil {
		fail(c, 400, err)
		return
	}
	gen, err := genadapt.Resolve(merged)
	if err != nil {
		fail(c, 400, err)
		return
	}
	if err := config.Save(merged); err != nil {
		fail(c, 500, err)
		return
	}
	a.mu.Lock()
	a.Settings = merged
	a.LLM = llm
	a.Gen = gen
	a.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"restart_required": false,
		"generation":       gin.H{"provider_id": gen.ID()},
		"llm":              gin.H{"provider_id": llm.ID(), "available": llm.Available()},
	})
}

func (a *App) getSettingsSystem(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"env_ready":     true,
		"platform":      runtime.GOOS,
		"architecture":  runtime.GOARCH,
		"go_version":    runtime.Version(),
		"user_data_dir": runtimepaths.UserDataDir(),
		"workspace_dir": runtimepaths.WorkspaceDir(),
	})
}

func (a *App) getWorkspacePaths(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"settings_file": runtimepaths.SettingsPath(),
		"user_data_dir": runtimepaths.UserDataDir(),
		"workspace_dir": runtimepaths.WorkspaceDir(),
		"assets_root":   runtimepaths.FilmAssetsRoot(),
		"film_db":       runtimepaths.FilmDBPath(),
	})
}

func providerCatalog() []map[string]any {
	out := []map[string]any{
		{"id": "volcengine", "kind": "generation", "label": "Volcengine Ark Seedance", "needs_api_key": true},
		{"id": "dashscope", "kind": "generation", "label": "DashScope 万相", "needs_api_key": true},
		{"id": "minimax", "kind": "generation", "label": "MiniMax Hailuo", "needs_api_key": true},
	}
	for _, p := range config.LLMProviderPresets() {
		out = append(out, map[string]any{
			"id":              p.ID,
			"kind":            "llm",
			"protocol":        p.Protocol,
			"label":           p.Label,
			"base_url":        p.BaseURL,
			"auth_style":      p.AuthStyle,
			"default_dialect": p.DefaultDialect,
			"models":          p.Models,
			"protocol_locked": !config.PresetAllowsProtocolEdit(p.ID),
		})
	}
	return out
}
