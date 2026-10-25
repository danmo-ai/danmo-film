package assemble

import (
	"fmt"
	"os"
	"os/exec"
)

// ExtractLastFrame writes the last video frame to outImagePath (PNG).
func ExtractLastFrame(videoPath, outImagePath string) error {
	if videoPath == "" {
		return fmt.Errorf("assemble: empty video path")
	}
	if _, err := os.Stat(videoPath); err != nil {
		return fmt.Errorf("assemble: video missing: %w", err)
	}
	ffmpeg, err := RequireFFmpeg()
	if err != nil {
		return err
	}
	cmd := exec.Command(ffmpeg, "-y", "-hide_banner", "-loglevel", "error",
		"-sseof", "-0.05", "-i", videoPath, "-frames:v", "1", outImagePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg extract last frame failed: %s", truncate(string(out), 800))
	}
	if _, err := os.Stat(outImagePath); err != nil {
		return fmt.Errorf("assemble: last frame output missing: %s", outImagePath)
	}
	return nil
}
