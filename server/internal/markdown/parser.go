package markdown

import (
	"log/slog"
	"regexp"
	"sort"
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
	// `\(...\)` -> \(...\)
	inlineRegex := regexp.MustCompile("(?s)`\\\\\\((.*?)\\\\\\)`")
	markdown = inlineRegex.ReplaceAllString(markdown, "\\($1\\)")

	// `\[...\]` -> \[...\]
	displayRegex := regexp.MustCompile("(?s)`\\\\\\[(.*?)\\\\\\]`")
	markdown = displayRegex.ReplaceAllString(markdown, "\\[$1\\]")

	return markdown
}

func (parser *Parser) escapeDollarSigns(text string) string {
	var builder strings.Builder
	runes := []rune(text)
	for i := range runes {
		if runes[i] == '$' {
			// Count backslashes before this dollar sign
			backslashCount := 0
			for j := i - 1; j >= 0 && runes[j] == '\\'; j-- {
				backslashCount++
			}
			// If backslash count is even, the dollar is unescaped
			if backslashCount%2 == 0 {
				builder.WriteRune('\\')
			}
		}
		builder.WriteRune(runes[i])
	}
	return builder.String()
}

func (parser *Parser) convertLatexMathDelimiters(markdown string) string {
	// \(...\) -> $...$
	inlineRegex := regexp.MustCompile(`(?s)\\\(.*?\\\)`)
	markdown = inlineRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		content := match[2 : len(match)-2]
		return "$" + strings.TrimSpace(content) + "$"
	})

	// \[...\] -> $$...$$
	displayRegex := regexp.MustCompile(`(?s)\\\[.*?\\\]`)
	markdown = displayRegex.ReplaceAllStringFunc(markdown, func(match string) string {
		content := match[2 : len(match)-2]
		// Preserve newlines for multi-line equations, only trim leading/trailing whitespace on same line
		if strings.Contains(content, "\n") {
			// Multi-line: preserve structure
			return "$$" + content + "$$"
		}
		// Single-line: trim spaces
		return "$$" + strings.TrimSpace(content) + "$$"
	})

	standaloneRegex := regexp.MustCompile(`(?m)^(\s*)\$([^$]+)\$([.,]?)(\s*)$`)
	markdown = standaloneRegex.ReplaceAllString(markdown, "$1$$$$$2$3$$$$$4")

	return markdown
}

func (parser *Parser) detectIndentationPattern(lines []string) int {
	indentLevels := make(map[int]int)
	// Match both unordered and ordered lists (optionally with emphasis markers)
	listRegex := regexp.MustCompile(`^(\s*)([*+-]|(?:\*{0,2}|_{0,2})\d+\.)\s+`)

	var sortedUniqueLevels []int
	for _, line := range lines {
		if match := listRegex.FindStringSubmatch(line); match != nil {
			indent := len(match[1])
			if indent > 0 {
				if indentLevels[indent] == 0 {
					sortedUniqueLevels = append(sortedUniqueLevels, indent)
				}
				indentLevels[indent]++
			}
		}
	}

	if len(indentLevels) == 0 {
		return 4
	}

	sort.Ints(sortedUniqueLevels)

	// Arithmetic progression check (from TS)
	if len(sortedUniqueLevels) >= 2 { // At least 2 levels needed to define a diff (0, diff, 2*diff)
		diff := sortedUniqueLevels[0] // First level relative to 0
		if len(sortedUniqueLevels) >= 2 {
			diff = sortedUniqueLevels[1] - sortedUniqueLevels[0]
		}

		isConsistent := true
		for i := 1; i < len(sortedUniqueLevels); i++ {
			if sortedUniqueLevels[i]-sortedUniqueLevels[i-1] != diff {
				isConsistent = false
				break
			}
		}
		if isConsistent && diff > 0 {
			return diff
		}
	}

	// Weighted frequency check
	totalCount := 0
	for _, count := range indentLevels {
		totalCount += count
	}

	twoSpaceScore := 0.0
	fourSpaceScore := 0.0

	for level, count := range indentLevels {
		weight := float64(count) / float64(totalCount)
		if level%4 == 0 {
			fourSpaceScore += weight
		} else if level%2 == 0 {
			twoSpaceScore += weight
		}
	}

	if fourSpaceScore > 0.6 || fourSpaceScore > twoSpaceScore {
		return 4
	}
	return 2
}

func (parser *Parser) splitByPipesOutsideMath(line string) []string {
	var cells []string
	var currentCell strings.Builder
	inInlineMath := false
	inDisplayMath := false

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Check for $$
		if char == '$' && i+1 < len(runes) && runes[i+1] == '$' {
			inDisplayMath = !inDisplayMath
			currentCell.WriteRune(char)
			currentCell.WriteRune(runes[i+1])
			i++
			continue
		}

		// Check for $
		if char == '$' && !inDisplayMath {
			inInlineMath = !inInlineMath
			currentCell.WriteRune(char)
			continue
		}

		// Split on | only if not inside math
		if char == '|' && !inInlineMath && !inDisplayMath {
			trimmed := strings.TrimSpace(currentCell.String())
			if trimmed != "" {
				cells = append(cells, trimmed)
			}
			currentCell.Reset()
		} else {
			currentCell.WriteRune(char)
		}
	}

	trimmed := strings.TrimSpace(currentCell.String())
	if trimmed != "" {
		cells = append(cells, trimmed)
	}

	return cells
}

