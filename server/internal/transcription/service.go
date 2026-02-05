package transcription

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	config "lectures/internal/configuration"
	"lectures/internal/models"
)

type Service struct {
	configuration *config.Configuration
	ffmpeg        *FFmpeg
	provider      Provider
}

func NewService(configuration *config.Configuration, provider Provider) *Service {
	return &Service{
		configuration: configuration,
		ffmpeg:        NewFFmpeg(),
		provider:      provider,
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
func (service *Service) TranscribeLecture(context context.Context, mediaFiles []models.LectureMedia, temporaryDirectory string, updateProgress func(int, string)) ([]models.TranscriptSegment, error) {
	var allSegments []models.TranscriptSegment
	var globalTimeOffsetMilliseconds int64 = 0

	// Validate FFmpeg
	if err := service.ffmpeg.CheckDependencies(); err != nil {
		return nil, fmt.Errorf("ffmpeg dependency check failed: %w", err)
	}

	totalMediaFiles := len(mediaFiles)

	for mediaIndex, media := range mediaFiles {
		updateProgress(int(float64(mediaIndex)/float64(totalMediaFiles)*100), fmt.Sprintf("Processing media file %d/%d...", mediaIndex+1, totalMediaFiles))

		// 1. Prepare Audio (Extract if needed, or just copy)
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

		// Sort segment files to ensure order (segment_000.mp3, segment_001.mp3...)
		sort.Strings(segmentFiles)

		var mediaSegments []models.TranscriptSegment

		// 3. Transcribe Segments
		totalSegments := len(segmentFiles)
		for segmentIndex, segmentFile := range segmentFiles {
			// Calculate progress within this media file
			currentProgress := int((float64(mediaIndex) + float64(segmentIndex)/float64(totalSegments)) / float64(totalMediaFiles) * 100)
			updateProgress(currentProgress, fmt.Sprintf("Transcribing segment %d/%d of media %d...", segmentIndex+1, totalSegments, mediaIndex+1))

			// Transcribe
			results, err := service.provider.Transcribe(context, segmentFile)
			if err != nil {
				return nil, fmt.Errorf("transcription failed for segment %s: %w", segmentFile, err)
			}

			// 4. Adjust Timestamps
			segmentBaseOffsetMilliseconds := int64(segmentIndex) * int64(segmentDurationSeconds) * 1000

			for _, segment := range results {
				startMilliseconds := int64(segment.Start * 1000)
				endMilliseconds := int64(segment.End * 1000)

				// Time relative to the specific media file
				originalStart := segmentBaseOffsetMilliseconds + startMilliseconds
				originalEnd := segmentBaseOffsetMilliseconds + endMilliseconds

				// Time relative to the entire lecture (unified transcript)
				globalStart := globalTimeOffsetMilliseconds + originalStart
				globalEnd := globalTimeOffsetMilliseconds + originalEnd

				mediaSegments = append(mediaSegments, models.TranscriptSegment{
					MediaID:                   media.ID,
					StartMillisecond:          globalStart,
					EndMillisecond:            globalEnd,
					OriginalStartMilliseconds: originalStart,
					OriginalEndMilliseconds:   originalEnd,
					Text:                      segment.Text,
					Confidence:                segment.Confidence,
					Speaker:                   segment.Speaker,
				})
			}
		}

		allSegments = append(allSegments, mediaSegments...)

		// Update global offset by adding the duration of this media file
		durationSeconds, err := service.ffmpeg.GetDuration(audioPath)
		if err != nil {
			// Fallback to estimation from segments
			durationSeconds = float64(len(segmentFiles) * segmentDurationSeconds)
		}
		globalTimeOffsetMilliseconds += int64(durationSeconds * 1000)

		// Cleanup temp files for this media
		os.Remove(audioPath)
		os.RemoveAll(segmentsDirectory)
	}

	return allSegments, nil
}
