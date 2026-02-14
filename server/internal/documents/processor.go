package documents

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lectures/internal/llm"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

type Processor struct {
	llmProvider   llm.Provider
	llmModel      string
	promptManager *prompts.Manager
	converter     DocumentConverter
	dpi           int
	binDir        string
}

func NewProcessor(llmProvider llm.Provider, llmModel string, promptManager *prompts.Manager, dpi int, binDir string) *Processor {
	return &Processor{
		llmProvider:   llmProvider,
		llmModel:      llmModel,
		promptManager: promptManager,
		converter:     &ExternalDocumentConverter{binDir: binDir},
		dpi:           dpi,
		binDir:        binDir,
	}
}

// SetConverter allows overriding the default converter (useful for testing)
func (processor *Processor) SetConverter(converter DocumentConverter) {
	processor.converter = converter
}

// CheckDependencies verifies that Ghostscript and LibreOffice are installed
func (processor *Processor) CheckDependencies() error {
	return processor.converter.CheckDependencies()
}

// ProcessDocument extracts pages as images and performs interpretation using a Vision LLM
func (processor *Processor) ProcessDocument(jobContext context.Context, document models.ReferenceDocument, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, models.JobMetrics, error) {
	var metrics models.JobMetrics
	if directoryError := os.MkdirAll(outputDirectory, 0755); directoryError != nil {
		return nil, metrics, fmt.Errorf("failed to create output directory: %w", directoryError)
	}

	extension := strings.ToLower(filepath.Ext(document.FilePath))
	var pdfPath string

	switch extension {
	case ".pdf":
		pdfPath = document.FilePath
	case ".pptx", ".docx":
		updateProgress(5, "Converting document to PDF...")
		temporaryPdfPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.pdf", document.ID))
		if conversionError := processor.converter.ConvertToPDF(document.FilePath, temporaryPdfPath); conversionError != nil {
			return nil, metrics, fmt.Errorf("failed to convert document to PDF: %w", conversionError)
		}
		pdfPath = temporaryPdfPath
		defer os.Remove(temporaryPdfPath)
	default:
		return nil, metrics, fmt.Errorf("unsupported document type: %s", extension)
	}

	return processor.processPDF(jobContext, pdfPath, document.ID, outputDirectory, languageCode, updateProgress)
}

func (processor *Processor) processPDF(jobContext context.Context, pdfPath string, documentID string, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, models.JobMetrics, error) {
	var metrics models.JobMetrics
	updateProgress(10, "Extracting pages as images...")
	imageFiles, extractionError := processor.converter.ExtractPagesAsImages(pdfPath, outputDirectory, processor.dpi)
	if extractionError != nil {
		return nil, metrics, extractionError
	}

	var extractedPages []models.ReferencePage
	totalImages := len(imageFiles)

	for imageIndex, imagePath := range imageFiles {
		pageNumber := imageIndex + 1
		progress := 10 + int(float64(imageIndex)/float64(totalImages)*90.0)
		updateProgress(progress, fmt.Sprintf("Interpreting page content %d/%d...", pageNumber, totalImages))

		extractedText, pageMetrics, interpretationError := processor.interpretPageContent(jobContext, imagePath, languageCode)
		if interpretationError != nil {
			return nil, metrics, fmt.Errorf("failed to interpret page %d: %w", pageNumber, interpretationError)
		}

		metrics.InputTokens += pageMetrics.InputTokens
		metrics.OutputTokens += pageMetrics.OutputTokens
		metrics.EstimatedCost += pageMetrics.EstimatedCost

		extractedPages = append(extractedPages, models.ReferencePage{
			DocumentID:    documentID,
			PageNumber:    pageNumber,
			ImagePath:     imagePath,
			ExtractedText: extractedText,
		})
	}

	return extractedPages, metrics, nil
}

func (processor *Processor) interpretPageContent(jobContext context.Context, imagePath string, languageCode string) (string, models.JobMetrics, error) {
	var metrics models.JobMetrics
	imageData, readError := os.ReadFile(imagePath)
	if readError != nil {
		return "", metrics, readError
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	var ingestPrompt string
	if processor.promptManager != nil {
		latexInstructions, _ := processor.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		languageRequirement, _ := processor.promptManager.GetPrompt(prompts.PromptLanguageRequirement, map[string]string{
			"language":      languageCode,
			"language_code": languageCode,
		})

		var promptError error
		ingestPrompt, promptError = processor.promptManager.GetPrompt(prompts.PromptIngestDocumentPage, map[string]string{
			"language_requirement": languageRequirement,
			"latex_instructions":   latexInstructions,
		})
		if promptError != nil {
			return "", metrics, promptError
		}
	} else {
		// Fallback prompt when promptManager is nil (e.g., in tests)
		ingestPrompt = fmt.Sprintf("Extract and transcribe all text content from this document page. The response must be written in %s.", languageCode)
	}

	request := llm.ChatRequest{
		Model: processor.llmModel,
		Messages: []llm.Message{
			{
				Role: "user",
				Content: []llm.ContentPart{
					{Type: "text", Text: ingestPrompt},
					{Type: "image", ImageURL: dataURL},
				},
			},
		},
	}

	responseChannel, chatError := processor.llmProvider.Chat(jobContext, &request)
	if chatError != nil {
		return "", metrics, chatError
	}

	var extractedTextBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return "", metrics, chunk.Error
		}
		extractedTextBuilder.WriteString(chunk.Text)
		metrics.InputTokens += chunk.InputTokens
		metrics.OutputTokens += chunk.OutputTokens
		metrics.EstimatedCost += chunk.Cost
	}

	return extractedTextBuilder.String(), metrics, nil
}
