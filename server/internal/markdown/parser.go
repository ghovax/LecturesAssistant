package markdown

import (
	"regexp"
	"strconv"
	"strings"
)

// Parser handles converting markdown text into an AST
type Parser struct {
	indentUnit int
}

// NewParser creates a new markdown parser
func NewParser() *Parser {
	return &Parser{
		indentUnit: 4, // Default
	}
}

// Parse converts markdown text into a hierarchical Document node
func (parser *Parser) Parse(markdown string) *Node {
	// STEP 1: Unwrap backtick-wrapped math expressions
	markdown = parser.unwrapBacktickMath(markdown)

	// STEP 2: Escape dollar signs in regular text
	markdown = parser.escapeDollarSigns(markdown)

	// STEP 3: Convert LaTeX-style math delimiters to markdown math
	markdown = parser.convertLatexMathDelimiters(markdown)

	lines := strings.Split(markdown, "\n")
	parser.indentUnit = parser.detectIndentationPattern(lines)

	var allElements []*Node
	for i := 0; i < len(lines); i++ {
		// Check for code blocks first
		if codeBlock, nextIndex := parser.parseCodeBlock(lines, i); codeBlock != nil {
			allElements = append(allElements, codeBlock)
			i = nextIndex
			continue
		}

		// Check for multi-line display equations
		if equation, nextIndex := parser.parseDisplayEquation(lines, i); equation != nil {
			allElements = append(allElements, equation)
			i = nextIndex
			continue
		}

		// Check for tables
		if table, nextIndex := parser.parseTable(lines, i); table != nil {
			allElements = append(allElements, table)
			i = nextIndex
			continue
		}

		// Check for multi-line footnotes
		if footnote, nextIndex := parser.parseFootnote(lines, i); footnote != nil {
			allElements = append(allElements, footnote)
			i = nextIndex
			continue
		}

		// Parse single-line elements
		if element := parser.parseMarkdownElement(lines[i]); element != nil {
			if element.Type == NodeParagraph {
				splitElements := parser.splitParagraphEquations(element)
				allElements = append(allElements, splitElements...)
			} else {
				allElements = append(allElements, element)
			}
		}
	}

	// Build list hierarchy
	nestedIndices := parser.buildListHierarchy(allElements)

	// Build hierarchical structure
	cleanElements := parser.removeNestedItems(allElements, nestedIndices)
	rootChildren := parser.buildSectionHierarchy(cleanElements)

	return &Node{
		Type:     NodeDocument,
		Children: rootChildren,
	}
}

func (parser *Parser) unwrapBacktickMath(markdown string) string {
	inlineRegex := regexp.MustCompile("(?s)`\\\\\\((.*?)\\\\\\)`")
	markdown = inlineRegex.ReplaceAllString(markdown, "\\($1\\)")

	displayRegex := regexp.MustCompile("(?s)`\\\\\\[(.*?)\\\\\\]`")
	markdown = displayRegex.ReplaceAllString(markdown, "\\[$1\\]")

	return markdown
}

func (parser *Parser) escapeDollarSigns(text string) string {
	var builder strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] == '$' {
			if i == 0 || text[i-1] != '\\' {
				builder.WriteString("\\$")
				continue
			}
		}
		builder.WriteByte(text[i])
	}
	return builder.String()
}

func (parser *Parser) convertLatexMathDelimiters(markdown string) string {
	inlineRegex := regexp.MustCompile("(?s)\\\\\\(.*?\\\\\\)")
	markdown = inlineRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		content := match[2 : len(match)-2]
		return "$" + strings.TrimSpace(content) + "$"
	})

	displayRegex := regexp.MustCompile("(?s)\\\\\\[.*?\\\\\\]")
	markdown = displayRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		content := match[2 : len(match)-2]
		return "$$" + strings.TrimSpace(content) + "$$"
	})

	standaloneRegex := regexp.MustCompile("(?m)^(\\s*)\\$([^$]+)\\$([.,]?)(\\s*)$")
	markdown = standaloneRegex.ReplaceAllString(markdown, "$1$$$2$3$$$4")

	return markdown
}

