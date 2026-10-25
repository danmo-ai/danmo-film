package gen

import (
	"fmt"
	"net/url"
	"strings"
)

func parseOpenAIStyleModels(m map[string]any, media string) ([]map[string]any, error) {
	data, ok := m["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected models list payload (missing data[])")
	}
	var out []map[string]any
	for _, raw := range data {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := first(strVal(row["id"]), strVal(row["model"]))
		if id == "" {
			continue
		}
		item := map[string]any{
			"id":    id,
			"label": first(strVal(row["name"]), strVal(row["label"]), id),
		}
		// Do not invent media tags — leave untyped so Settings can show everything.
		out = append(out, item)
	}
	return filterMedia(out, media), nil
}

func parseMiniMaxModels(m map[string]any, media string) ([]map[string]any, error) {
	if data, ok := m["data"].([]any); ok {
		return parseOpenAIStyleModels(map[string]any{"data": data}, media)
	}
	if models, ok := m["models"].([]any); ok {
		return parseOpenAIStyleModels(map[string]any{"data": models}, media)
	}
	if data, ok := m["data"].(map[string]any); ok {
		if models, ok := data["models"].([]any); ok {
			return parseOpenAIStyleModels(map[string]any{"data": models}, media)
		}
	}
	return nil, fmt.Errorf("unexpected minimax models list payload")
}

func dashscopeModelsQuery(media string, pageNo, pageSize int) string {
	q := url.Values{}
	q.Set("page_no", fmt.Sprintf("%d", pageNo))
	q.Set("page_size", fmt.Sprintf("%d", pageSize))
	q.Set("language", "zh-CN")
	// Only add capability filter when caller asks — otherwise return the full page.
	switch media {
	case "image":
		q.Add("capabilities", "IG")
	case "video":
		q.Add("capabilities", "VG")
	}
	return q.Encode()
}

func parseDashScopeModelsPage(m map[string]any, media string) (items []map[string]any, total int, err error) {
	outp, _ := m["output"].(map[string]any)
	if outp == nil {
		outp = m
	}
	total = int(asFloat(outp["total"]))
	arr, _ := outp["models"].([]any)
	if arr == nil {
		arr, _ = m["models"].([]any)
	}
	for _, raw := range arr {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := first(strVal(row["model"]), strVal(row["id"]))
		if id == "" {
			continue
		}
		label := first(strVal(row["name"]), id)
		item := map[string]any{"id": id, "label": label}
		caps, _ := row["capabilities"].([]any)
		med := ""
		for _, c := range caps {
			switch strings.ToUpper(fmt.Sprint(c)) {
			case "IG":
				med = "image"
			case "VG":
				med = "video"
			}
		}
		if med == "" && media != "" {
			// Request already filtered by capability; accept untagged rows.
			med = media
		}
		if media != "" && med != "" && med != media {
			continue
		}
		if med != "" {
			item["media"] = med
		}
		items = append(items, item)
	}
	return items, total, nil
}