func (parser *Parser) parseMarkdownElement(line string) *Node {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" {
		return nil
	}

	// Check for equation with footnote reference: $...$ [^N] or $$...$$ [^N]
	// Go regexp doesn't support backreferences (\1), so we use a more explicit approach
	if strings.Contains(trimmed, "[^") {
		// Try display equation with footnote
		if match := regexp.MustCompile(`^\$\$([^$]+)\$\$\s*(\[\^\d+\][.,]?)(.*)$`).FindStringSubmatch(trimmed); match != nil {
			return &Node{
				Type:    NodeDisplayEquation,
				Content: strings.TrimSpace(match[1]),
			}
		}
		// Try inline equation with footnote
		if match := regexp.MustCompile(`^\$([^$]+)\$\s*(\[\^\d+\][.,]?)(.*)$`).FindStringSubmatch(trimmed); match != nil {
			return &Node{
				Type:    NodeDisplayEquation,
				Content: strings.TrimSpace(match[1]),
			}
		}
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

	// Ordered list item (handles cases like "1. A", "*1. A:", and "**1. A:**")
	orderedRegex := regexp.MustCompile(`^(\s*)(\*{0,2}|_{0,2})(\d+)\.(\*{0,2}|_{0,2})\s+(.*)$`)
	if match := orderedRegex.FindStringSubmatch(line); match != nil {
		indentLength := len(match[1])
		prefixAsterisks := match[2]
		listIndex, _ := strconv.Atoi(match[3])
		suffixAsterisks := match[4]
		content := prefixAsterisks + suffixAsterisks + match[5]

		indentUnit := parser.indentUnit
		if indentUnit == 0 {
			indentUnit = 4
		}
		depth := indentLength / indentUnit

		return &Node{
			Type:     NodeListItem,
			Content:  content,
			Depth:    depth,
			ListType: ListOrdered,
			Index:    listIndex,
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

	headerCells := parser.splitByPipesOutsideMath(headerLine)
	var rows []*TableRow
	rows = append(rows, &TableRow{Cells: headerCells, IsHeader: true})

	currentIndex := startIndex + 2
	for currentIndex < len(lines) {
		line := strings.TrimSpace(lines[currentIndex])
		if !strings.Contains(line, "|") || line == "" {
			break
		}
		cells := parser.splitByPipesOutsideMath(line)
		rows = append(rows, &TableRow{Cells: cells, IsHeader: false})
		currentIndex++
	}

	return &Node{
		Type: NodeTable,
		Rows: rows,
	}, currentIndex - 1
}

func (parser *Parser) parseFootnote(lines []string, startIndex int) (*Node, int) {
	line := strings.TrimSpace(lines[startIndex])
	// Match [^N]: Content
	match := regexp.MustCompile(`^\[\^(\d+)\]:\s+(.+)$`).FindStringSubmatch(line)
	if match != nil {
		number, _ := strconv.Atoi(match[1])
		fullContent := strings.TrimSpace(match[2])

		// Try to extract metadata if present: Content (`file.pdf` , pp. 1–2)
		// We'll use a more flexible regex that handles optional commas and spaces.
		metadataRegex := regexp.MustCompile(`^(.*?)\s*\(\s*\x60(.*?)\x60\s*(?:,\s*)?(p{1,2}\.\s*([\d–\-, ]+))?\s*\)$`)
		metaMatch := metadataRegex.FindStringSubmatch(fullContent)
		slog.Info("Footnote metadata match", "content", fullContent, "match", metaMatch)

		if metaMatch != nil {
			content := strings.TrimSpace(metaMatch[1])
			filename := strings.TrimSpace(metaMatch[2])
			pageString := ""
			if len(metaMatch) > 4 && metaMatch[4] != "" {
				pageString = metaMatch[4]
			}

			return &Node{
				Type:           NodeFootnote,
				FootnoteNumber: number,
				Content:        content,
				SourceFile:     filename,
				SourcePages:    ParsePageString(pageString),
			}, startIndex
		}

		return &Node{
			Type:           NodeFootnote,
			Content:        fullContent,
			FootnoteNumber: number,
		}, startIndex
	}
	return nil, startIndex
}

func (parser *Parser) splitParagraphEquations(paragraph *Node) []*Node {
	content := paragraph.Content
	var parts []*Node

	// Match escaped dollar signs (\$\$...\$\$) in the content
	equationRegex := regexp.MustCompile(`\\\$\\\$([^\$\\]+)\\\$\\\$`)
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
