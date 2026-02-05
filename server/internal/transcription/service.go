package transcription

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"lectures/internal/configuration"
	"lectures/internal/llm"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

type Service struct {
	configuration *configuration.Configuration
	ffmpeg        *FFmpeg
	provider      Provider
	llmProvider   llm.Provider
	promptManager *prompts.Manager
}

func NewService(configuration *configuration.Configuration, provider Provider, llmProvider llm.Provider, promptManager *prompts.Manager) *Service {
	return &Service{
		configuration: configuration,
		ffmpeg:        NewFFmpeg(),
		provider:      provider,
		llmProvider:   llmProvider,
		promptManager: promptManager,
	}
}

// CheckDependencies verifies that FFmpeg and the provider are available
func (service *Service) CheckDependencies() error {
	if err := service.ffmpeg.CheckDependencies(); err != nil {
		return err
	}
	return service.provider.CheckDependencies()
}

// TranscribeLecture processes a list of media files and returns a unified list of transcript segments
func (service *Service) TranscribeLecture(jobContext context.Context, mediaFiles []models.LectureMedia, temporaryDirectory string, updateProgress func(int, string, any)) ([]models.TranscriptSegment, error) {
	var allSegments []models.TranscriptSegment
	var globalTimeOffsetMilliseconds int64 = 0

	// Validate FFmpeg
	if err := service.ffmpeg.CheckDependencies(); err != nil {
		return nil, fmt.Errorf("ffmpeg dependency check failed: %w", err)
	}

	// Load transcription instructions
	transcriptionPrompt, err := service.promptManager.GetPrompt(prompts.PromptTranscribeRecording, nil)
	if err == nil {
		service.provider.SetPrompt(transcriptionPrompt)
	}

	totalMediaFiles := len(mediaFiles)

	for mediaIndex, media := range mediaFiles {
		mediaMetadata := map[string]any{
			"media_index": mediaIndex + 1,
			"total_media": totalMediaFiles,
			"media_id":    media.ID,
		}
		updateProgress(int(float64(mediaIndex)/float64(totalMediaFiles)*100), "Preparing media file for transcription...", mediaMetadata)

		// 1. Prepare Audio
		audioPath := filepath.Join(temporaryDirectory, fmt.Sprintf("source_%s.mp3", media.ID))
		if err := service.ffmpeg.ExtractAudio(media.FilePath, audioPath); err != nil {
			return nil, fmt.Errorf("failed to extract audio from %s: %w", media.FilePath, err)
		}

		// 2. Split Audio
		segmentsDirectory := filepath.Join(temporaryDirectory, fmt.Sprintf("segments_%s", media.ID))
		segmentDurationSeconds := 300 // 5 minutes
		segmentFiles, err := service.ffmpeg.SplitAudio(audioPath, segmentsDirectory, segmentDurationSeconds)
		if err != nil {
			return nil, fmt.Errorf("failed to split audio: %w", err)
		}
		sort.Strings(segmentFiles)

		var mediaSegments []models.TranscriptSegment

		// 3. Transcribe Segments in chunks of 3 for cleanup
		totalSegments := len(segmentFiles)
		for segmentChunkStart := 0; segmentChunkStart < totalSegments; segmentChunkStart += 3 {
			segmentChunkEnd := segmentChunkStart + 3
			if segmentChunkEnd > totalSegments {
				segmentChunkEnd = totalSegments
			}

			var chunkSegments []models.TranscriptSegment
			var chunkTextBuilder strings.Builder

			for segmentIndex := segmentChunkStart; segmentIndex < segmentChunkEnd; segmentIndex++ {
				segmentFile := segmentFiles[segmentIndex]

				currentProgress := int((float64(mediaIndex) + float64(segmentIndex)/float64(totalMediaFiles)) / float64(totalMediaFiles) * 100)

				segmentMetadata := map[string]any{
					"media_index":    mediaIndex + 1,
					"segment_index":  segmentIndex + 1,
					"total_segments": totalSegments,
				}
				updateProgress(currentProgress, "Transcribing audio segment...", segmentMetadata)

				results, err := service.provider.Transcribe(jobContext, segmentFile)
				if err != nil {
					return nil, fmt.Errorf("transcription failed for segment %s: %w", segmentFile, err)
				}

				segmentBaseOffsetMilliseconds := int64(segmentIndex) * int64(segmentDurationSeconds) * 1000

				for _, segment := range results {
					originalStart := segmentBaseOffsetMilliseconds + int64(segment.Start*1000)
					originalEnd := segmentBaseOffsetMilliseconds + int64(segment.End*1000)

					chunkSegments = append(chunkSegments, models.TranscriptSegment{
						MediaID:                   media.ID,
						OriginalStartMilliseconds: originalStart,
						OriginalEndMilliseconds:   originalEnd,
						Text:                      segment.Text,
						Confidence:                segment.Confidence,
						Speaker:                   segment.Speaker,
					})
					chunkTextBuilder.WriteString(segment.Text + " ")
				}
			}

			// 4. LLM Cleanup for the chunk
			if chunkTextBuilder.Len() > 0 {
				cleanupProgress := int((float64(mediaIndex) + float64(segmentChunkEnd)/float64(totalSegments)) / float64(totalMediaFiles) * 100)
				updateProgress(cleanupProgress, "Cleaning up and polishing transcripts...", mediaMetadata)

				cleanedText, err := service.cleanupTranscriptChunk(jobContext, chunkTextBuilder.String())
				if err == nil {
					firstSegment := chunkSegments[0]
					lastSegment := chunkSegments[len(chunkSegments)-1]

					mediaSegments = append(mediaSegments, models.TranscriptSegment{
						MediaID:                   media.ID,
						StartMillisecond:          globalTimeOffsetMilliseconds + firstSegment.OriginalStartMilliseconds,
						EndMillisecond:            globalTimeOffsetMilliseconds + lastSegment.OriginalEndMilliseconds,
						OriginalStartMilliseconds: firstSegment.OriginalStartMilliseconds,
						OriginalEndMilliseconds:   lastSegment.OriginalEndMilliseconds,
						Text:                      cleanedText,
						Confidence:                1.0,
					})
				} else {
					// Fallback to original segments if LLM fails
					for _, segment := range chunkSegments {
						segment.StartMillisecond = globalTimeOffsetMilliseconds + segment.OriginalStartMilliseconds
						segment.EndMillisecond = globalTimeOffsetMilliseconds + segment.OriginalEndMilliseconds
						mediaSegments = append(mediaSegments, segment)
					}
				}
			}
		}

		allSegments = append(allSegments, mediaSegments...)

		durationSeconds, err := service.ffmpeg.GetDuration(audioPath)
		if err != nil {
			durationSeconds = float64(len(segmentFiles) * segmentDurationSeconds)
		}
		globalTimeOffsetMilliseconds += int64(durationSeconds * 1000)

		os.Remove(audioPath)
		os.RemoveAll(segmentsDirectory)
	}

	return allSegments, nil
}

func (service *Service) cleanupTranscriptChunk(jobContext context.Context, rawText string) (string, error) {
	latexInstructions, _ := service.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	cleanupPrompt, err := service.promptManager.GetPrompt(prompts.PromptCleanTranscript, map[string]string{
		"transcript":         rawText,
		"latex_instructions": latexInstructions,
	})
	if err != nil {
		return "", err
	}

	model := service.configuration.LLM.OpenRouter.DefaultModel
	if service.configuration.LLM.Provider == "ollama" {
		model = service.configuration.LLM.Ollama.DefaultModel
	}

	responseChannel, err := service.llmProvider.Chat(jobContext, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: cleanupPrompt}}},
		},
	})
	if err != nil {
		return "", err
	}

	var cleanedTextBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return "", chunk.Error
		}
		cleanedTextBuilder.WriteString(chunk.Text)
	}

	return cleanedTextBuilder.String(), nil
}
