package config

import (
	"fmt"
	"strings"
)

// KnownImageVendors are Image Provider presets (adapter kinds).
var KnownImageVendors = []string{"danmo-make", "volcengine", "dashscope", "minimax", "openai"}

// ImageProfile is one image provider instance (same shape as GenProfile / LLM).
type ImageProfile = GenProfile

// ImageVendorLabel returns a short default display name for an image preset.
func ImageVendorLabel(id string) string {
	switch NormalizeImageVendorID(id) {
	case "volcengine":
		return "火山引擎"
	case "dashscope":
		return "阿里云"
	case "minimax":
		return "MiniMax"
	case "danmo-make":
		return "Danmo Make"
	case "openai":
		return "OpenAI"
	default:
		return id
	}
}

// ImageVendorDefaultBaseURL returns the Settings default Base URL for an image preset.
func ImageVendorDefaultBaseURL(vendor string) string {
	switch NormalizeImageVendorID(vendor) {
	case "danmo-make":
		return "http://127.0.0.1:7800"
	case "volcengine":
		return "https://ark.cn-beijing.volces.com"
	case "dashscope":
		return "https://dashscope.aliyuncs.com"
	case "minimax":
		return "https://api.minimax.chat"
	case "openai":
		return "https://api.openai.com/v1"
	default:
		return ""
	}
}

// IsKnownImageVendor reports whether id is a known image adapter preset.
func IsKnownImageVendor(id string) bool {
	id = NormalizeImageVendorID(id)
	for _, k := range KnownImageVendors {
		if k == id {
			return true
		}
	}
	return false
}

// NormalizeImageVendorID maps aliases to a known image vendor preset id.
func NormalizeImageVendorID(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "volcengine", "bytedance", "seedream", "ark", "火山引擎":
		return "volcengine"
	case "dashscope", "alibaba", "ali", "wanxiang", "wanx", "阿里云":
		return "dashscope"
	case "minimax", "hailuo", "minimax hailuo":
		return "minimax"
	case "danmo-make", "make", "danqing", "danmo make":
		return "danmo-make"
	case "openai", "dall-e", "dalle", "gpt-image":
		return "openai"
	default:
		return strings.TrimSpace(raw)
	}
}

// ImageProfiles returns configured image provider instances.
func ImageProfiles(s Settings) []ImageProfile {
	raw := s.Map("providers")["image_profiles"]
	if profiles, ok := parseImageProfileObjects(raw); ok && len(profiles) > 0 {
		return profiles
	}
	return nil
}

func parseImageProfileObjects(raw any) ([]ImageProfile, bool) {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil, false
	}
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
	out := make([]ImageProfile, 0, len(arr))
	seen := map[string]bool{}
	for _, v := range arr {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		p := parseImageProfileMap(m)
		if p.ID == "" || seen[p.ID] {
			continue
		}
		if !IsKnownImageVendor(p.Vendor) {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out, len(out) > 0
}

func parseImageProfileMap(m map[string]any) ImageProfile {
	id := strings.TrimSpace(strAny(m["id"]))
	vendor := NormalizeImageVendorID(FirstNonEmpty(strAny(m["vendor"]), strAny(m["preset_id"]), id))
	label := strings.TrimSpace(strAny(m["label"]))
	if label == "" {
		label = ImageVendorLabel(vendor)
	}
	p := ImageProfile{
		ID:      id,
		Label:   label,
		Vendor:  vendor,
		BaseURL: strings.TrimSpace(strAny(m["base_url"])),
		APIKey:  SanitizeAPIKey(strAny(m["api_key"])),
		Enabled: boolAny(m["enabled"], true),
		Models:  parseGenModels(m),
	}
	if p.BaseURL == "" {
		p.BaseURL = ImageVendorDefaultBaseURL(vendor)
	}
	if len(p.Models) == 0 {
		legacy := strings.TrimSpace(strAny(m["model"]))
		if legacy != "" {
			p.Models = []GenModel{{ID: legacy, Label: legacy, Enabled: true}}
		}
	}
	return p
}

// FindImageProfile returns an image profile by instance id.
func FindImageProfile(s Settings, profileID string) (ImageProfile, bool) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return ImageProfile{}, false
	}
	for _, p := range ImageProfiles(s) {
		if p.ID == profileID {
			return p, true
		}
	}
	return ImageProfile{}, false
}

// ListEnabledImageModels returns picker refs for enabled image profiles × models.
func ListEnabledImageModels(s Settings) []EnabledGenModel {
	var out []EnabledGenModel
	for _, p := range ImageProfiles(s) {
		if !p.Enabled {
			continue
		}
		plabel := p.Label
		if plabel == "" {
			plabel = ImageVendorLabel(p.Vendor)
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
				Vendor:        NormalizeImageVendorID(p.Vendor),
			})
		}
	}
	return out
}

// DefaultImageModelID returns providers.image_default_model as profile_id/model_id.
func DefaultImageModelID(s Settings) string {
	def := strings.TrimSpace(s.String("providers", "image_default_model"))
	enabled := ListEnabledImageModels(s)
	if def != "" {
		for _, m := range enabled {
			if m.Ref == def {
				return m.Ref
			}
		}
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

// FindImageVendorForModel returns the runtime binding for an image profile_id/model_id.
func FindImageVendorForModel(s Settings, ref string) (GenVendor, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = DefaultImageModelID(s)
	}
	if ref == "" {
		return GenVendor{}, "", fmt.Errorf("no image model configured")
	}
	providerID, modelID, err := ParseLLMModelRef(ref)
	if err != nil {
		if !strings.Contains(ref, "/") {
			var matchProvider, matchModel string
			hits := 0
			for _, m := range ListEnabledImageModels(s) {
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
	p, ok := FindImageProfile(s, providerID)
	if !ok {
		return GenVendor{}, "", fmt.Errorf("image provider %q not found", providerID)
	}
	if !p.Enabled {
		return GenVendor{}, "", fmt.Errorf("image provider %q is disabled", providerID)
	}
	for _, m := range p.Models {
		if m.Enabled && m.ID == modelID {
			return GenVendor{
				ID:      p.ID,
				Vendor:  NormalizeImageVendorID(p.Vendor),
				Enabled: p.Enabled,
				BaseURL: p.BaseURL,
				APIKey:  p.APIKey,
				Models:  p.Models,
			}, modelID, nil
		}
	}
	return GenVendor{}, "", fmt.Errorf("model %q is not enabled on image provider %q", modelID, providerID)
}

// ImageProfilesToAny serializes image profiles for settings JSON.
func ImageProfilesToAny(profiles []ImageProfile) []any {
	return GenProfilesToAny(profiles)
}
