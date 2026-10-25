package gen

// filterMedia keeps rows matching media. Untyped rows (no media field) are kept
// so the user can pick — Film does not invent modality from model id names.
func filterMedia(models []map[string]any, media string) []map[string]any {
	if media == "" {
		return models
	}
	anyTyped := false
	for _, m := range models {
		if strVal(m["media"]) != "" {
			anyTyped = true
			break
		}
	}
	if !anyTyped {
		return models
	}
	var out []map[string]any
	for _, m := range models {
		got := strVal(m["media"])
		if got == "" || got == media {
			out = append(out, m)
		}
	}
	return out
}

func ensureConfiguredModel(models []map[string]any, configured, media, fallbackLabel string) []map[string]any {
	configured = strVal(configured)
	if configured == "" {
		return models
	}
	for _, m := range models {
		if strVal(m["id"]) == configured {
			return models
		}
	}
	row := map[string]any{
		"id": configured, "label": first(fallbackLabel, configured),
	}
	if media != "" {
		row["media"] = media
	}
	return append([]map[string]any{row}, models...)
}