func (parser *Parser) detectIndentationPattern(lines []string) int {
	indentLevels := make(map[int]int)
	listRegex := regexp.MustCompile(`^(\s*)([*+-]|\d+\.)\s+`)

	for _, line := range lines {
		if match := listRegex.FindStringSubmatch(line); match != nil {
			indent := len(match[1])
			if indent > 0 {
				indentLevels[indent]++
			}
		}
	}

	if len(indentLevels) == 0 {
		return 4
	}

	// Simple logic: if 2-space pattern is present anywhere, assume 2-space
	if indentLevels[2] > 0 || indentLevels[6] > 0 {
		return 2
	}
	return 4
}

func (parser *Parser) parseMarkdownElement(line string) *Node {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" {
		return nil
	}

	headingRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	if match := headingRegex.FindStringSubmatch(trimmed); match != nil {
		return &Node{
			Type:    NodeHeading,
			Content: parser.cleanTitle(match[2]),
			Level:   len(match[1]),
		}
	}

	unorderedRegex := regexp.MustCompile(`^(\s*)[*+-]\s+(.+)$`)
	if match := unorderedRegex.FindStringSubmatch(line); match != nil {
		depth := len(match[1]) / parser.indentUnit
		return &Node{
			Type:     NodeListItem,
			Content:  match[2],
			Depth:    depth,
			ListType: ListUnordered,
		}
	}

	orderedRegex := regexp.MustCompile(`^(\s*)(\d+)\.\s+(.+)$`)
	if match := orderedRegex.FindStringSubmatch(line); match != nil {
		depth := len(match[1]) / parser.indentUnit
		index, _ := strconv.Atoi(match[2])
		return &Node{
			Type:     NodeListItem,
			Content:  match[3],
			Depth:    depth,
			ListType: ListOrdered,
			Index:    index,
		}
	}

	if match := regexp.MustCompile(`^\$([^$]+)\$$`).FindStringSubmatch(trimmed); match != nil {
		return &Node{
			Type:    NodeDisplayEquation,
			Content: strings.TrimSpace(match[1]),
		}
	}

	return &Node{
		Type:    NodeParagraph,
		Content: trimmed,
	}
}

func (parser *Parser) cleanTitle(title string) string {
	regex := regexp.MustCompile(`^(?:\d+|[IVXLCDM]+)\.\s*`)
	return regex.ReplaceAllString(title, "")
}

func (parser *Parser) buildListHierarchy(elements []*Node) map[int]bool {
	var stack []*Node
	nestedIndices := make(map[int]bool)
	for i, element := range elements {
		if element.Type == NodeListItem {
			for len(stack) > 0 && stack[len(stack)-1].Depth >= element.Depth {
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, element)
				nestedIndices[i] = true
			}
			stack = append(stack, element)
		}
	}
	return nestedIndices
}

func (parser *Parser) removeNestedItems(elements []*Node, nestedIndices map[int]bool) []*Node {
	var result []*Node
	for i, element := range elements {
		if nestedIndices[i] {
			continue
		}
		if element.Type == NodeFootnote {
			continue
		}
		result = append(result, element)
	}
	return result
}

func (parser *Parser) buildSectionHierarchy(elements []*Node) []*Node {
	var result []*Node
	var stack []*Node

	for _, element := range elements {
		if element.Type == NodeHeading {
			sectionNode := &Node{
				Type:  NodeSection,
				Title: element.Content,
				Level: element.Level,
			}

			for len(stack) > 0 && stack[len(stack)-1].Level >= element.Level {
				stack = stack[:len(stack)-1]
			}

			if len(stack) > 0 {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, sectionNode)
			} else {
				result = append(result, sectionNode)
			}
			stack = append(stack, sectionNode)
		} else {
			if len(stack) > 0 {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, element)
			} else {
				result = append(result, element)
			}
		}
	}
	return result
}

