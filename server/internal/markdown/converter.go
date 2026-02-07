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

	// Locate the custom XeLaTeX template in server root directory
	// Server must be run from the server root directory
	templatePath := "xelatex-template.tex"

	slog.Debug("Using XeLaTeX template", "path", templatePath, "exists", fileExists(templatePath))

	arguments := []string{
		"-f", "html",
		"-t", "pdf",
		"--pdf-engine-opt=-Zcontinue-on-errors",
		"--pdf-engine=tectonic",
		"--template", templatePath,
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (converter *ExternalConverter) writeMetadataFile(path string, options ConversionOptions) error {
	slog.Info("Preparing PDF metadata",
		"language", options.Language,
		"description", options.Description,
		"creation_date", options.CreationDate,
		"reference_files", options.ReferenceFiles,
		"audio_files", options.AudioFiles)

	var builder strings.Builder

	// Add the language for pandoc
	fmt.Fprintf(&builder, "lang: \"%s\"\n", options.Language)

	// Add translated labels for the template
	fmt.Fprintf(&builder, "abstract-title: \"%s\"\n", getI18nLabel(options.Language, "abstract"))
	fmt.Fprintf(&builder, "audio-files-title: \"%s\"\n", getI18nLabel(options.Language, "audio_files"))
	fmt.Fprintf(&builder, "reference-files-title: \"%s\"\n", getI18nLabel(options.Language, "reference_files"))

	if options.Description != "" {
		fmt.Fprintf(&builder, "abstract: \"%s\"\n", strings.ReplaceAll(options.Description, "\"", "\\\""))
	}

	if !options.CreationDate.IsZero() {
		dateString := formatLocalizedDate(options.CreationDate, options.Language)
		fmt.Fprintf(&builder, "date: \"%s\"\n", dateString)
	}

	if len(options.ReferenceFiles) > 0 {
		builder.WriteString("referencefile:\n")
		pageLabel := getI18nLabel(options.Language, "page_label")
		pagesLabel := getI18nLabel(options.Language, "pages_label")

		for _, file := range options.ReferenceFiles {
			metadataStr := ""
			if file.PageRange != "" {
				metadataStr = pagesLabel + " " + file.PageRange
			} else if file.PageCount > 0 {
				label := pagesLabel
				if file.PageCount == 1 {
					label = pageLabel
				}
				metadataStr = fmt.Sprintf("%s 1--%d", label, file.PageCount)
			}
			fmt.Fprintf(&builder, "  - filename: \"%s\"\n    metadata: \"%s\"\n", file.Filename, metadataStr)
		}
	}

	if len(options.AudioFiles) > 0 {
		builder.WriteString("audiofile:\n")
		hourLabel := getI18nLabel(options.Language, "hour_label")
		minuteLabel := getI18nLabel(options.Language, "minute_label")
		secondLabel := getI18nLabel(options.Language, "second_label")

		for _, file := range options.AudioFiles {
			durationStr := ""
			if file.Duration > 0 {
				hours := file.Duration / 3600
				minutes := (file.Duration % 3600) / 60
				seconds := file.Duration % 60

				if hours > 0 {
					// Show hours and minutes only
					durationStr = fmt.Sprintf("%d%s %d%s", hours, hourLabel, minutes, minuteLabel)
				} else if minutes > 0 {
					// Show minutes and seconds only
					durationStr = fmt.Sprintf("%d%s %d%s", minutes, minuteLabel, seconds, secondLabel)
				} else {
					// Show seconds only
					durationStr = fmt.Sprintf("%d%s", seconds, secondLabel)
				}
			}
			slog.Debug("Adding audio file to PDF metadata", "filename", file.Filename, "duration_seconds", file.Duration, "duration_formatted", durationStr)
			fmt.Fprintf(&builder, "  - filename: \"%s\"\n    metadata: \"%s\"\n", file.Filename, durationStr)
		}
	}

	yamlContent := builder.String()
	slog.Info("Writing PDF metadata YAML", "path", path, "yaml_length", len(yamlContent), "yaml_content", yamlContent)

	return os.WriteFile(path, []byte(yamlContent), 0644)
}
