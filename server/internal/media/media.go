package media

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
)

// GetDurationMilliseconds extracts the duration of a media file using ffprobe
func GetDurationMilliseconds(filePath string) (int64, error) {
	// Use ffprobe to get duration in seconds with decimal precision
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Parse duration as float (seconds with decimal)
	durationSeconds, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	// Convert to milliseconds
	durationMs := int64(durationSeconds * 1000)

	slog.Debug("Extracted media duration", "file_path", filePath, "duration_seconds", durationSeconds, "duration_milliseconds", durationMs)

	return durationMs, nil
}

// CheckDependencies verifies that ffprobe is available
func CheckDependencies() error {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found in PATH (install ffmpeg)")
	}
	return nil
}
