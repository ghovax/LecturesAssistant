package markdown

import (
	"fmt"
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
	return strings.Join(lines, "\n")
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
			if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
				*lines = append(*lines, "")
			}
			*lines = append(*lines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Title))
			*lines = append(*lines, "")
		}
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, lines)
		}

	case NodeParagraph:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		*lines = append(*lines, node.Content)
		*lines = append(*lines, "")

	case NodeHeading:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		*lines = append(*lines, fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Content))
		*lines = append(*lines, "")

	case NodeListItem:
		indent := strings.Repeat(" ", node.Depth*reconstructor.indentUnit)
		bullet := "- "
		if node.ListType == ListOrdered {
			bullet = fmt.Sprintf("%d. ", node.Index)
		}
		*lines = append(*lines, fmt.Sprintf("%s%s%s", indent, bullet, node.Content))
		for _, child := range node.Children {
			reconstructor.reconstructNode(child, lines)
		}
		if node.Depth == 0 {
			*lines = append(*lines, "")
		}

	case NodeTable:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		reconstructor.reconstructTable(node, lines)
		*lines = append(*lines, "")

	case NodeDisplayEquation:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		if node.IsMultiline {
			*lines = append(*lines, "$$")
			*lines = append(*lines, node.Content)
			*lines = append(*lines, "$$")
		} else {
			*lines = append(*lines, fmt.Sprintf("$$%s$$", node.Content))
		}
		*lines = append(*lines, "")

	case NodeCodeBlock:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		*lines = append(*lines, "```")
		*lines = append(*lines, node.Content)
		*lines = append(*lines, "```")
		*lines = append(*lines, "")

	case NodeHorizontalRule:
		if len(*lines) > 0 && (*lines)[len(*lines)-1] != "" {
			*lines = append(*lines, "")
		}
		*lines = append(*lines, "---")
		*lines = append(*lines, "")
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
