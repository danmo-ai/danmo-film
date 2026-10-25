package gen

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func sleepPoll() { time.Sleep(2 * time.Second) }

func writeHTTPFile(resp *http.Response, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func downloadURL(client *http.Client, url, dest string, headers map[string]string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed (%d): %s", resp.StatusCode, truncate(string(b), 300))
	}
	return writeHTTPFile(resp, dest)
}

func taskEnvelope(id, status, kind string, extra map[string]any) map[string]any {
	m := map[string]any{"id": id, "status": status, "kind": kind}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func nestedOrFlatID(m map[string]any) string {
	if id := strVal(m["id"]); id != "" {
		return id
	}
	if t, ok := m["task"].(map[string]any); ok {
		return strVal(t["id"])
	}
	return strVal(m["task_id"])
}

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func filepathBase(p string) string {
	return filepath.Base(p)
}

func writeDataURIFile(dataURI, dest string) error {
	const prefix = "base64,"
	i := strings.Index(dataURI, prefix)
	if i < 0 {
		return fmt.Errorf("unsupported data uri")
	}
	raw, err := base64.StdEncoding.DecodeString(dataURI[i+len(prefix):])
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, raw, 0o644)
}

func fileDataURI(path, mime string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}
