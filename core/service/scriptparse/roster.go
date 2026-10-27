package scriptparse

import (
	"fmt"
	"strings"
)

func sanitizeDecision(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case DecisionReuse:
		return DecisionReuse
	case DecisionCreate:
		return DecisionCreate
	default:
		return ""
	}
}

func sanitizeScope(s string, decision string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case ScopeSeries:
		return ScopeSeries
	case ScopeEpisode:
		return ScopeEpisode
	}
	if decision == DecisionReuse {
		return ScopeSeries
	}
	if decision == DecisionCreate {
		return ScopeEpisode
	}
	return ""
}

func defaultLookLabel(locale string) string {
	if NormalizeLocale(locale) == "zh" {
		return "默认"
	}
	return "default"
}

func composeCharacterLookBody(lk RegistryLook) string {
	if b := strings.TrimSpace(lk.Body); b != "" {
		return b
	}
	var parts []string
	if h := strings.TrimSpace(lk.Hair); h != "" {
		parts = append(parts, "发型："+h)
	}
	if w := strings.TrimSpace(lk.Wardrobe); w != "" {
		parts = append(parts, "服装："+w)
	}
	if a := strings.TrimSpace(lk.Accessories); a != "" {
		parts = append(parts, "随身物："+a)
	}
	return strings.Join(parts, " | ")
}

func characterHasLookDesc(c RegistryCharacter) bool {
	if strings.TrimSpace(c.LookBody) != "" {
		return true
	}
	for _, lk := range c.Looks {
		if composeCharacterLookBody(lk) != "" {
			return true
		}
	}
	return false
}

func locationHasLookDesc(loc RegistryLocation) bool {
	if strings.TrimSpace(loc.Body) != "" {
		return true
	}
	for _, lk := range loc.Looks {
		if strings.TrimSpace(lk.Body) != "" {
			return true
		}
	}
	return false
}

func primaryLookBody(c RegistryCharacter) string {
	if len(c.Looks) > 0 {
		if b := composeCharacterLookBody(c.Looks[0]); b != "" {
			return b
		}
	}
	return strings.TrimSpace(c.LookBody)
}

func primaryLocationBody(loc RegistryLocation) string {
	if len(loc.Looks) > 0 {
		if b := strings.TrimSpace(loc.Looks[0].Body); b != "" {
			return b
		}
	}
	return strings.TrimSpace(loc.Body)
}

func defaultLookID(entityID string, looks []RegistryLook) string {
	if len(looks) > 0 {
		if id := strings.TrimSpace(looks[0].LookID); id != "" {
			return id
		}
	}
	return entityID + "_default"
}

func parseRegistryLooks(raw []map[string]any, lookBodyFallback, defaultLabel, idPrefix string) []RegistryLook {
	var out []RegistryLook
	for i, m := range raw {
		label := strings.TrimSpace(asString(m["label"]))
		if label == "" {
			label = defaultLabel
		}
		dec := sanitizeDecision(asString(m["decision"]))
		scope := sanitizeScope(asString(m["scope"]), dec)
		existID := strings.TrimSpace(asString(m["existing_look_id"]))
		lookID := strings.TrimSpace(asString(m["look_id"]))
		if lookID == "" && dec == DecisionReuse && existID != "" {
			lookID = existID
		}
		if lookID == "" {
			lookID = fmt.Sprintf("%s%d", idPrefix, i)
		}
		lk := RegistryLook{
			LookID:         lookID,
			Decision:       dec,
			ExistingLookID: existID,
			Scope:          scope,
			Label:          label,
			Hair:           strings.TrimSpace(asString(m["hair"])),
			Wardrobe:       strings.TrimSpace(asString(m["wardrobe"])),
			Accessories:    strings.TrimSpace(asString(m["accessories"])),
			Body:           strings.TrimSpace(asString(m["body"])),
		}
		out = append(out, lk)
	}
	if len(out) == 0 && strings.TrimSpace(lookBodyFallback) != "" {
		out = append(out, RegistryLook{
			LookID:   idPrefix + "0",
			Decision: DecisionCreate,
			Scope:    ScopeEpisode,
			Label:    defaultLabel,
			Wardrobe: strings.TrimSpace(lookBodyFallback),
			Body:     strings.TrimSpace(lookBodyFallback),
		})
	}
	return out
}

