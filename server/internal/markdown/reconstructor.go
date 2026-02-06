package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// Reconstructor handles converting an AST back into markdown text
type Reconstructor struct {
	indentUnit int
}

// NewReconstructor creates a new markdown reconstructor
func NewReconstructor() *Reconstructor {
	return &Reconstructor{
		indentUnit: 4,
	}
}

// Reconstruct converts an AST node (and its children) into markdown text
func (reconstructor *Reconstructor) Reconstruct(node *Node) string {
	var lines []string
	reconstructor.reconstructNode(node, &lines)

	result := strings.Join(lines, "\n")

	// Remove spaces before footnote references: "text [^1]" -> "text[^1]"
	result = strings.ReplaceAll(result, " [^", "[^")

	return strings.TrimSpace(result) + "\n"
}

// AppendCitations appends footnote definitions to the end of the markdown content
func (reconstructor *Reconstructor) AppendCitations(content string, citations []ParsedCitation) string {
	if len(citations) == 0 {
		return content
	}

	var lines []string
	lines = append(lines, strings.TrimSpace(content))

	for _, citation := range citations {
		reconstructor.reconstructNode(&Node{
			Type:           NodeFootnote,
			FootnoteNumber: citation.Number,
			Content:        citation.Description,
			SourceFile:     citation.File,
			SourcePages:    citation.Pages,
		}, &lines)
	}

	return strings.Join(lines, "\n")
}

func (reconstructor *Reconstructor) ensureBlankLine(lines *[]string) {
	if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
		*lines = append(*lines, "")
	}
}

func (reconstructor *Reconstructor) reconstructNode(node *Node, lines *[]string) {
	if node == nil {
		return
	}

	switch node.Type {
	case NodeDocument:
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, lines)
		}

	case NodeSection:
		if node.Title != "" {
			reconstructor.ensureBlankLine(lines)
			*lines = append(*lines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Title))
		}
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, lines)
		}

	case NodeParagraph:
		reconstructor.ensureBlankLine(lines)
		*lines = append(*lines, node.Content)

	case NodeHeading:
		reconstructor.ensureBlankLine(lines)
		*lines = append(*lines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Content))

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
		if node.Depth == 0 && len(*lines) > 0 && !strings.HasPrefix(strings.TrimSpace((*lines)[len(*lines)-1]), "-") && !regexp.MustCompile(`^\d+\.`).MatchString(strings.TrimSpace((*lines)[len(*lines)-1])) {
			reconstructor.ensureBlankLine(lines)
		}

		*lines = append(*lines, fmt.Sprintf("%s%s%s", indent, bullet, node.Content))
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, lines)
		}

	case NodeFootnote:
		reconstructor.ensureBlankLine(lines)

		footnoteText := node.Content
		if node.SourceFile != "" {
			pageInfo := ""
			if len(node.SourcePages) > 0 {
				formattedPages := FormatPageNumbers(node.SourcePages)
				if len(node.SourcePages) == 1 {
					pageInfo = "p. " + formattedPages
				} else {
					pageInfo = "pp. " + formattedPages
				}
			}
			if pageInfo != "" {
				footnoteText = fmt.Sprintf("%s (`%s` %s)", footnoteText, node.SourceFile, pageInfo)
			} else {
				footnoteText = fmt.Sprintf("%s (`%s`)", footnoteText, node.SourceFile)
			}
		}

		*lines = append(*lines, fmt.Sprintf("[^%d]: %s", node.FootnoteNumber, footnoteText))

	case NodeTable:
		reconstructor.ensureBlankLine(lines)
		reconstructor.reconstructTable(node, lines)

	case NodeDisplayEquation:
		reconstructor.ensureBlankLine(lines)
		if node.IsMultiline {
			*lines = append(*lines, "$$")
			*lines = append(*lines, node.Content)
			*lines = append(*lines, "$$")
		} else {
			*lines = append(*lines, fmt.Sprintf("$$%s$$", node.Content))
		}

	case NodeCodeBlock:
		reconstructor.ensureBlankLine(lines)
		*lines = append(*lines, "```")
		*lines = append(*lines, node.Content)
		*lines = append(*lines, "```")

	case NodeHorizontalRule:
		reconstructor.ensureBlankLine(lines)
		*lines = append(*lines, "---")
	}
}

func (reconstructor *Reconstructor) reconstructTable(node *Node, lines *[]string) {
	if len(node.Rows) == 0 {
		return
	}

	for _, row := range node.Rows {
		*lines = append(*lines, "| "+strings.Join(row.Cells, " | ")+" |")
		if row.IsHeader {
			var align []string
			for range row.Cells {
				align = append(align, "---")
			}
			*lines = append(*lines, "| "+strings.Join(align, " | ")+" |")
		}
	}
}
