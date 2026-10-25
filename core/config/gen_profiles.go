package config

import (
	"fmt"
	"strings"
)

// KnownGenVendors are adapter presets (not unique instance ids).
var KnownGenVendors = []string{"danmo-make", "volcengine", "dashscope", "minimax"}

// GenModel is one enabled/disabled model row on a video profile.
type GenModel struct {
	ID      string `json:"id"`
	Label   string `json:"label,omitempty"`
	Enabled bool   `json:"enabled"`
}

// GenProfile is one video provider instance (LLM-style): same Vendor may appear
// multiple times with different id/label/base_url/api_key.
type GenProfile struct {
	ID      string     `json:"id"`
	Label   string     `json:"label,omitempty"`
	Vendor  string     `json:"vendor"` // adapter kind: danmo-make|volcengine|dashscope|minimax
	BaseURL string     `json:"base_url"`
	APIKey  string     `json:"api_key,omitempty"`
	Enabled bool       `json:"enabled"`
	Models  []GenModel `json:"models"`
}

// GenVendor is the runtime adapter binding for a profile (ID = profile id).
type GenVendor struct {
	ID      string // profile instance id (routing / task prefix)
	Vendor  string // adapter kind
	Enabled bool
	BaseURL string
	APIKey  string
	Models  []GenModel
}

// EnabledGenModel is a create-page picker row: profile_id/model_id.
type EnabledGenModel struct {
	Ref           string `json:"ref"`
	ID            string `json:"id"`
	Label         string `json:"label"`
	ProviderID    string `json:"provider_id"`
	ProviderLabel string `json:"provider_label"`
	Vendor        string `json:"vendor,omitempty"`
}

// GenVendorLabel returns a short default display name for a vendor preset.
func GenVendorLabel(id string) string {
	switch NormalizeGenVendorID(id) {
	case "volcengine":
		return "火山引擎"
	case "dashscope":
		return "阿里云"
	case "minimax":
		return "MiniMax"
	case "danmo-make":
		return "Danmo Make"
	default:
		return id
	}
}

// GenVendorDefaultBaseURL returns the Settings default Base URL for a preset.
func GenVendorDefaultBaseURL(vendor string) string {
	switch NormalizeGenVendorID(vendor) {
	case "danmo-make":
		return "http://127.0.0.1:7800"
	case "volcengine":
		return "https://ark.cn-beijing.volces.com"
	case "dashscope":
		return "https://dashscope.aliyuncs.com"
	case "minimax":
		return "https://api.minimax.chat"
	default:
		return ""
	}
}

// IsKnownGenVendor reports whether id is a known adapter preset.
func IsKnownGenVendor(id string) bool {
	id = NormalizeGenVendorID(id)
	for _, k := range KnownGenVendors {
		if k == id {
			return true
		}
	}
	return false
}

// GenerationProfiles returns configured video provider instances.
// Migrates legacy string[] ids + providers.<vendor> blocks when needed.
func GenerationProfiles(s Settings) []GenProfile {
	raw := s.Map("providers")["generation_profiles"]
	if profiles, ok := parseGenProfileObjects(raw); ok && len(profiles) > 0 {
		return profiles
	}
	return migrateLegacyGenProfiles(s, raw)
}