func (parser *Parser) parseCodeBlock(lines []string, startIndex int) (*Node, int) {
	line := strings.TrimSpace(lines[startIndex])
	if !strings.HasPrefix(line, "```") {
		return nil, startIndex
	}

	var codeLines []string
	for i := startIndex + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			return &Node{
				Type:    NodeCodeBlock,
				Content: strings.Join(codeLines, "\n"),
			}, i
		}
		codeLines = append(codeLines, lines[i])
	}
	return nil, startIndex
}

func (parser *Parser) parseDisplayEquation(lines []string, startIndex int) (*Node, int) {
	line := strings.TrimSpace(lines[startIndex])
	if line == "$$" {
		var equationLines []string
		for i := startIndex + 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "$$" {
				return &Node{
					Type:        NodeDisplayEquation,
					Content:     strings.Join(equationLines, "\n"),
					IsMultiline: true,
				}, i
			}
			equationLines = append(equationLines, lines[i])
		}
	}
	return nil, startIndex
}

func (parser *Parser) parseTable(lines []string, startIndex int) (*Node, int) {
	if startIndex+1 >= len(lines) {
		return nil, startIndex
	}
	headerLine := strings.TrimSpace(lines[startIndex])
	alignmentLine := strings.TrimSpace(lines[startIndex+1])

	if !strings.Contains(headerLine, "|") || !strings.Contains(alignmentLine, "|") {
		return nil, startIndex
	}

	if !regexp.MustCompile(`^[:\-| ]+$`).MatchString(alignmentLine) {
		return nil, startIndex
	}

	headerCells := parser.splitTableLine(headerLine)
	var rows []*TableRow
	rows = append(rows, &TableRow{Cells: headerCells, IsHeader: true})

	currentIndex := startIndex + 2
	for currentIndex < len(lines) {
		line := strings.TrimSpace(lines[currentIndex])
		if !strings.Contains(line, "|") || line == "" {
			break
		}
		cells := parser.splitTableLine(line)
		rows = append(rows, &TableRow{Cells: cells, IsHeader: false})
		currentIndex++
	}

	return &Node{
		Type: NodeTable,
		Rows: rows,
	}, currentIndex - 1
}

func (parser *Parser) splitTableLine(line string) []string {
	parts := strings.Split(strings.Trim(line, "|"), "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

func (parser *Parser) parseFootnote(lines []string, startIndex int) (*Node, int) {
	line := lines[startIndex]
	match := regexp.MustCompile(`^\[\^(\d+)\]:\s+(.+)$`).FindStringSubmatch(line)
	if match != nil {
		number, _ := strconv.Atoi(match[1])
		return &Node{
			Type:           NodeFootnote,
			Content:        match[2],
			FootnoteNumber: number,
		}, startIndex
	}
	return nil, startIndex
}

func (parser *Parser) splitParagraphEquations(paragraph *Node) []*Node {
	content := paragraph.Content
	var parts []*Node

	equationRegex := regexp.MustCompile(`\$\$([^$]+)\$\$`)
	matches := equationRegex.FindAllStringSubmatchIndex(content, -1)

	lastIndex := 0
	for _, match := range matches {
		textBefore := strings.TrimSpace(content[lastIndex:match[0]])
		if textBefore != "" {
			parts = append(parts, &Node{
				Type:    NodeParagraph,
				Content: textBefore,
			})
		}

		equationContent := strings.TrimSpace(content[match[2]:match[3]])
		parts = append(parts, &Node{
			Type:    NodeDisplayEquation,
			Content: equationContent,
		})

		lastIndex = match[1]
	}

	textAfter := strings.TrimSpace(content[lastIndex:])
	if textAfter != "" {
		parts = append(parts, &Node{
			Type:    NodeParagraph,
			Content: textAfter,
		})
	}

	if len(parts) == 0 {
		return []*Node{paragraph}
	}
	return parts
}
