package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// Reconstructor handles converting an AST back into markdown text
type Reconstructor struct {
	indentUnit    int
	Language      string
	IncludeImages bool
}

// NewReconstructor creates a new markdown reconstructor
func NewReconstructor() *Reconstructor {
	return &Reconstructor{
		indentUnit:    4,
		Language:      "en",
		IncludeImages: true,
	}
}

// Reconstruct converts an AST node (and its children) into markdown text
func (reconstructor *Reconstructor) Reconstruct(node *Node) string {
	var markdownLines []string
	reconstructor.reconstructNode(node, &markdownLines)

	result := strings.Join(markdownLines, "\n")
	result = reconstructor.applyCitationPostProcessing(result)

	return strings.TrimSpace(result) + "\n"
}

func (reconstructor *Reconstructor) applyCitationPostProcessing(text string) string {
	// Move periods/commas before footnote references to before and consolidate duplicates
	// 1. Move all surrounding punctuation to the left and strip whitespace
	result := regexp.MustCompile(`[ \t]*([.,]*)[ \t]*(\[\^\d+\])[ \t]*([.,]*)`).ReplaceAllString(text, "$1$3$2")

	// 2. Remove punctuation/whitespace between consecutive citations: "[^1]. [^2]" -> ".[^1][^2]"
	result = regexp.MustCompile(`(\[\^\d+\])[ \t.,]*(\[\^\d+\])`).ReplaceAllString(result, "$1$2")

	// 3. Consolidate multiple dots/commas into one (now at the left of the first reference)
	result = regexp.MustCompile(`([.,]{2,})(\[\^\d+\])`).ReplaceAllStringFunc(result, func(match string) string {
		sub := regexp.MustCompile(`([.,]{2,})(\[\^\d+\])`).FindStringSubmatch(match)
		return sub[1][:1] + sub[2]
	})

	// 4. Ensure space after citation if followed by text (e.g., "[^1]Next" -> "[^1] Next")
	// Use a positive lookahead for any non-whitespace, non-punctuation character
	result = regexp.MustCompile(`(\[\^\d+\])([^\s.,:;!?)\]\[])`).ReplaceAllString(result, "$1 $2")

	// 5. Ensure space after a dot if followed by an uppercase letter: ".Cio" -> ". Cio"
	// This also handles ellipsis followed by a capital letter.
	result = regexp.MustCompile(`(\.+)([A-Z])`).ReplaceAllString(result, "$1 $2")

	// 6. Ensure space after a colon if followed by a character: "Word:Next" -> "Word: Next"
	// We avoid common URL patterns (http:, https:) and don't add space if it's a protocol (followed by //)
	// We also handle cases where bold/italic markers follow the colon: "**Word:**Burun" -> "**Word:** Burun"
	result = regexp.MustCompile(`(\w+:)([*\s_]*)([^\s/])`).ReplaceAllStringFunc(result, func(match string) string {
		if strings.HasPrefix(strings.ToLower(match), "http:") || strings.HasPrefix(strings.ToLower(match), "https:") {
			return match
		}

		// If there's already a space after the colon (even with markers), return as is
		if strings.Contains(match, ": ") {
			return match
		}

		// Find the colon and insert space after all markers
		parts := strings.SplitN(match, ":", 2)
		// parts[1] contains markers and the next character
		// We want to find where the "content" starts (non-marker, non-space)
		// But wait, if it's ":**Burun", parts[1] is "**Burun".
		// We should put space after "**".

		markersRegex := regexp.MustCompile(`^([*_]+)`)
		markersMatch := markersRegex.FindString(parts[1])

		if markersMatch != "" {
			return parts[0] + ":" + markersMatch + " " + parts[1][len(markersMatch):]
		}

		return parts[0] + ": " + parts[1]
	})

	return result
}

