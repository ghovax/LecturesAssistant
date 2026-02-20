package transcription

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"lectures/internal/configuration"
	"lectures/internal/llm"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

type Service struct {
	configuration  *configuration.Configuration
	mediaProcessor MediaProcessor
	provider       Provider
	llmProvider    llm.Provider
	promptManager  *prompts.Manager
}

func NewService(configuration *configuration.Configuration, provider Provider, llmProvider llm.Provider, promptManager *prompts.Manager) *Service {
	return &Service{
		configuration:  configuration,
		mediaProcessor: NewFFmpeg(configuration.Storage.BinDirectory),
		provider:       provider,
		llmProvider:    llmProvider,
		promptManager:  promptManager,
	}
}

// SetMediaProcessor allows overriding the default media processor (useful for testing)
func (service *Service) SetMediaProcessor(processor MediaProcessor) {
	service.mediaProcessor = processor
}

// CheckDependencies verifies that FFmpeg and the provider are available
func (service *Service) CheckDependencies() error {
	if err := service.mediaProcessor.CheckDependencies(); err != nil {
		return err
	}
	return service.provider.CheckDependencies()
}

// TranscribeLecture processes a list of media files and returns a unified list of transcript segments
func (service *Service) TranscribeLecture(jobContext context.Context, mediaFiles []models.LectureMedia, temporaryDirectory string, updateProgress func(int, string, any)) ([]models.TranscriptSegment, models.JobMetrics, error) {
	var allSegments []models.TranscriptSegment
	var globalTimeOffsetMilliseconds int64 = 0
	var totalMetrics models.JobMetrics

	// Validate FFmpeg
	if err := service.mediaProcessor.CheckDependencies(); err != nil {
		return nil, totalMetrics, fmt.Errorf("ffmpeg dependency check failed: %w", err)
	}

	// Load transcription instructions
	if service.promptManager != nil {
		transcriptionPrompt, err := service.promptManager.GetPrompt(prompts.PromptTranscribeRecording, nil)
		if err == nil {
			service.provider.SetPrompt(transcriptionPrompt)
		}
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
		if extractionError := service.mediaProcessor.ExtractAudio(media.FilePath, audioPath); extractionError != nil {
			return nil, totalMetrics, fmt.Errorf("failed to extract audio from %s: %w", media.FilePath, extractionError)
		}

		// 2. Split Audio
		segmentsDirectory := filepath.Join(temporaryDirectory, fmt.Sprintf("segments_%s", media.ID))
		segmentDurationSeconds := service.configuration.Transcription.AudioChunkLengthSeconds
		if segmentDurationSeconds <= 0 {
			segmentDurationSeconds = 300
		}
		segmentFiles, splitError := service.mediaProcessor.SplitAudio(audioPath, segmentsDirectory, segmentDurationSeconds)
		if splitError != nil {
			return nil, totalMetrics, fmt.Errorf("failed to split audio: %w", splitError)
		}
		sort.Strings(segmentFiles)

		var mediaSegments []models.TranscriptSegment

		// 3. Transcribe Segments in chunks for cleanup
		totalSegments := len(segmentFiles)
		batchSize := service.configuration.Transcription.RefiningBatchSize
		if batchSize <= 0 {
			batchSize = 3
		}

		for segmentChunkStart := 0; segmentChunkStart < totalSegments; segmentChunkStart += batchSize {
			segmentChunkEnd := segmentChunkStart + batchSize
			if segmentChunkEnd > totalSegments {
				segmentChunkEnd = totalSegments
			}

			type segmentResult struct {
				index    int
				segments []models.TranscriptSegment
				text     string
				metrics  models.JobMetrics
				err      error
			}

			resultChan := make(chan segmentResult, segmentChunkEnd-segmentChunkStart)
			var wg sync.WaitGroup

			// Concurrency limit for transcription (e.g., 5 concurrent segments)
			semaphore := make(chan struct{}, 5)

			for segmentIndex := segmentChunkStart; segmentIndex < segmentChunkEnd; segmentIndex++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					select {
					case semaphore <- struct{}{}:
						defer func() { <-semaphore }()
					case <-jobContext.Done():
						return
					}

					segmentFile := segmentFiles[idx]

					transcriptionResults, stepMetrics, transcriptionError := service.provider.Transcribe(jobContext, segmentFile)
					if transcriptionError != nil {
						resultChan <- segmentResult{err: transcriptionError}
						return
					}

					// Get actual segment duration as fallback
					actualSegmentDuration, _ := service.mediaProcessor.GetDuration(segmentFile)
					segmentBaseOffsetMilliseconds := int64(idx) * int64(segmentDurationSeconds) * 1000

					var segs []models.TranscriptSegment
					var textBuilder strings.Builder
					for _, transcriptSegment := range transcriptionResults {
						originalStart := segmentBaseOffsetMilliseconds + int64(transcriptSegment.Start*1000)
						endSeconds := transcriptSegment.End
						if endSeconds == 0 && actualSegmentDuration > 0 {
							endSeconds = actualSegmentDuration
						}
						originalEnd := segmentBaseOffsetMilliseconds + int64(endSeconds*1000)

						segs = append(segs, models.TranscriptSegment{
							MediaID:                   media.ID,
							OriginalStartMilliseconds: originalStart,
							OriginalEndMilliseconds:   originalEnd,
							Text:                      transcriptSegment.Text,
							Confidence:                transcriptSegment.Confidence,
							Speaker:                   transcriptSegment.Speaker,
						})
						textBuilder.WriteString(transcriptSegment.Text + " ")
					}

					resultChan <- segmentResult{
						index:    idx,
						segments: segs,
						text:     textBuilder.String(),
						metrics:  stepMetrics,
					}
				}(segmentIndex)
			}

			wg.Wait()
			close(resultChan)

			// Collect and sort results
			var results []segmentResult
			for res := range resultChan {
				if res.err != nil {
					return nil, totalMetrics, fmt.Errorf("transcription failed: %w", res.err)
				}
				results = append(results, res)
			}
			sort.Slice(results, func(i, j int) bool {
				return results[i].index < results[j].index
			})

			var chunkSegments []models.TranscriptSegment
			var chunkTextBuilder strings.Builder

			for _, res := range results {
				chunkSegments = append(chunkSegments, res.segments...)
				chunkTextBuilder.WriteString(res.text)
				totalMetrics.InputTokens += res.metrics.InputTokens
				totalMetrics.OutputTokens += res.metrics.OutputTokens
				totalMetrics.EstimatedCost += res.metrics.EstimatedCost
			}

			// Update progress
			currentProgress := int((float64(mediaIndex) + float64(segmentChunkEnd)/float64(totalSegments)) / float64(totalMediaFiles) * 100)
			updateProgress(currentProgress, "Transcribing audio segments...", mediaMetadata)

			// 4. LLM Cleanup for the chunk
			if chunkTextBuilder.Len() > 0 {
				cleanupProgress := int((float64(mediaIndex) + float64(segmentChunkEnd)/float64(totalSegments)) / float64(totalMediaFiles) * 100)
				updateProgress(cleanupProgress, "Cleaning up and polishing transcripts...", mediaMetadata)

				cleanedText, cleanupMetrics, cleanupError := service.cleanupTranscriptChunk(jobContext, chunkTextBuilder.String())
				totalMetrics.InputTokens += cleanupMetrics.InputTokens
				totalMetrics.OutputTokens += cleanupMetrics.OutputTokens
				totalMetrics.EstimatedCost += cleanupMetrics.EstimatedCost

				if cleanupError == nil {
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

		durationSeconds, durationError := service.mediaProcessor.GetDuration(audioPath)
		if durationError != nil {
			durationSeconds = float64(len(segmentFiles) * segmentDurationSeconds)
		}
		globalTimeOffsetMilliseconds += int64(durationSeconds * 1000)

		os.Remove(audioPath)
		os.RemoveAll(segmentsDirectory)
	}

	return allSegments, totalMetrics, nil
}

func (service *Service) cleanupTranscriptChunk(jobContext context.Context, rawText string) (string, models.JobMetrics, error) {
	var metrics models.JobMetrics
	if service.promptManager == nil {
		return rawText, metrics, nil
	}

	latexInstructions, _ := service.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	// Inject a strong language preservation instruction
	languagePreservation := "Mandatory: You MUST preserve the original language(s) of the transcript. Do NOT translate. If the text is in Italian, it must stay in Italian. If it is in English, it must stay in English. If it is mixed, preserve the mix."

	cleanupPrompt, promptError := service.promptManager.GetPrompt(prompts.PromptCleanTranscript, map[string]string{
		"transcript":           rawText,
		"latex_instructions":   latexInstructions,
		"language_requirement": languagePreservation,
	})
	if promptError != nil {
		return "", metrics, promptError
	}

	// Use content_polishing model if configured, otherwise fallback to default
	model := service.configuration.LLM.GetModelForTask("content_polishing")
	if model == "" {
		model = service.configuration.LLM.Model
	}

	// Set max_tokens to 16384 to prevent truncation of long transcript segments
	const maxTokens = 16384

	responseChannel, chatError := service.llmProvider.Chat(jobContext, &llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: cleanupPrompt}}},
		},
		MaxTokens: maxTokens,
	})
	if chatError != nil {
		return "", metrics, chatError
	}

	var cleanedTextBuilder strings.Builder
	for responseChunk := range responseChannel {
		if responseChunk.Error != nil {
			return "", metrics, responseChunk.Error
		}
		cleanedTextBuilder.WriteString(responseChunk.Text)
		metrics.InputTokens += responseChunk.InputTokens
		metrics.OutputTokens += responseChunk.OutputTokens
		metrics.EstimatedCost += responseChunk.Cost
	}

	return cleanedTextBuilder.String(), metrics, nil
}
