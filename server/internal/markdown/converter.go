package markdown

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lectures/internal/media"
)

// MarkdownConverter defines the interface for document format conversions
type MarkdownConverter interface {
	CheckDependencies() error
	MarkdownToHTML(markdownText string) (string, error)
	NormalizeMath(markdownText string) string
	HTMLToPDF(htmlContent string, outputPath string, options ConversionOptions) error
	HTMLToDocx(htmlContent string, outputPath string, options ConversionOptions) error
	HTMLToAnki(toolType string, toolContent string, outputPath string) error
	HTMLToCSV(toolType string, toolContent string, outputPath string) error
	SaveMarkdown(markdownText string, outputPath string) error
	GenerateMetadataHeader(options ConversionOptions) string
}

// ExternalConverter handles document format conversions using Pandoc
type ExternalConverter struct {
	dataDirectory string
	binDir        string
}

// NewConverter creates a new document converter
func NewConverter(dataDirectory string, binDir string) MarkdownConverter {
	return &ExternalConverter{
		dataDirectory: dataDirectory,
		binDir:        binDir,
	}
}

// CheckDependencies verifies that pandoc and tectonic are installed
func (converter *ExternalConverter) CheckDependencies() error {
	p := media.ResolveBinaryPath("pandoc", converter.binDir)
	if _, err := exec.LookPath(p); err != nil {
		return fmt.Errorf("pandoc not found")
	}
	t := media.ResolveBinaryPath("tectonic", converter.binDir)
	if _, err := exec.LookPath(t); err != nil {
		return fmt.Errorf("tectonic not found")
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
	CourseTitle    string
	CreationDate   time.Time
	ReferenceFiles []ReferenceFileMetadata
	AudioFiles     []AudioFileMetadata
	QRCodePath     string
}

// MarkdownToHTML converts markdown text to HTML string
func (converter *ExternalConverter) MarkdownToHTML(markdownText string) (string, error) {
	// Normalize LaTeX delimiters before passing to pandoc
	markdownText = converter.normalizeMathDelimiters(markdownText)

	bin := media.ResolveBinaryPath("pandoc", converter.binDir)
	command := exec.Command(bin,
		"-f", "gfm+smart+tex_math_dollars",
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

// NormalizeMath exposes the normalization logic for other formats
func (converter *ExternalConverter) NormalizeMath(markdownText string) string {
	return converter.normalizeMathDelimiters(markdownText)
}

func (converter *ExternalConverter) normalizeMathDelimiters(markdown string) string {
	// 1. Escape pre-existing dollar signs (e.g., currency) BEFORE converting LaTeX delimiters
	// This ensures that actual math content wrapped in LaTeX delimiters isn't escaped,
	// but standalone dollars are.
	markdown = converter.escapeUnescapedDollars(markdown)

	// 2. \(...\) -> $...$
	inlineRegex := regexp.MustCompile(`(?s)\\\((.*?)\\\)`)
	markdown = inlineRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		// Extract content between \( and \)
		content := match[2 : len(match)-2]
		return "$" + strings.TrimSpace(content) + "$"
	})

	// 3. \[...\] -> $$...$$
	displayRegex := regexp.MustCompile(`(?s)\\\[(.*?)\\\]`)
	markdown = displayRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		// Extract content between \[ and \]
		content := match[2 : len(match)-2]
		return "$$" + content + "$$"
	})

	// 4. Escape literal asterisks used in parentheses like (*) to prevent <em> tags
	// Biology often uses (*) for stop codons
	markdown = strings.ReplaceAll(markdown, "(*)", "(\\*)")

	return markdown
}

