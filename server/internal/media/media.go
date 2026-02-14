package media

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// ResolveBinaryPath looks for a binary in the configured bin directory or the system PATH
func ResolveBinaryPath(binName, configuredBinDir string) string {
	if configuredBinDir != "" {
		// Clean the path and expand home if needed
		if len(configuredBinDir) > 0 && configuredBinDir[0] == '~' {
			home, _ := os.UserHomeDir()
			configuredBinDir = filepath.Join(home, configuredBinDir[1:])
		}

		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		localPath := filepath.Join(configuredBinDir, binName+ext)
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
	}

	// Fallback to system PATH
	path, err := exec.LookPath(binName)
	if err == nil {
		return path
	}

	return binName // Return name and let exec.Command fail later if not found
}

// GetDurationMilliseconds extracts the duration of a media file using ffprobe
func GetDurationMilliseconds(filePath string) (int64, error) {
	// We use a dummy dir here as this static call doesn't have config context
	// In the real service, we'll use the resolved path.
	binPath := ResolveBinaryPath("ffprobe", "")

	// Use ffprobe to get duration in seconds with decimal precision
	cmd := exec.Command(binPath,
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
func CheckDependencies(binDir string) error {
	path := ResolveBinaryPath("ffprobe", binDir)
	if _, err := exec.LookPath(path); err != nil {
		return fmt.Errorf("ffprobe not found (install ffmpeg or place in bin folder)")
	}
	return nil
}
