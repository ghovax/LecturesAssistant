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
	markdown = parser.unwrapBacktickMath(markdown)
	markdown = parser.convertLatexMathDelimiters(markdown)

	lines := strings.Split(markdown, "\n")
	parser.indentUnit = parser.detectIndentationPattern(lines)

	var allElements []*Node
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		// Check for code blocks first
		if codeBlock, nextIndex := parser.parseCodeBlock(lines, lineIndex); codeBlock != nil {
			allElements = append(allElements, codeBlock)
			lineIndex = nextIndex
			continue
		}

		// Check for multi-line display equations
		if equation, nextIndex := parser.parseDisplayEquation(lines, lineIndex); equation != nil {
			allElements = append(allElements, equation)
			lineIndex = nextIndex
			continue
		}

		// Check for tables
		if table, nextIndex := parser.parseTable(lines, lineIndex); table != nil {
			allElements = append(allElements, table)
			lineIndex = nextIndex
			continue
		}

		// Check for multi-line footnotes
		if footnote, nextIndex := parser.parseFootnote(lines, lineIndex); footnote != nil {
			allElements = append(allElements, footnote)
			lineIndex = nextIndex
			continue
		}

		// Parse single-line elements
		if element := parser.parseMarkdownElement(lines[lineIndex]); element != nil {
			if element.Type == NodeParagraph {
				// Split equations in paragraphs
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

func (parser *Parser) escapeUnescapedDollars(text string) string {
	var builder strings.Builder
	runes := []rune(text)
	for runeIndex := range runes {
		if runes[runeIndex] == '$' {
			// Count backslashes before this dollar sign
			backslashCount := 0
			for backslashIndex := runeIndex - 1; backslashIndex >= 0 && runes[backslashIndex] == '\\'; backslashIndex-- {
				backslashCount++
			}
			// If backslash count is even, the dollar is unescaped
			if backslashCount%2 == 0 {
				builder.WriteRune('\\')
			}
		}
		builder.WriteRune(runes[runeIndex])
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
		// Preserve newlines for multi-line equations
		return "$$" + content + "$$"
	})

	// Upgrade standalone $...$ to $$...$$
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

	// Arithmetic progression check
	if len(sortedUniqueLevels) >= 1 {
		diff := sortedUniqueLevels[0]
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
	for runeIndex := 0; runeIndex < len(runes); runeIndex++ {
		character := runes[runeIndex]

		// Check for $$
		if character == '$' && runeIndex+1 < len(runes) && runes[runeIndex+1] == '$' {
			inDisplayMath = !inDisplayMath
			currentCell.WriteRune(character)
			currentCell.WriteRune(runes[runeIndex+1])
			runeIndex++
			continue
		}

		// Check for $
		if character == '$' && !inDisplayMath {
			inInlineMath = !inInlineMath
			currentCell.WriteRune(character)
			continue
		}

		// Split on | only if not inside math
		if character == '|' && !inInlineMath && !inDisplayMath {
			trimmed := strings.TrimSpace(currentCell.String())
			if trimmed != "" {
				cells = append(cells, trimmed)
			}
			currentCell.Reset()
		} else {
			currentCell.WriteRune(character)
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
		level := len(match[1])
		title := parser.cleanTitle(match[2])
		slog.Debug("Parsed heading", "level", level, "title", title, "raw", trimmed)
		return &Node{
			Type:    NodeHeading,
			Content: title,
			Level:   level,
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
	// Robust structural regex for heading prefixes:
	// 1. ^([[:alpha:]]+\s+)?      - Optional single word followed by space (e.g., "Chapter ", "Section ")
	// 2. (\d+|[IVXLCDM]+|[A-Z])   - Mandatory Index: Digits, Roman Numerals, or a single Letter
	// 3. [\.\:\)]                 - Mandatory "Hard" Separator: Dot, Colon, or Closing Parenthesis
	// 4. \s+                      - Mandatory trailing space(s) before the actual title content
	regex := regexp.MustCompile(`(?i)^([[:alpha:]]+\s+)?(\d+|[IVXLCDM]+|[A-Z])[\.\:\)]\s+`)

	cleaned := regex.ReplaceAllString(title, "")

	// Fallback: If stripping resulted in an empty string (meaning the title was ONLY the prefix),
	// return the original trimmed title.
	if cleaned == "" {
		return strings.TrimSpace(title)
	}

	return strings.TrimSpace(cleaned)
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
	for lineIndex := startIndex + 1; lineIndex < len(lines); lineIndex++ {
		if strings.TrimSpace(lines[lineIndex]) == "```" {
			return &Node{
				Type:    NodeCodeBlock,
				Content: strings.Join(codeLines, "\n"),
			}, lineIndex
		}
		codeLines = append(codeLines, lines[lineIndex])
	}
	return nil, startIndex
}

func (parser *Parser) parseDisplayEquation(lines []string, startIndex int) (*Node, int) {
	line := strings.TrimSpace(lines[startIndex])
	if line == "$$" {
		var equationLines []string
		for lineIndex := startIndex + 1; lineIndex < len(lines); lineIndex++ {
			if strings.TrimSpace(lines[lineIndex]) == "$$" {
				return &Node{
					Type:        NodeDisplayEquation,
					Content:     strings.Join(equationLines, "\n"),
					IsMultiline: true,
				}, lineIndex
			}
			equationLines = append(equationLines, lines[lineIndex])
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

		// Robust metadata extraction:
		// Expects: "Description (`file.pdf`, p. 1)" or "Description (file.pdf p. 1)"
		// or even just "Description (file.pdf)"
		// 1. (.*?) - The description
		// 2. \(\s* - Opening parenthesis
		// 3. [\x60]? - Optional backtick
		// 4. ([^\x60,\s)]+) - The filename (anything not a backtick, comma, space, or closing paren)
		// 5. [\x60]? - Optional backtick
		// 6. (?:(?:\s*,\s*)|\s+)? - Optional separator (comma or space)
		// 7. (?:([a-zA-Z]{1,2}\.?\s*([\d–\-, ]+)))? - Optional page info (e.g., p. 1, S. 5)
		// 8. \s*\)$ - Closing parenthesis
		metadataRegex := regexp.MustCompile(`^(.*?)\s*\(\s*[\x60]?([^\x60,\s)]+)[\x60]?(?:(?:\s*,\s*)|\s+)?(?:([a-zA-Z]{1,2}\.?\s*([\d–\-, ]+)))?\s*\)$`)
		metaMatch := metadataRegex.FindStringSubmatch(fullContent)
		slog.Debug("Footnote metadata match", "content", fullContent, "matched", metaMatch != nil)

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

	// Match dollar signs ($...$ or $$...$$) in the content
	// We need to be careful with ordering: check for $$ first
	equationRegex := regexp.MustCompile(`(\$\${1,2})([^\$]+)(\$\${1,2})|(\$)([^\$]+)(\$)`)
	matches := equationRegex.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		return []*Node{paragraph}
	}

	lastIndex := 0
	for _, match := range matches {
		textBefore := content[lastIndex:match[0]]
		if textBefore != "" {
			parts = append(parts, &Node{
				Type:    NodeText,
				Content: textBefore,
			})
		}

		isDisplay := false
		var equationContent string
		if match[2] != -1 { // Group 1 (\$\${1,2}) matched something that might be $$
			delim := content[match[2]:match[3]]
			if delim == "$$" {
				isDisplay = true
				equationContent = strings.TrimSpace(content[match[4]:match[5]])
			} else {
				// It matched $ but via the first group? Should not happen with the regex above but let's be safe
				equationContent = strings.TrimSpace(content[match[4]:match[5]])
			}
		} else { // Group 7 ($) matched
			equationContent = strings.TrimSpace(content[match[10]:match[11]])
		}

		nodeType := NodeInlineMath
		if isDisplay {
			nodeType = NodeDisplayEquation
		}

		parts = append(parts, &Node{
			Type:    nodeType,
			Content: equationContent,
		})

		lastIndex = match[1]
	}

	textAfter := content[lastIndex:]
	if textAfter != "" {
		parts = append(parts, &Node{
			Type:    NodeText,
			Content: textAfter,
		})
	}

	if len(parts) == 0 {
		return []*Node{paragraph}
	}
	return parts
}
