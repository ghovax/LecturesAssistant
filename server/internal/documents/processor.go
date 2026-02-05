package documents

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
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
}

func NewProcessor(llmProvider llm.Provider, llmModel string, promptManager *prompts.Manager) *Processor {
	return &Processor{
		llmProvider:   llmProvider,
		llmModel:      llmModel,
		promptManager: promptManager,
	}
}

// CheckDependencies verifies that Ghostscript and LibreOffice are installed
func (processor *Processor) CheckDependencies() error {
	if _, err := exec.LookPath("gs"); err != nil {
		return fmt.Errorf("ghostscript (gs) not found in PATH")
	}
	if _, err := exec.LookPath("soffice"); err != nil {
		return fmt.Errorf("libreoffice (soffice) not found in PATH")
	}
	return nil
}

// ProcessDocument extracts pages as images and performs OCR using a Vision LLM
func (processor *Processor) ProcessDocument(context context.Context, document models.ReferenceDocument, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, error) {
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	extension := strings.ToLower(filepath.Ext(document.FilePath))
	var pdfPath string

	switch extension {
	case ".pdf":
		pdfPath = document.FilePath
	case ".pptx", ".docx":
		updateProgress(5, "Converting document to PDF...")
		temporaryPdfPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.pdf", document.ID))
		if err := processor.convertToPDF(document.FilePath, temporaryPdfPath); err != nil {
			return nil, fmt.Errorf("failed to convert document to PDF: %w", err)
		}
		pdfPath = temporaryPdfPath
		defer os.Remove(temporaryPdfPath)
	default:
		return nil, fmt.Errorf("unsupported document type: %s", extension)
	}

	return processor.processPDF(context, pdfPath, document.ID, outputDirectory, languageCode, updateProgress)
}

func (processor *Processor) convertToPDF(inputPath string, outputPath string) error {
	// soffice --headless --convert-to pdf --outdir /tmp input.docx
	outputDirectory := filepath.Dir(outputPath)
	command := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", outputDirectory, inputPath)

	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("libreoffice conversion failed: %v, stderr: %s", err, stderr.String())
	}

	// LibreOffice generates <filename_without_ext>.pdf in the outdir
	generatedFilename := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) + ".pdf"
	generatedPath := filepath.Join(outputDirectory, generatedFilename)

	if _, err := os.Stat(generatedPath); os.IsNotExist(err) {
		return fmt.Errorf("converted PDF file not found at %s", generatedPath)
	}

	// Move to the requested outputPath if different
	if generatedPath != outputPath {
		if err := os.Rename(generatedPath, outputPath); err != nil {
			return fmt.Errorf("failed to move converted PDF: %w", err)
		}
	}

	return nil
}

func (processor *Processor) processPDF(context context.Context, pdfPath string, documentID string, outputDirectory string, languageCode string, updateProgress func(int, string)) ([]models.ReferencePage, error) {
	// gs -dSAFER -dBATCH -dNOPAUSE -sDEVICE=png16m -r150 -sOutputFile=page_%03d.png input.pdf
	outputPattern := filepath.Join(outputDirectory, "page_%03d.png")
	command := exec.Command("gs", "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=png16m", "-r150", fmt.Sprintf("-sOutputFile=%s", outputPattern), pdfPath)

	updateProgress(10, "Extracting pages as images...")
	var stderr strings.Builder
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("ghostscript page extraction failed: %v, stderr: %s", err, stderr.String())
	}

	imageFiles, err := filepath.Glob(filepath.Join(outputDirectory, "page_*.png"))
	if err != nil {
		return nil, err
	}

	var extractedPages []models.ReferencePage
	totalImages := len(imageFiles)

	for index, imagePath := range imageFiles {
		pageNumber := index + 1
		progress := 10 + int(float64(index)/float64(totalImages)*90.0)
		updateProgress(progress, fmt.Sprintf("Performing OCR on page %d/%d...", pageNumber, totalImages))

		extractedText, err := processor.performOCR(context, imagePath, languageCode)
		if err != nil {
			extractedText = fmt.Sprintf("[OCR Failed: %v]", err)
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

func (processor *Processor) performOCR(context context.Context, imagePath string, languageCode string) (string, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	ingestPrompt, err := processor.promptManager.GetPrompt(prompts.PromptIngestDocumentPage, map[string]string{
		"language_requirement": fmt.Sprintf("The response must be written in %s.", languageCode),
		"latex_instructions":   "", // Optional
	})
	if err != nil {
		return "", err
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

	responseChannel, err := processor.llmProvider.Chat(context, request)
	if err != nil {
		return "", err
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