func formatExistingRosterBlock(roster *ExistingRoster) string {
	if roster == nil {
		return ""
	}
	var blocks []string
	if len(roster.Characters) > 0 {
		var lines []string
		lines = append(lines, "Existing characters and looks (reuse these ids when the script refers to the same person/outfit):")
		for _, ch := range roster.Characters {
			id := strings.TrimSpace(ch.ID)
			name := strings.TrimSpace(ch.Name)
			if id == "" && name == "" {
				continue
			}
			desc := strings.TrimSpace(ch.VisualIdentity)
			if base := strings.TrimSpace(ch.IdentityBaseline); base != "" {
				if desc != "" {
					desc = base + "；" + desc
				} else {
					desc = base
				}
			}
			head := fmt.Sprintf("- id=%s name=%s", id, name)
			if len(ch.Aliases) > 0 {
				head += fmt.Sprintf(" aliases=%v", ch.Aliases)
			}
			if desc != "" {
				head += " identity=" + desc
			}
			lines = append(lines, head)
			for _, lk := range ch.Looks {
				lid := strings.TrimSpace(lk.ID)
				llabel := strings.TrimSpace(lk.Label)
				lbody := strings.TrimSpace(lk.Body)
				if lid == "" && llabel == "" && lbody == "" {
					continue
				}
				lines = append(lines, fmt.Sprintf("  look id=%s label=%s body=%s", lid, llabel, lbody))
			}
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	if len(roster.Scenes) > 0 {
		var lines []string
		lines = append(lines, "Existing scenes / locations (reuse these ids when the script refers to the same place):")
		for _, sc := range roster.Scenes {
			id := strings.TrimSpace(sc.ID)
			name := strings.TrimSpace(sc.Name)
			if id == "" && name == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("- id=%s name=%s", id, name))
			for _, lk := range sc.Looks {
				lid := strings.TrimSpace(lk.ID)
				llabel := strings.TrimSpace(lk.Label)
				lbody := strings.TrimSpace(lk.Body)
				if lid == "" && llabel == "" && lbody == "" {
					continue
				}
				lines = append(lines, fmt.Sprintf("  look id=%s label=%s body=%s", lid, llabel, lbody))
			}
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	return strings.Join(blocks, "\n\n")
}

func resolveCharacterID(c RegistryCharacter, name string) string {
	if c.Decision == DecisionReuse && strings.TrimSpace(c.ExistingCharacterID) != "" {
		return strings.TrimSpace(c.ExistingCharacterID)
	}
	if id := strings.TrimSpace(c.CharacterID); id != "" {
		return id
	}
	return stableID("char_", name)
}

func resolveLocationID(loc RegistryLocation, name string) string {
	if loc.Decision == DecisionReuse && strings.TrimSpace(loc.ExistingLocationID) != "" {
		return strings.TrimSpace(loc.ExistingLocationID)
	}
	if id := strings.TrimSpace(loc.LocationID); id != "" {
		return id
	}
	return stableID("loc_", name)
}

func normalizeCharacterLooks(c *RegistryCharacter, locale string) {
	label := defaultLookLabel(locale)
	if len(c.Looks) == 0 && strings.TrimSpace(c.LookBody) != "" {
		c.Looks = []RegistryLook{{
			LookID:   c.CharacterID + "_default",
			Decision: DecisionCreate,
			Scope:    ScopeEpisode,
			Label:    label,
			Wardrobe: strings.TrimSpace(c.LookBody),
		}}
	}
	for i := range c.Looks {
		lk := &c.Looks[i]
		if lk.Decision == "" {
			if lk.ExistingLookID != "" {
				lk.Decision = DecisionReuse
			} else {
				lk.Decision = DecisionCreate
			}
		}
		lk.Scope = sanitizeScope(lk.Scope, lk.Decision)
		if lk.Label == "" {
			lk.Label = label
		}
		if lk.Decision == DecisionReuse && lk.ExistingLookID != "" {
			lk.LookID = lk.ExistingLookID
		} else if lk.LookID == "" {
			lk.LookID = fmt.Sprintf("%s_look_%d", c.CharacterID, i)
			if i == 0 {
				lk.LookID = c.CharacterID + "_default"
			}
		}
		if composed := composeCharacterLookBody(*lk); composed != "" {
			lk.Body = composed
		}
	}
	if c.LookBody == "" {
		c.LookBody = primaryLookBody(*c)
	}
}

func normalizeLocationLooks(loc *RegistryLocation, locale string) {
	label := defaultLookLabel(locale)
	if len(loc.Looks) == 0 && strings.TrimSpace(loc.Body) != "" {
		loc.Looks = []RegistryLook{{
			LookID:   loc.LocationID + "_default",
			Decision: DecisionCreate,
			Scope:    ScopeEpisode,
			Label:    label,
			Body:     strings.TrimSpace(loc.Body),
		}}
	}
	for i := range loc.Looks {
		lk := &loc.Looks[i]
		if lk.Decision == "" {
			if lk.ExistingLookID != "" {
				lk.Decision = DecisionReuse
			} else {
				lk.Decision = DecisionCreate
			}
		}
		lk.Scope = sanitizeScope(lk.Scope, lk.Decision)
		if lk.Label == "" {
			lk.Label = label
		}
		if lk.Decision == DecisionReuse && lk.ExistingLookID != "" {
			lk.LookID = lk.ExistingLookID
		} else if lk.LookID == "" {
			lk.LookID = fmt.Sprintf("%s_look_%d", loc.LocationID, i)
			if i == 0 {
				lk.LookID = loc.LocationID + "_default"
			}
		}
	}
	if loc.Body == "" {
		loc.Body = primaryLocationBody(*loc)
	}
}

func episodeOnlyFromScope(scope, decision string) bool {
	if decision == DecisionCreate {
		return true
	}
	return scope == ScopeEpisode
}
