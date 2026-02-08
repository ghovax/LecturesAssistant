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
	ExtractPagesAsImages(pdfPath string, outputDirectory string, dpi int) ([]string, error)
}

// ExternalDocumentConverter implementation that uses Ghostscript and LibreOffice
type ExternalDocumentConverter struct{}

func (c *ExternalDocumentConverter) resolveSofficePath() (string, error) {
	// 1. Check PATH
	if path, err := exec.LookPath("soffice"); err == nil {
		return path, nil
	}

	// 2. Check common macOS location
	macOSPath := "/Applications/LibreOffice.app/Contents/MacOS/soffice"
	if _, err := os.Stat(macOSPath); err == nil {
		return macOSPath, nil
	}

	return "", fmt.Errorf("libreoffice (soffice) not found in PATH or /Applications")
}

func (c *ExternalDocumentConverter) CheckDependencies() error {
	if _, lookError := exec.LookPath("gs"); lookError != nil {
		return fmt.Errorf("ghostscript (gs) not found in PATH")
	}
	if _, err := c.resolveSofficePath(); err != nil {
		return err
	}
	return nil
}

func (c *ExternalDocumentConverter) ConvertToPDF(inputPath string, outputPath string) error {
	sofficePath, err := c.resolveSofficePath()
	if err != nil {
		return err
	}

	outputDirectory := filepath.Dir(outputPath)
	command := exec.Command(sofficePath, "--headless", "--convert-to", "pdf", "--outdir", outputDirectory, inputPath)

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

func (c *ExternalDocumentConverter) ExtractPagesAsImages(pdfPath string, outputDirectory string, dpi int) ([]string, error) {
	if dpi <= 0 {
		dpi = 150 // Fallback
	}
	outputPattern := filepath.Join(outputDirectory, "page_%03d.png")
	command := exec.Command("gs", "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=png16m", fmt.Sprintf("-r%d", dpi), fmt.Sprintf("-sOutputFile=%s", outputPattern), pdfPath)

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
