package transcription

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FFmpeg handles media processing using the ffmpeg CLI tool
type FFmpeg struct{}

// NewFFmpeg creates a new FFmpeg handler
func NewFFmpeg() *FFmpeg {
	return &FFmpeg{}
}

// CheckDependencies verifies that ffmpeg is installed
func (ffmpeg *FFmpeg) CheckDependencies() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	return nil
}

// ExtractAudio extracts the audio track from a video file to an audio file (mp3)
func (ffmpeg *FFmpeg) ExtractAudio(inputPath string, outputPath string) error {
	// ffmpeg -y -i input.mp4 -vn -acodec libmp3lame -q:a 2 output.mp3
	command := exec.Command("ffmpeg", "-y", "-i", inputPath, "-vn", "-acodec", "libmp3lame", "-q:a", "2", outputPath)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("ffmpeg extract failed: %v, stderr: %s", err, stderr.String())
	}
	return nil
}

// SplitAudio splits an audio file into segments of a specified duration (in seconds)
// Returns the list of generated segment file paths
func (ffmpeg *FFmpeg) SplitAudio(inputPath string, outputDirectory string, segmentDuration int) ([]string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Output pattern: segment_001.mp3
	outputPattern := filepath.Join(outputDirectory, "segment_%03d.mp3")

	// ffmpeg -y -i input.mp3 -f segment -segment_time 600 -c copy output_%03d.mp3
	command := exec.Command("ffmpeg", "-y", "-i", inputPath, "-f", "segment", "-segment_time", strconv.Itoa(segmentDuration), "-c", "copy", outputPattern)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg split failed: %v, stderr: %s", err, stderr.String())
	}

	// List generated files
	segmentFiles, err := filepath.Glob(filepath.Join(outputDirectory, "segment_*.mp3"))
	if err != nil {
		return nil, err
	}
	return segmentFiles, nil
}

// GetDuration returns the duration of the media file in seconds
func (ffmpeg *FFmpeg) GetDuration(inputPath string) (float64, error) {
	// ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 input.mp4
	command := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v, stderr: %s", err, stderr.String())
	}

	durationString := strings.TrimSpace(stdout.String())
	duration, err := strconv.ParseFloat(durationString, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}
	return duration, nil
}