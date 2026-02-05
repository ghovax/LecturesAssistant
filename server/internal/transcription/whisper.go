package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type WhisperProvider struct {
	model  string
	device string
	prompt string
}

func NewWhisperProvider(model, device string) *WhisperProvider {
	return &WhisperProvider{
		model:  model,
		device: device,
	}
}

func (whisper *WhisperProvider) SetPrompt(prompt string) {
	whisper.prompt = prompt
}

func (whisper *WhisperProvider) Name() string {
	return "whisper-local"
}

func (whisper *WhisperProvider) CheckDependencies() error {
	_, err := exec.LookPath("whisper")
	if err != nil {
		return fmt.Errorf("whisper executable not found in PATH")
	}
	return nil
}

type whisperOutput struct {
	Text     string           `json:"text"`
	Segments []whisperSegment `json:"segments"`
}

type whisperSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func (whisper *WhisperProvider) Transcribe(context context.Context, audioPath string) ([]Segment, error) {
	outputDirectory := filepath.Dir(audioPath)

	arguments := []string{
		audioPath,
		"--model", whisper.model,
		"--output_format", "json",
		"--output_dir", outputDirectory,
	}

	if whisper.prompt != "" {
		arguments = append(arguments, "--initial_prompt", whisper.prompt)
	}

	command := exec.CommandContext(context, "whisper", arguments...)

	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("whisper execution failed: %w", err)
	}
	// Read the JSON output file
	baseName := filepath.Base(audioPath)
	extension := filepath.Ext(baseName)
	jsonFileName := baseName[0:len(baseName)-len(extension)] + ".json"
	jsonPath := filepath.Join(outputDirectory, jsonFileName)

	defer os.Remove(jsonPath) // Cleanup

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read whisper output: %w", err)
	}

	var output whisperOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse whisper output: %w", err)
	}

	var segments []Segment
	for _, whisperSegmentItem := range output.Segments {
		segments = append(segments, Segment{
			Start: whisperSegmentItem.Start,
			End:   whisperSegmentItem.End,
			Text:  whisperSegmentItem.Text,
		})
	}

	return segments, nil
}
