package assemble

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RequireFFmpeg() (string, error) {
	exe, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found on PATH (required to assemble long-video segments)")
	}
	return exe, nil
}

func StitchSegmentPaths(segmentPaths []string, output string, overlapFrames int, fps float64) error {
	_, _ = overlapFrames, fps
	if len(segmentPaths) == 0 {
		return fmt.Errorf("assemble: no segment videos")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if len(segmentPaths) == 1 {
		return copyFile(segmentPaths[0], output)
	}
	ffmpeg, err := RequireFFmpeg()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "film_stitch_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	listPath := filepath.Join(tmp, "concat.txt")
	var lines []string
	for _, p := range segmentPaths {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("assemble: missing segment file %s", p)
		}
		abs, _ := filepath.Abs(p)
		esc := strings.ReplaceAll(abs, "'", `'\''`)
		lines = append(lines, "file '"+esc+"'")
	}
	if err := os.WriteFile(listPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	cmd := exec.Command(ffmpeg, "-y", "-hide_banner", "-loglevel", "error",
		"-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", output)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg stitch failed: %s", truncate(string(out), 800))
	}
	if _, err := os.Stat(output); err != nil {
		return fmt.Errorf("assemble: output missing after ffmpeg: %s", output)
	}
	return nil
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
