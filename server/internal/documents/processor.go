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
}

func NewProcessor(llmProvider llm.Provider, llmModel string, promptManager *prompts.Manager) *Processor {
	return &Processor{
		llmProvider:   llmProvider,
		llmModel:      llmModel,
		promptManager: promptManager,
		converter:     &ExternalDocumentConverter{},
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
func (processor *Processor) ProcessDocument(jobContext context.Context, document models.ReferenceDocument, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, error) {
	if directoryError := os.MkdirAll(outputDirectory, 0755); directoryError != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", directoryError)
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
			return nil, fmt.Errorf("failed to convert document to PDF: %w", conversionError)
		}
		pdfPath = temporaryPdfPath
		defer os.Remove(temporaryPdfPath)
	default:
		return nil, fmt.Errorf("unsupported document type: %s", extension)
	}

	return processor.processPDF(jobContext, pdfPath, document.ID, outputDirectory, languageCode, updateProgress)
}

func (processor *Processor) processPDF(jobContext context.Context, pdfPath string, documentID string, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, error) {
	updateProgress(10, "Extracting pages as images...")
	imageFiles, extractionError := processor.converter.ExtractPagesAsImages(pdfPath, outputDirectory)
	if extractionError != nil {
		return nil, extractionError
	}

	var extractedPages []models.ReferencePage
	totalImages := len(imageFiles)

	for index, imagePath := range imageFiles {
		pageNumber := index + 1
		progress := 10 + int(float64(index)/float64(totalImages)*90.0)
		updateProgress(progress, fmt.Sprintf("Interpreting page content %d/%d...", pageNumber, totalImages))

		extractedText, interpretationError := processor.interpretPageContent(jobContext, imagePath, languageCode)
		if interpretationError != nil {
			return nil, fmt.Errorf("failed to interpret page %d: %w", pageNumber, interpretationError)
		}

		extractedPages = append(extractedPages, models.ReferencePage{
			DocumentID:    documentID,
			PageNumber:    pageNumber,
			ImagePath:     imagePath,
			ExtractedText: extractedText,
		})
	}

	return extractedPages, nil
}

func (processor *Processor) interpretPageContent(jobContext context.Context, imagePath string, languageCode string) (string, error) {
	imageData, readError := os.ReadFile(imagePath)
	if readError != nil {
		return "", readError
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	latexInstructions, _ := processor.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	ingestPrompt, promptError := processor.promptManager.GetPrompt(prompts.PromptIngestDocumentPage, map[string]string{
		"language_requirement": fmt.Sprintf("The response must be written in %s.", languageCode),
		"latex_instructions":   latexInstructions,
	})
	if promptError != nil {
		return "", promptError
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

	responseChannel, chatError := processor.llmProvider.Chat(jobContext, request)
	if chatError != nil {
		return "", chatError
	}

	var extractedTextBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return "", chunk.Error
		}
		extractedTextBuilder.WriteString(chunk.Text)
	}

	return extractedTextBuilder.String(), nil
}
