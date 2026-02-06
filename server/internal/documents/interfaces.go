package documents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DocumentConverter defines the interface for document processing operations
type DocumentConverter interface {
	CheckDependencies() error
	ConvertToPDF(inputPath string, outputPath string) error
	ExtractPagesAsImages(pdfPath string, outputDirectory string) ([]string, error)
}

// ExternalDocumentConverter implementation that uses Ghostscript and LibreOffice
type ExternalDocumentConverter struct{}

func (c *ExternalDocumentConverter) CheckDependencies() error {
	if _, lookError := exec.LookPath("gs"); lookError != nil {
		return fmt.Errorf("ghostscript (gs) not found in PATH")
	}
	if _, lookError := exec.LookPath("soffice"); lookError != nil {
		return fmt.Errorf("libreoffice (soffice) not found in PATH")
	}
	return nil
}

func (c *ExternalDocumentConverter) ConvertToPDF(inputPath string, outputPath string) error {
	outputDirectory := filepath.Dir(outputPath)
	command := exec.Command("soffice", "--headless", "--convert-to", "pdf", "--outdir", outputDirectory, inputPath)

	var stderr strings.Builder
	command.Stderr = &stderr
	if executionError := command.Run(); executionError != nil {
		return fmt.Errorf("libreoffice conversion failed: %v, stderr: %s", executionError, stderr.String())
	}

	generatedFilename := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) + ".pdf"
	generatedPath := filepath.Join(outputDirectory, generatedFilename)

	if _, statError := os.Stat(generatedPath); os.IsNotExist(statError) {
		return fmt.Errorf("converted PDF file not found at %s", generatedPath)
	}

	if generatedPath != outputPath {
		if renameError := os.Rename(generatedPath, outputPath); renameError != nil {
			return fmt.Errorf("failed to move converted PDF: %w", renameError)
		}
	}

	return nil
}

func (c *ExternalDocumentConverter) ExtractPagesAsImages(pdfPath string, outputDirectory string) ([]string, error) {
	outputPattern := filepath.Join(outputDirectory, "page_%03d.png")
	command := exec.Command("gs", "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=png16m", "-r150", fmt.Sprintf("-sOutputFile=%s", outputPattern), pdfPath)

	var stderr strings.Builder
	command.Stderr = &stderr
	if executionError := command.Run(); executionError != nil {
		return nil, fmt.Errorf("ghostscript page extraction failed: %v, stderr: %s", executionError, stderr.String())
	}

	imageFiles, globError := filepath.Glob(filepath.Join(outputDirectory, "page_*.png"))
	if globError != nil {
		return nil, globError
	}

	return imageFiles, nil
}
