package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// Reconstructor handles converting an AST back into markdown text
type Reconstructor struct {
	indentUnit int
	Language   string
}

// NewReconstructor creates a new markdown reconstructor
func NewReconstructor() *Reconstructor {
	return &Reconstructor{
		indentUnit: 4,
		Language:   "en",
	}
}

// Reconstruct converts an AST node (and its children) into markdown text
func (reconstructor *Reconstructor) Reconstruct(node *Node) string {
	var markdownLines []string
	reconstructor.reconstructNode(node, &markdownLines)

	result := strings.Join(markdownLines, "\n")

	// Remove spaces before footnote references: "text [^1]" -> "text[^1]"
	result = strings.ReplaceAll(result, " [^", "[^")

	return strings.TrimSpace(result) + "\n"
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

	return strings.Join(markdownLines, "\n")
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
		}
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, markdownLines)
		}

	case NodeParagraph:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, node.Content)

	case NodeHeading:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Content))

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

	case NodeCodeBlock:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, "```")
		*markdownLines = append(*markdownLines, node.Content)
		*markdownLines = append(*markdownLines, "```")

	case NodeHorizontalRule:
		reconstructor.ensureBlankLine(markdownLines)
		*markdownLines = append(*markdownLines, "---")
	}
}

func (reconstructor *Reconstructor) reconstructTable(node *Node, markdownLines *[]string) {
	if len(node.Rows) == 0 {
		return
	}

	for _, tableRow := range node.Rows {
		*markdownLines = append(*markdownLines, "| "+strings.Join(tableRow.Cells, " | ")+" |")
		if tableRow.IsHeader {
			var align []string
			for range tableRow.Cells {
				align = append(align, "---")
			}
			*markdownLines = append(*markdownLines, "| "+strings.Join(align, " | ")+" |")
		}
	}
}