func (converter *ExternalConverter) escapeUnescapedDollars(text string) string {
	// Match both double and single dollar math environments to skip them
	// Double dollars: $$ ... $$
	// Single dollars: $ ... $ (must not be empty and must not start/end with space)
	// We use a more robust regex that handles multi-line for double dollars
	combinedRegex := regexp.MustCompile(`(\$\$[\s\S]*?\$\$)|(\$[^\$\s][^\$]*?[^\$\s]\$)|(\$[^\$\s]\$)|(\$)`)

	return combinedRegex.ReplaceAllStringFunc(text, func(match string) string {
		// If it's a math block, return as is
		if len(match) > 1 && strings.HasPrefix(match, "$") {
			return match
		}
		// It's a single raw dollar sign
		return "\\\\$"
	})
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

	pandoc := media.ResolveBinaryPath("pandoc", converter.binDir)
	tectonic := media.ResolveBinaryPath("tectonic", converter.binDir)

	arguments := []string{
		"-f", "html",
		"-t", "pdf",
		"--resource-path", converter.dataDirectory,
		"--pdf-engine-opt=-Zcontinue-on-errors",
		"--pdf-engine=" + tectonic,
		"--template", templatePath,
		"--toc",
		"--shift-heading-level-by=-1",
		"--metadata-file", metadataPath,
		"-o", outputPath,
	}

	command := exec.Command(pandoc, arguments...)
	command.Stdin = strings.NewReader(htmlContent)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	// Handle Tectonic cache
	if os.Getenv("IN_DOCKER_ENV") == "true" {
		// In Docker, use a persistent cache directory within the data volume
		cacheDir := filepath.Join(converter.dataDirectory, "tectonic_cache")
		os.MkdirAll(cacheDir, 0755)
		command.Env = append(os.Environ(), "TECTONIC_CACHE="+cacheDir)
	} else {
		// Locally, create a temporary, unique cache directory for this run
		tempCacheDir, err := os.MkdirTemp("", "tectonic-cache-*")
		if err == nil {
			defer os.RemoveAll(tempCacheDir)
			command.Env = append(os.Environ(), "TECTONIC_CACHE="+tempCacheDir)
		}
	}

	if executionError := command.Run(); executionError != nil {
		return fmt.Errorf("pandoc pdf conversion failed: %v, stderr: %s", executionError, stderr.String())
	}

	return nil
}

// HTMLToDocx converts HTML content to a Docx file
func (converter *ExternalConverter) HTMLToDocx(htmlContent string, outputPath string, options ConversionOptions) error {
	bin := media.ResolveBinaryPath("pandoc", converter.binDir)
	arguments := []string{
		"-f", "html",
		"-t", "docx",
		"--resource-path", converter.dataDirectory,
		"--toc",
		"-o", outputPath,
	}

	if options.QRCodePath != "" {
		arguments = append(arguments, "--metadata", "qrcode-path="+strings.ReplaceAll(options.QRCodePath, "\\", "/"))
	}
	if options.CourseTitle != "" {
		arguments = append(arguments, "--metadata", "course-title="+options.CourseTitle)
		arguments = append(arguments, "--metadata", "course-title-label="+getI18nLabel(options.Language, "course_label"))
	}

	command := exec.Command(bin, arguments...)
	command.Stdin = strings.NewReader(htmlContent)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	if executionError := command.Run(); executionError != nil {
		return fmt.Errorf("pandoc docx conversion failed: %v, stderr: %s", executionError, stderr.String())
	}

	return nil
}

// HTMLToAnki converts tool content to an Anki-compatible tab-separated file
func (converter *ExternalConverter) HTMLToAnki(toolType string, toolContent string, outputPath string) error {
	var builder strings.Builder

	if toolType == "flashcard" {
		var flashcards []map[string]string
		if err := json.Unmarshal([]byte(toolContent), &flashcards); err != nil {
			return err
		}
		for _, fc := range flashcards {
			// Anki format: Front \t Back
			front := strings.ReplaceAll(fc["front"], "\n", "<br>")
			back := strings.ReplaceAll(fc["back"], "\n", "<br>")
			fmt.Fprintf(&builder, "%s\t%s\n", front, back)
		}
	} else if toolType == "quiz" {
		var quiz []map[string]any
		if err := json.Unmarshal([]byte(toolContent), &quiz); err != nil {
			return err
		}
		for _, item := range quiz {
			question := fmt.Sprintf("%v", item["question"])
			options, _ := json.Marshal(item["options"])
			correct := fmt.Sprintf("%v", item["correct_answer"])
			explanation := fmt.Sprintf("%v", item["explanation"])

			fmt.Fprintf(&builder, "%s\t%s\t%s\t%s\n", question, options, correct, explanation)
		}
	}

	return os.WriteFile(outputPath, []byte(builder.String()), 0644)
}

// HTMLToCSV converts tool content to a standard CSV file
func (converter *ExternalConverter) HTMLToCSV(toolType string, toolContent string, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	switch toolType {
	case "flashcard":
		var flashcards []map[string]string
		if err := json.Unmarshal([]byte(toolContent), &flashcards); err != nil {
			return err
		}
		writer.Write([]string{"Front", "Back"})
		for _, fc := range flashcards {
			writer.Write([]string{fc["front"], fc["back"]})
		}
	case "quiz":
		var quiz []map[string]any
		if err := json.Unmarshal([]byte(toolContent), &quiz); err != nil {
			return err
		}
		writer.Write([]string{"Question", "Options", "Correct Answer", "Explanation"})
		for _, item := range quiz {
			optionsBytes, _ := json.Marshal(item["options"])
			writer.Write([]string{
				fmt.Sprintf("%v", item["question"]),
				string(optionsBytes),
				fmt.Sprintf("%v", item["correct_answer"]),
				fmt.Sprintf("%v", item["explanation"]),
			})
		}
	}

	return nil
}

