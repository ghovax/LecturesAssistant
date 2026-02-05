package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles loading and templating prompts
type Manager struct {
	baseDirectory string
}

// NewManager creates a new prompt manager
func NewManager(baseDirectory string) *Manager {
	return &Manager{
		baseDirectory: baseDirectory,
	}
}

// GetPrompt loads a prompt file and performs variable replacements
func (manager *Manager) GetPrompt(promptPath string, variables map[string]string) (string, error) {
	fullPath := filepath.Join(manager.baseDirectory, promptPath)
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	content := string(contentBytes)
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}

// Prompt constants for easier access
const (
	PromptAnalyzeLectureStructure        = "general/analyze-lecture-structure.md"
	PromptCleanDocumentTitle             = "general/clean-document-title.md"
	PromptCleanTranscript                = "general/clean-transcript.md"
	PromptCorrectProjectTitleDescription = "general/correct-project-title-description.md"
	PromptCorrectUserMessage             = "general/correct-user-message.md"
	PromptFormatFootnotes                = "general/format-footnotes.md"
	PromptGenerateChatQuestions          = "general/generate-chat-questions.md"
	PromptGenerateDocumentDescription    = "general/generate-document-description.md"
	PromptGenerateDocumentIcon           = "general/generate-document-icon.md"
	PromptGenerateProjectIcon            = "general/generate-project-icon.md"
	PromptGetRelevantPages               = "general/get-relevant-pages.md"
	PromptParseFootnotes                 = "general/parse-footnotes.md"
	PromptReadingAssistantMultiChat      = "general/reading-assistant-multi-chat.md"
	PromptStyleConcise                   = "general/style-concise.md"
	PromptStyleLearning                  = "general/style-learning.md"
	PromptStyleNormal                    = "general/style-normal.md"
	PromptVerifySectionAdherence         = "general/verify-section-adherence.md"

	PromptIngestDocumentPage  = "media/ingest-document-page.md"
	PromptTextToSpeechSection = "media/text-to-speech-section.md"
	PromptTranscribeRecording = "media/transcribe-recording.md"

	PromptCitationInstructions              = "study-guides/citation-instructions.md"
	PromptStudyGuideWithCitationsExample    = "study-guides/study-guide-with-citations-example.md"
	PromptStudyGuideWithoutCitationsExample = "study-guides/study-guide-without-citations-example.md"
	PromptGenerateFlashcards                = "study-guides/generate-flashcards.md"
	PromptGenerateQuiz                      = "study-guides/generate-quiz.md"
	PromptLanguageRequirement               = "study-guides/language-requirement.md"
	PromptLatexInstructions                 = "study-guides/latex-instructions.md"
	PromptSectionWithCitationsExample       = "study-guides/section-with-citations-example.md"
	PromptSectionWithoutCitationsExample    = "study-guides/section-without-citations-example.md"
	PromptStudyGuideInitialContext          = "study-guides/study-guide-initial-context.md"
	PromptStudyGuideSectionGeneration       = "study-guides/study-guide-section-generation.md"
)