func parseGenProfileObjects(raw any) ([]GenProfile, bool) {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil, false
	}
	// Detect object list vs legacy string list.
	hasObj := false
	for _, v := range arr {
		if _, ok := v.(map[string]any); ok {
			hasObj = true
			break
		}
	}
	if !hasObj {
		return nil, false
	}
	out := make([]GenProfile, 0, len(arr))
	seen := map[string]bool{}
	for _, v := range arr {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		p := parseGenProfileMap(m)
		if p.ID == "" || seen[p.ID] {
			continue
		}
		if !IsKnownGenVendor(p.Vendor) {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out, len(out) > 0
}

func parseGenProfileMap(m map[string]any) GenProfile {
	id := strings.TrimSpace(strAny(m["id"]))
	vendor := NormalizeGenVendorID(FirstNonEmpty(strAny(m["vendor"]), strAny(m["preset_id"]), id))
	label := strings.TrimSpace(strAny(m["label"]))
	if label == "" {
		label = GenVendorLabel(vendor)
	}
	p := GenProfile{
		ID:      id,
		Label:   label,
		Vendor:  vendor,
		BaseURL: strings.TrimSpace(strAny(m["base_url"])),
		APIKey:  strings.TrimSpace(strAny(m["api_key"])),
		Enabled: boolAny(m["enabled"], true),
		Models:  parseGenModels(m),
	}
	if p.BaseURL == "" {
		p.BaseURL = GenVendorDefaultBaseURL(vendor)
	}
	if len(p.Models) == 0 {
		legacy := strings.TrimSpace(strAny(m["model"]))
		if legacy != "" {
			p.Models = []GenModel{{ID: legacy, Label: legacy, Enabled: true}}
		}
	}
	return p
}

func migrateLegacyGenProfiles(s Settings, raw any) []GenProfile {
	known := map[string]bool{}
	for _, id := range KnownGenVendors {
		known[id] = true
	}
	var ids []string
	seen := map[string]bool{}
	if arr, ok := raw.([]any); ok {
		for _, v := range arr {
			id := NormalizeGenVendorID(fmt.Sprint(v))
			if !known[id] || seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		active := NormalizeGenVendorID(s.String("providers", "generation", "provider"))
		if active != "" && known[active] {
			ids = append(ids, active)
			seen[active] = true
		}
		for _, id := range KnownGenVendors {
			if seen[id] {
				continue
			}
			block := legacyVendorBlock(s, id)
			if strings.TrimSpace(strAny(block["api_key"])) != "" {
				ids = append(ids, id)
				seen[id] = true
				continue
			}
			models := parseGenModels(block)
			for _, m := range models {
				if m.Enabled && strings.TrimSpace(m.ID) != "" {
					ids = append(ids, id)
					seen[id] = true
					break
				}
			}
		}
	}
	out := make([]GenProfile, 0, len(ids))
	for _, id := range ids {
		out = append(out, profileFromLegacyBlock(s, id))
	}
	return out
}

func legacyVendorBlock(s Settings, vendor string) map[string]any {
	vendor = NormalizeGenVendorID(vendor)
	m := s.Map("providers", vendor)
	if vendor == "danmo-make" && len(m) == 0 {
		m = s.Map("providers", "generation")
	}
	return m
}

func profileFromLegacyBlock(s Settings, vendor string) GenProfile {
	vendor = NormalizeGenVendorID(vendor)
	m := legacyVendorBlock(s, vendor)
	p := GenProfile{
		ID:      vendor,
		Label:   GenVendorLabel(vendor),
		Vendor:  vendor,
		BaseURL: strings.TrimSpace(strAny(m["base_url"])),
		APIKey:  strings.TrimSpace(strAny(m["api_key"])),
		Enabled: true,
		Models:  parseGenModels(m),
	}
	if _, ok := m["enabled"]; ok {
		p.Enabled = boolAny(m["enabled"], true)
	}
	if vendor == "danmo-make" {
		if p.BaseURL == "" {
			p.BaseURL = strings.TrimSpace(s.String("providers", "generation", "base_url"))
		}
		if p.APIKey == "" {
			p.APIKey = strings.TrimSpace(s.String("providers", "generation", "api_key"))
		}
	}
	if p.BaseURL == "" {
		p.BaseURL = GenVendorDefaultBaseURL(vendor)
	}
	if len(p.Models) == 0 {
		legacy := strings.TrimSpace(strAny(m["model"]))
		if legacy == "" && vendor == NormalizeGenVendorID(s.String("providers", "generation", "provider")) {
			legacy = strings.TrimSpace(s.String("providers", "generation", "model"))
		}
		if legacy != "" {
			p.Models = []GenModel{{ID: legacy, Label: legacy, Enabled: true}}
		}
	}
	return p
}

// NormalizeGenVendorID maps aliases / legacy labels to a known vendor preset id.
func NormalizeGenVendorID(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "volcengine", "bytedance", "seedance", "ark", "火山引擎":
		return "volcengine"
	case "dashscope", "alibaba", "ali", "wanxiang", "阿里云":
		return "dashscope"
	case "minimax", "hailuo", "minimax hailuo":
		return "minimax"
	case "danmo-make", "make", "danqing", "danmo make":
		return "danmo-make"
	default:
		return strings.TrimSpace(raw)
	}
}

// FindGenProfile returns a profile by instance id.
func FindGenProfile(s Settings, profileID string) (GenProfile, bool) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return GenProfile{}, false
	}
	for _, p := range GenerationProfiles(s) {
		if p.ID == profileID {
			return p, true
		}
	}
	// Legacy: profile id equals vendor id before multi-profile migration ran.
	if IsKnownGenVendor(profileID) {
		for _, p := range GenerationProfiles(s) {
			if p.Vendor == NormalizeGenVendorID(profileID) && p.ID == NormalizeGenVendorID(profileID) {
				return p, true
			}
		}
	}
	return GenProfile{}, false
}

// GenVendorConfig loads runtime vendor binding for a profile id (legacy name kept).
func GenVendorConfig(s Settings, profileID string) GenVendor {
	if p, ok := FindGenProfile(s, profileID); ok {
		return profileToVendor(p)
	}
	// Fall back to synthesizing from legacy vendor block.
	vendor := NormalizeGenVendorID(profileID)
	if IsKnownGenVendor(vendor) {
		return profileToVendor(profileFromLegacyBlock(s, vendor))
	}
	return GenVendor{ID: profileID, Vendor: vendor}
}