// AppendCitations appends footnote definitions to the end of the markdown content
func (reconstructor *Reconstructor) AppendCitations(content string, citations []ParsedCitation) string {
	if len(citations) == 0 {
		return content
	}

	var markdownLines []string
	markdownLines = append(markdownLines, strings.TrimSpace(content))

	for _, citation := range citations {
		reconstructor.reconstructNode(&Node{
			Type:           NodeFootnote,
			FootnoteNumber: citation.Number,
			Content:        citation.Description,
			SourceFile:     citation.File,
			SourcePages:    citation.Pages,
		}, &markdownLines)
	}

	result := strings.Join(markdownLines, "\n")
	return reconstructor.applyCitationPostProcessing(result)
}

func (reconstructor *Reconstructor) ensureBlankLine(markdownLines *[]string) {
	if len(*markdownLines) > 0 && (*markdownLines)[len(*markdownLines)-1] != "" {
		*markdownLines = append(*markdownLines, "")
	}
}

func (reconstructor *Reconstructor) reconstructNode(node *Node, markdownLines *[]string) {
	if node == nil {
		return
	}

	switch node.Type {
	case NodeDocument:
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, markdownLines)
		}

	case NodeSection:
		if node.Title != "" {
			reconstructor.ensureBlankLine(markdownLines)
			*markdownLines = append(*markdownLines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Title))
			*markdownLines = append(*markdownLines, "") // Force blank line after heading
		}
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, markdownLines)
		}

	case NodeParagraph:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, node.Content)

	case NodeText:
		if len(*markdownLines) > 0 && (*markdownLines)[len(*markdownLines)-1] != "" {
			// Append to the last line if it's not empty
			(*markdownLines)[len(*markdownLines)-1] = (*markdownLines)[len(*markdownLines)-1] + node.Content
		} else {
			reconstructor.ensureBlankLine(markdownLines)
			*markdownLines = append(*markdownLines, node.Content)
		}

	case NodeHeading:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Content))
		*markdownLines = append(*markdownLines, "") // Force blank line after heading

	case NodeListItem:
		// Items in a list don't strictly need blank lines between them unless they are "loose"
		// We'll follow the rule: top-level list (depth 0) first item gets a blank line before it.
		// Consecutive list items at the same depth don't get blank lines.
		indent := strings.Repeat(" ", node.Depth*reconstructor.indentUnit)
		bullet := "- "
		if node.ListType == ListOrdered {
			bullet = fmt.Sprintf("%d. ", node.Index)
		}

		// If it's a top-level list, ensure there's a blank line before the whole list
		if node.Depth == 0 && len(*markdownLines) > 0 && !strings.HasPrefix(strings.TrimSpace((*markdownLines)[len(*markdownLines)-1]), "-") && !regexp.MustCompile(`^\d+\.`).MatchString(strings.TrimSpace((*markdownLines)[len(*markdownLines)-1])) {
			reconstructor.ensureBlankLine(markdownLines)
		}

		*markdownLines = append(*markdownLines, fmt.Sprintf("%s%s%s", indent, bullet, node.Content))
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, markdownLines)
		}

	case NodeFootnote:
		reconstructor.ensureBlankLine(markdownLines)

		footnoteText := node.Content
		// Only append structured metadata if it's NOT already in the text
		// This prevents "Description (file.pdf, p. 1) (file.pdf, p. 1)"
		if node.SourceFile != "" && !strings.Contains(footnoteText, node.SourceFile) {
			pageInfo := ""
			if len(node.SourcePages) > 0 {
				formattedPages := FormatPageNumbers(node.SourcePages)
				if len(node.SourcePages) == 1 {
					pageInfo = getI18nLabel(reconstructor.Language, "page_label") + " " + formattedPages
				} else {
					pageInfo = getI18nLabel(reconstructor.Language, "pages_label") + " " + formattedPages
				}
			}
			if pageInfo != "" {
				footnoteText = fmt.Sprintf("%s (`%s`, %s)", footnoteText, node.SourceFile, pageInfo)
			} else {
				footnoteText = fmt.Sprintf("%s (`%s`)", footnoteText, node.SourceFile)
			}
		}

		*markdownLines = append(*markdownLines, fmt.Sprintf("[^%d]: %s", node.FootnoteNumber, footnoteText))

	case NodeTable:
		reconstructor.ensureBlankLine(markdownLines)
		reconstructor.reconstructTable(node, markdownLines)

	case NodeDisplayEquation:
		reconstructor.ensureBlankLine(markdownLines)
		if node.IsMultiline {
			*markdownLines = append(*markdownLines, "$$")
			*markdownLines = append(*markdownLines, node.Content)
			*markdownLines = append(*markdownLines, "$$")
		} else {
			*markdownLines = append(*markdownLines, fmt.Sprintf("$$%s$$", node.Content))
		}

	case NodeInlineMath:
		if len(*markdownLines) > 0 && (*markdownLines)[len(*markdownLines)-1] != "" {
			// Append to the last line
			(*markdownLines)[len(*markdownLines)-1] = (*markdownLines)[len(*markdownLines)-1] + fmt.Sprintf("$%s$", node.Content)
		} else {
			*markdownLines = append(*markdownLines, fmt.Sprintf("$%s$", node.Content))
		}

	case NodeCodeBlock:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, "```")
		*markdownLines = append(*markdownLines, node.Content)
		*markdownLines = append(*markdownLines, "```")

	case NodeHorizontalRule:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, "---")

	case NodeImage:
		if !reconstructor.IncludeImages {
			break // Skip images when IncludeImages is false
		}

		reconstructor.ensureBlankLine(markdownLines)

		// 1. Build the structured metadata part (Source File + Pages)
		metadataCaption := ""
		if node.SourceFile != "" {
			pageInfo := ""
			if len(node.SourcePages) > 0 {
				formattedPages := FormatPageNumbers(node.SourcePages)
				if len(node.SourcePages) == 1 {
					pageInfo = getI18nLabel(reconstructor.Language, "page_label") + " " + formattedPages
				} else {
					pageInfo = getI18nLabel(reconstructor.Language, "pages_label") + " " + formattedPages
				}
			}

			if pageInfo != "" {
				// Use <code> tags for figcaption (HTML block)
				metadataCaption = fmt.Sprintf("<code>%s</code>, %s", node.SourceFile, pageInfo)
			} else {
				metadataCaption = fmt.Sprintf("<code>%s</code>", node.SourceFile)
			}
		} else if node.Title != "" {
			metadataCaption = node.Title
		}

		// 2. Output as HTML figure
		*markdownLines = append(*markdownLines, "<figure>")
		*markdownLines = append(*markdownLines, fmt.Sprintf("  <img src=\"%s\" alt=\"\" />", node.Content))
		if metadataCaption != "" {
			*markdownLines = append(*markdownLines, fmt.Sprintf("  <figcaption>%s</figcaption>", strings.TrimSpace(metadataCaption)))
		}
		*markdownLines = append(*markdownLines, "</figure>")
	}
}

func (reconstructor *Reconstructor) reconstructTable(node *Node, markdownLines *[]string) {
	if len(node.Rows) == 0 {
		return
	}

	for _, tableRow := range node.Rows {
		// Escape dollar signs in table cells to avoid breaking some markdown renderers
		// and to satisfy robustness requirements.
		escapedCells := make([]string, len(tableRow.Cells))
		for i, cell := range tableRow.Cells {
			escapedCells[i] = strings.ReplaceAll(cell, "$", "\\$")
		}

		*markdownLines = append(*markdownLines, "| "+strings.Join(escapedCells, " | ")+" |")
		if tableRow.IsHeader {
			var align []string
			for range tableRow.Cells {
				align = append(align, "---")
			}
			*markdownLines = append(*markdownLines, "| "+strings.Join(align, " | ")+" |")
		}
	}
}