// SaveMarkdown saves the markdown text to a file (GFM format)
func (converter *ExternalConverter) SaveMarkdown(markdownText string, outputPath string) error {
	return os.WriteFile(outputPath, []byte(markdownText), 0644)
}

// FormatDuration formats a duration in seconds into a localized string
func (converter *ExternalConverter) FormatDuration(durationSeconds int64, language string) string {
	if durationSeconds <= 0 {
		return ""
	}

	hourLabel := getI18nLabel(language, "hour_label")
	minuteLabel := getI18nLabel(language, "minute_label")
	secondLabel := getI18nLabel(language, "second_label")

	hours := durationSeconds / 3600
	minutes := (durationSeconds % 3600) / 60
	seconds := durationSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d%s %d%s", hours, hourLabel, minutes, minuteLabel)
	} else if minutes > 0 {
		return fmt.Sprintf("%d%s %d%s", minutes, minuteLabel, seconds, secondLabel)
	} else {
		return fmt.Sprintf("%d%s", seconds, secondLabel)
	}
}

// imageToBase64 reads an image file and returns a data URI
func imageToBase64(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	base64Data := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:image/png;base64,%s", base64Data)
}

// GenerateMetadataHeader generates a localized Markdown header containing the document metadata
func (converter *ExternalConverter) GenerateMetadataHeader(options ConversionOptions) string {
	var builder strings.Builder

	// QR Code (Top right-ish if possible, but in MD it's just top)
	if options.QRCodePath != "" {
		dataURI := imageToBase64(options.QRCodePath)
		if dataURI != "" {
			// We use a small image for the QR code
			fmt.Fprintf(&builder, "![](%s){ width=80px }\n\n", dataURI)
		}
	}

	// 0. Course
	if options.CourseTitle != "" {
		courseLabel := getI18nLabel(options.Language, "course_label")
		fmt.Fprintf(&builder, "**%s**: %s\n\n", courseLabel, options.CourseTitle)
	}

	// 1. Date
	if !options.CreationDate.IsZero() {
		dateLabel := getI18nLabel(options.Language, "date_label")
		dateString := formatLocalizedDate(options.CreationDate, options.Language)
		fmt.Fprintf(&builder, "**%s**: %s\n\n", dateLabel, dateString)
	}

	// 2. Abstract
	if options.Description != "" {
		abstractLabel := getI18nLabel(options.Language, "abstract")
		// Capitalize first letter of label
		capitalizedLabel := strings.ToUpper(abstractLabel[:1]) + abstractLabel[1:]
		fmt.Fprintf(&builder, "### %s\n\n%s\n\n", capitalizedLabel, options.Description)
	}

	// 3. Audio Files
	if len(options.AudioFiles) > 0 {
		audioLabel := getI18nLabel(options.Language, "audio_files")
		fmt.Fprintf(&builder, "### %s\n\n", audioLabel)
		for _, audio := range options.AudioFiles {
			duration := converter.FormatDuration(audio.Duration, options.Language)
			if duration != "" {
				fmt.Fprintf(&builder, "- `%s` (%s)\n", audio.Filename, duration)
			} else {
				fmt.Fprintf(&builder, "- `%s`\n", audio.Filename)
			}
		}
		builder.WriteString("\n")
	}

	// 4. Reference Files
	if len(options.ReferenceFiles) > 0 {
		referenceLabel := getI18nLabel(options.Language, "reference_files")
		pageLabel := getI18nLabel(options.Language, "page_label")
		pagesLabel := getI18nLabel(options.Language, "pages_label")

		fmt.Fprintf(&builder, "### %s\n\n", referenceLabel)
		for _, file := range options.ReferenceFiles {
			metadataStr := ""
			if file.PageRange != "" {
				metadataStr = pagesLabel + " " + file.PageRange
			} else if file.PageCount > 0 {
				label := pagesLabel
				if file.PageCount == 1 {
					label = pageLabel
				}
				metadataStr = fmt.Sprintf("%s 1-%d", label, file.PageCount)
			}

			if metadataStr != "" {
				fmt.Fprintf(&builder, "- `%s` (%s)\n", file.Filename, metadataStr)
			} else {
				fmt.Fprintf(&builder, "- `%s`\n", file.Filename)
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// findAvailableFont checks which fonts are installed on the system using fc-list
// and returns the first matching candidate, or empty string if none found.
func findAvailableFont(candidates []string) string {
	for _, font := range candidates {
		cmd := exec.Command("fc-list", font)
		output, err := cmd.Output()
		if err == nil && len(bytes.TrimSpace(output)) > 0 {
			slog.Debug("Found CJK font", "font", font)
			return font
		}
	}
	slog.Warn("No CJK font found from candidates", "candidates", candidates)
	return ""
}

func (converter *ExternalConverter) writeMetadataFile(path string, options ConversionOptions) error {
	slog.Info("Preparing PDF metadata",
		"language", options.Language,
		"description_length", len(options.Description),
		"creation_date", options.CreationDate,
		"reference_files", options.ReferenceFiles,
		"audio_files", options.AudioFiles)

	var builder strings.Builder

	// Normalize language code (e.g., "ja-JP" -> "ja")
	normalizedLanguage := strings.ToLower(options.Language)
	if idx := strings.Index(normalizedLanguage, "-"); idx != -1 {
		normalizedLanguage = normalizedLanguage[:idx]
	}

	// CJK languages need special handling for XeLaTeX
	cjkLanguages := map[string]bool{"ja": true, "ko": true, "zh": true}
	isCJK := cjkLanguages[normalizedLanguage]

	// Add the language for pandoc
	// For CJK languages, use the full language tag for proper rendering
	if isCJK {
		// Use BCP 47 tag for CJK (e.g., ja-JP, ko-KR, zh-CN)
		fmt.Fprintf(&builder, "lang: \"%s\"\n", options.Language)
	} else {
		fmt.Fprintf(&builder, "lang: \"%s\"\n", options.Language)
	}

	// Add translated labels for the template
	fmt.Fprintf(&builder, "course-title-label: \"%s\"\n", getI18nLabel(options.Language, "course_label"))
	fmt.Fprintf(&builder, "abstract-title: \"%s\"\n", getI18nLabel(options.Language, "abstract"))
	fmt.Fprintf(&builder, "audio-files-title: \"%s\"\n", getI18nLabel(options.Language, "audio_files"))
	fmt.Fprintf(&builder, "reference-files-title: \"%s\"\n", getI18nLabel(options.Language, "reference_files"))

	// Add font settings for CJK languages
	if isCJK {
		// Ordered list of font candidates per language (first available wins)
		languageFontCandidates := map[string][]string{
			"ja": {"Noto Serif JP", "Noto Serif CJK JP", "Hiragino Mincho ProN", "Noto Sans CJK JP"},
			"ko": {"Noto Serif KR", "Noto Serif CJK KR", "Apple SD Gothic Neo", "Noto Sans CJK KR"},
			"zh": {"Noto Serif SC", "Noto Serif CJK SC", "STSong", "Noto Sans CJK SC"},
		}
		if candidates, ok := languageFontCandidates[normalizedLanguage]; ok {
			if font := findAvailableFont(candidates); font != "" {
				fmt.Fprintf(&builder, "header-includes:\n")
				fmt.Fprintf(&builder, "  - |\n")
				fmt.Fprintf(&builder, "    ```{=latex}\n")
				fmt.Fprintf(&builder, "    \\newfontfamily\\cjkfont{%s}\n", font)
				fmt.Fprintf(&builder, "    \\XeTeXlinebreaklocale \"%s\"\n", normalizedLanguage)
				fmt.Fprintf(&builder, "    \\XeTeXlinebreakskip = 0pt plus 1pt\n")
				fmt.Fprintf(&builder, "    \\usepackage{ucharclasses}\n")
				fmt.Fprintf(&builder, "    \\setTransitionsForCJK{\\cjkfont}{\\rmfamily}\n")
				fmt.Fprintf(&builder, "    ```\n")
			}
		}
	}

	if options.CourseTitle != "" {
		fmt.Fprintf(&builder, "course-title: \"%s\"\n", strings.ReplaceAll(options.CourseTitle, "\"", "\\\""))
	}

	if options.QRCodePath != "" {
		fmt.Fprintf(&builder, "qrcode-path: \"%s\"\n", strings.ReplaceAll(options.QRCodePath, "\\", "/"))
	}

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
		for _, file := range options.AudioFiles {
			durationStr := converter.FormatDuration(file.Duration, options.Language)
			slog.Debug("Adding audio file to PDF metadata", "filename", file.Filename, "duration_seconds", file.Duration, "duration_formatted", durationStr)
			fmt.Fprintf(&builder, "  - filename: \"%s\"\n    metadata: \"%s\"\n", file.Filename, durationStr)
		}
	}

	yamlContent := builder.String()
	slog.Info("Writing PDF metadata YAML", "path", path, "yaml_length", len(yamlContent))

	return os.WriteFile(path, []byte(yamlContent), 0644)
}