func profileToVendor(p GenProfile) GenVendor {
	return GenVendor{
		ID:      p.ID,
		Vendor:  NormalizeGenVendorID(p.Vendor),
		Enabled: p.Enabled,
		BaseURL: p.BaseURL,
		APIKey:  p.APIKey,
		Models:  p.Models,
	}
}

func parseGenModels(m map[string]any) []GenModel {
	raw, ok := m["models"]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []GenModel
	for _, item := range arr {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(strAny(row["id"]))
		if id == "" {
			continue
		}
		label := strings.TrimSpace(strAny(row["label"]))
		if label == "" {
			label = id
		}
		en := boolAny(row["enabled"], true)
		out = append(out, GenModel{ID: id, Label: label, Enabled: en})
	}
	return out
}

// ListEnabledGenModels returns picker refs for enabled profiles × enabled models.
func ListEnabledGenModels(s Settings) []EnabledGenModel {
	var out []EnabledGenModel
	for _, p := range GenerationProfiles(s) {
		if !p.Enabled {
			continue
		}
		plabel := p.Label
		if plabel == "" {
			plabel = GenVendorLabel(p.Vendor)
		}
		for _, m := range p.Models {
			if !m.Enabled || strings.TrimSpace(m.ID) == "" {
				continue
			}
			ref := FormatLLMModelRef(p.ID, m.ID)
			label := m.Label
			if label == "" || label == m.ID {
				label = m.ID
			}
			out = append(out, EnabledGenModel{
				Ref:           ref,
				ID:            m.ID,
				Label:         label,
				ProviderID:    p.ID,
				ProviderLabel: plabel,
				Vendor:        NormalizeGenVendorID(p.Vendor),
			})
		}
	}
	return out
}

// DefaultGenerationModelID returns providers.generation_default_model as
// profile_id/model_id, or the first enabled gen model ref.
func DefaultGenerationModelID(s Settings) string {
	def := strings.TrimSpace(s.String("providers", "generation_default_model"))
	enabled := ListEnabledGenModels(s)
	if def != "" {
		for _, m := range enabled {
			if m.Ref == def {
				return m.Ref
			}
		}
		// Accept legacy bare model id if unique among enabled.
		if !strings.Contains(def, "/") {
			var hit string
			for _, m := range enabled {
				if m.ID == def {
					if hit != "" && hit != m.Ref {
						hit = ""
						break
					}
					hit = m.Ref
				}
			}
			if hit != "" {
				return hit
			}
		}
	}
	if len(enabled) > 0 {
		return enabled[0].Ref
	}
	return ""
}

// FindGenVendorForModel returns the runtime vendor binding for profile_id/model_id.
func FindGenVendorForModel(s Settings, ref string) (GenVendor, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = DefaultGenerationModelID(s)
	}
	if ref == "" {
		return GenVendor{}, "", fmt.Errorf("no generation model configured")
	}
	providerID, modelID, err := ParseLLMModelRef(ref)
	if err != nil {
		// Legacy bare model: unique match among enabled.
		if !strings.Contains(ref, "/") {
			var matchProvider, matchModel string
			hits := 0
			for _, m := range ListEnabledGenModels(s) {
				if m.ID == ref {
					hits++
					matchProvider, matchModel = m.ProviderID, m.ID
				}
			}
			if hits == 1 {
				providerID, modelID = matchProvider, matchModel
			} else if hits > 1 {
				return GenVendor{}, "", fmt.Errorf("ambiguous bare model %q; use provider_id/model_id", ref)
			} else {
				return GenVendor{}, "", err
			}
		} else {
			return GenVendor{}, "", err
		}
	}
	// profile id is opaque — do not NormalizeGenVendorID (would collapse volcengine-work).
	p, ok := FindGenProfile(s, providerID)
	if !ok {
		return GenVendor{}, "", fmt.Errorf("generation provider %q not found", providerID)
	}
	if !p.Enabled {
		return GenVendor{}, "", fmt.Errorf("generation provider %q is disabled", providerID)
	}
	for _, m := range p.Models {
		if m.Enabled && m.ID == modelID {
			return profileToVendor(p), modelID, nil
		}
	}
	return GenVendor{}, "", fmt.Errorf("model %q is not enabled on provider %q", modelID, providerID)
}

// GenProfilesToAny serializes profiles for settings JSON.
func GenProfilesToAny(profiles []GenProfile) []any {
	out := make([]any, 0, len(profiles))
	for _, p := range profiles {
		models := make([]any, 0, len(p.Models))
		for _, m := range p.Models {
			models = append(models, map[string]any{
				"id": m.ID, "label": m.Label, "enabled": m.Enabled,
			})
		}
		out = append(out, map[string]any{
			"id":       p.ID,
			"label":    p.Label,
			"vendor":   NormalizeGenVendorID(p.Vendor),
			"base_url": p.BaseURL,
			"api_key":  p.APIKey,
			"enabled":  p.Enabled,
			"models":   models,
		})
	}
	return out
}
