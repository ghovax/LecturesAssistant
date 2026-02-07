package markdown

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownConverter defines the interface for document format conversions
type MarkdownConverter interface {
	CheckDependencies() error
	MarkdownToHTML(markdownText string) (string, error)
	HTMLToPDF(htmlContent string, outputPath string, options ConversionOptions) error
}

// ExternalConverter handles document format conversions using Pandoc
type ExternalConverter struct {
	dataDirectory string
}

// NewConverter creates a new document converter
func NewConverter(dataDirectory string) MarkdownConverter {
	return &ExternalConverter{
		dataDirectory: dataDirectory,
	}
}

// CheckDependencies verifies that pandoc and tectonic are installed
func (converter *ExternalConverter) CheckDependencies() error {
	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf("pandoc not found in PATH")
	}
	if _, err := exec.LookPath("tectonic"); err != nil {
		return fmt.Errorf("tectonic not found in PATH")
	}
	return nil
}

// ReferenceFileMetadata represents a reference file for PDF metadata
type ReferenceFileMetadata struct {
	Filename  string
	PageRange string
	PageCount int
}

// AudioFileMetadata represents an audio file for PDF metadata
type AudioFileMetadata struct {
	Filename string
	Duration int64 // seconds
}

// ConversionOptions contains settings for PDF generation
type ConversionOptions struct {
	Language       string
	Description    string
	CreationDate   time.Time
	ReferenceFiles []ReferenceFileMetadata
	AudioFiles     []AudioFileMetadata
}

// MarkdownToHTML converts markdown text to HTML string
func (converter *ExternalConverter) MarkdownToHTML(markdownText string) (string, error) {
	command := exec.Command("pandoc",
		"-f", "gfm+smart",
		"-t", "html",
		"--standalone=false",
		"--mathml",
		"--wrap=none",
		"--toc",
		"--section-divs=false",
	)

	command.Stdin = strings.NewReader(markdownText)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return "", fmt.Errorf("pandoc html conversion failed: %v, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// HTMLToPDF converts HTML content to a PDF file
func (converter *ExternalConverter) HTMLToPDF(htmlContent string, outputPath string, options ConversionOptions) error {
	metadataPath := filepath.Join(os.TempDir(), fmt.Sprintf("metadata-%d.yaml", time.Now().UnixNano()))
	if err := converter.writeMetadataFile(metadataPath, options); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	defer os.Remove(metadataPath)

	arguments := []string{
		"-f", "html",
		"-t", "pdf",
		"--pdf-engine-opt=-Zcontinue-on-errors",
		"--pdf-engine=tectonic",
		"--toc",
		"--shift-heading-level-by=-1",
		"--metadata-file", metadataPath,
		"-o", outputPath,
	}

	// Add font settings based on language
	languageFontMap := map[string]struct {
		mainfont   string
		cjkOptions string
	}{
		"ja": {mainfont: "Noto Serif JP", cjkOptions: "AutoFakeBold"},
		"ko": {mainfont: "Noto Serif KR", cjkOptions: "AutoFakeBold"},
		"zh": {mainfont: "Noto Serif SC", cjkOptions: "AutoFakeBold"},
		"ar": {mainfont: "Noto Serif Arabic", cjkOptions: ""},
		"he": {mainfont: "Noto Serif Hebrew", cjkOptions: ""},
		"th": {mainfont: "Noto Serif Thai", cjkOptions: ""},
		"hi": {mainfont: "Noto Serif Devanagari", cjkOptions: ""},
		"bn": {mainfont: "Noto Serif Bengali", cjkOptions: ""},
		"ta": {mainfont: "Noto Serif Tamil", cjkOptions: ""},
		"hy": {mainfont: "Noto Serif Armenian", cjkOptions: ""},
		"ka": {mainfont: "Noto Serif Georgian", cjkOptions: ""},
		"ru": {mainfont: "Noto Serif", cjkOptions: ""},
	}

	if fontInfo, ok := languageFontMap[options.Language]; ok {
		if fontInfo.mainfont != "" {
			arguments = append(arguments, "-V", "mainfont="+fontInfo.mainfont)
		}
		if fontInfo.cjkOptions != "" {
			arguments = append(arguments, "-V", "CJKoptions="+fontInfo.cjkOptions)
		}
	}

	command := exec.Command("pandoc", arguments...)
	command.Stdin = strings.NewReader(htmlContent)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("pandoc pdf conversion failed: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

func (converter *ExternalConverter) writeMetadataFile(path string, options ConversionOptions) error {
	var builder strings.Builder

	if options.Description != "" {
		fmt.Fprintf(&builder, "abstract: \"%s\"\n", strings.ReplaceAll(options.Description, "\"", "\\\""))
	}

	if !options.CreationDate.IsZero() {
		dateString := options.CreationDate.Format("January 2, 2006")
		fmt.Fprintf(&builder, "date: \"%s\"\n", dateString)
	}

	if len(options.ReferenceFiles) > 0 {
		builder.WriteString("referencefile:\n")
		for _, file := range options.ReferenceFiles {
			metadataStr := ""
			if file.PageRange != "" {
				metadataStr = "pp. " + file.PageRange
			} else if file.PageCount > 0 {
				label := "pp."
				if file.PageCount == 1 {
					label = "p."
				}
				metadataStr = fmt.Sprintf("%s 1--%d", label, file.PageCount)
			}
			fmt.Fprintf(&builder, "  - filename: \"%s\"\n    metadata: \"%s\"\n", file.Filename, metadataStr)
		}
	}

	if len(options.AudioFiles) > 0 {
		builder.WriteString("audiofile:\n")
		for _, file := range options.AudioFiles {
			durationStr := ""
			if file.Duration > 0 {
				hours := file.Duration / 3600
				minutes := (file.Duration % 3600) / 60
				if hours > 0 {
					durationStr = fmt.Sprintf("%dh %dm", hours, minutes)
				} else {
					durationStr = fmt.Sprintf("%dm", minutes)
				}
			}
			fmt.Fprintf(&builder, "  - filename: \"%s\"\n    metadata: \"%s\"\n", file.Filename, durationStr)
		}
	}

	yamlContent := builder.String()
	slog.Debug("Writing PDF metadata YAML", "path", path, "content", yamlContent)

	return os.WriteFile(path, []byte(yamlContent), 0644)
}
