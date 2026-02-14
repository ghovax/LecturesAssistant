package documents

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"lectures/internal/media"
)

// DocumentConverter defines the interface for document processing operations
type DocumentConverter interface {
	CheckDependencies() error
	ConvertToPDF(inputPath string, outputPath string) error
	ExtractPagesAsImages(pdfPath string, outputDirectory string, dpi int) ([]string, error)
}

// ExternalDocumentConverter implementation that uses Ghostscript and LibreOffice
type ExternalDocumentConverter struct {
	binDir string
}

func (c *ExternalDocumentConverter) resolveSofficePath() (string, error) {
	// 1. Check local bin
	local := media.ResolveBinaryPath("soffice", c.binDir)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}

	// 2. Check PATH
	if path, err := exec.LookPath("soffice"); err == nil {
		return path, nil
	}

	// 3. Check common macOS location
	macOSPath := "/Applications/LibreOffice.app/Contents/MacOS/soffice"
	if _, err := os.Stat(macOSPath); err == nil {
		return macOSPath, nil
	}

	return "", fmt.Errorf("libreoffice (soffice) not found in PATH or /Applications")
}

func (c *ExternalDocumentConverter) CheckDependencies() error {
	gs := media.ResolveBinaryPath("gs", c.binDir)
	if _, lookError := exec.LookPath(gs); lookError != nil {
		return fmt.Errorf("ghostscript (gs) not found")
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
	outputPattern := filepath.Join(outputDirectory, "%03d.png")
	gs := media.ResolveBinaryPath("gs", c.binDir)
	command := exec.Command(gs, "-dSAFER", "-dBATCH", "-dNOPAUSE", "-sDEVICE=png16m", fmt.Sprintf("-r%d", dpi), fmt.Sprintf("-sOutputFile=%s", outputPattern), pdfPath)

	var stderr strings.Builder
	command.Stderr = &stderr
	if executionError := command.Run(); executionError != nil {
		return nil, fmt.Errorf("ghostscript page extraction failed: %v, stderr: %s", executionError, stderr.String())
	}

	imageFiles, globError := filepath.Glob(filepath.Join(outputDirectory, "[0-9][0-9][0-9].png"))
	if globError != nil {
		return nil, globError
	}

	return imageFiles, nil
}
