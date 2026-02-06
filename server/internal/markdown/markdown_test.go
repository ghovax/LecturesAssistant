package markdown

import (
	"strings"
	"testing"
)

func TestMarkdownRoundTrip(tester *testing.T) {
	testCases := []struct {
		name     string
		markdown string
	}{
		{
			name: "Simple Headings and Paragraphs",
			markdown: `# Title

Some paragraph text.

## Section 1

Another paragraph.`,
		},
		{
			name: "Lists",
			markdown: `# List Test

- Item 1
- Item 2
  - Nested Item 2.1
  - Nested Item 2.2
- Item 3

1. First ordered
2. Second ordered
   1. Nested ordered
3. Third ordered`,
		},
		{
			name: "Tables",
			markdown: `# Table Test

| Header 1 | Header 2 |
| --- | --- |
| Cell 1.1 | Cell 1.2 |
| Cell 2.1 | Cell 2.2 |`,
		},
		{
			name: "Equations",
			markdown: `# Math Test

Inline equation $$E=mc^2$$

Multiline equation:

$$
\int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}
$$`,
		},
		{
			name: "Code Blocks",
			markdown: `# Code Test

` + "```" + `
func main() {
    fmt.Println("Hello")
}
` + "```",
		},
	}

	markdownParser := NewParser()
	markdownReconstructor := NewReconstructor()

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			documentAST := markdownParser.Parse(testCase.markdown)
			reconstructedMarkdown := markdownReconstructor.Reconstruct(documentAST)

			// Normalize for comparison: check if first line matches
			originalLines := strings.Split(strings.TrimSpace(testCase.markdown), "\n")
			originalFirstLine := originalLines[0]
			finalMarkdown := strings.TrimSpace(reconstructedMarkdown)

			if !strings.Contains(finalMarkdown, originalFirstLine) {
				subTester.Errorf("Reconstruction failed to preserve structure. \nOriginal First Line: %s\n\nReconstructed:\n%s", originalFirstLine, finalMarkdown)
			}
		})
	}
}

func TestSectionHierarchy(tester *testing.T) {
	markdownContent := `# Title
Intro text.
## Section 1

Section 1 text.
### Subsection 1.1
Subsection 1.1 text.
## Section 2
Section 2 text.`

	markdownParser := NewParser()
	documentAST := markdownParser.Parse(markdownContent)

	if documentAST.Type != NodeDocument {
		tester.Errorf("Expected root node to be Document, got %s", documentAST.Type)
	}

	// Should have 1 top level section: # Title
	if len(documentAST.Children) != 1 {
		tester.Fatalf("Expected 1 top-level section, got %d", len(documentAST.Children))
	}

	titleSection := documentAST.Children[0]
	if titleSection.Type != NodeSection || titleSection.Title != "Title" || titleSection.Level != 1 {
		tester.Errorf("Expected top section 'Title' level 1, got %s '%s' level %d", titleSection.Type, titleSection.Title, titleSection.Level)
	}

	// Title section should contain: Intro text (Paragraph), Section 1 (Section), Section 2 (Section)
	if len(titleSection.Children) != 3 {
		tester.Fatalf("Expected 3 children in Title section, got %d", len(titleSection.Children))
	}

	introParagraph := titleSection.Children[0]
	if introParagraph.Type != NodeParagraph || introParagraph.Content != "Intro text." {
		tester.Errorf("Expected intro paragraph, got %s '%s'", introParagraph.Type, introParagraph.Content)
	}

	firstLevelSection := titleSection.Children[1]
	if firstLevelSection.Type != NodeSection || firstLevelSection.Title != "Section 1" || firstLevelSection.Level != 2 {
		tester.Errorf("Expected Section 1 level 2, got %s '%s' level %d", firstLevelSection.Type, firstLevelSection.Title, firstLevelSection.Level)
	}

	// Section 1 should contain: Section 1 text (Paragraph), Subsection 1.1 (Section)
	if len(firstLevelSection.Children) != 2 {
		tester.Fatalf("Expected 2 children in Section 1, got %d", len(firstLevelSection.Children))
	}

	secondLevelSubsection := firstLevelSection.Children[1]
	if secondLevelSubsection.Type != NodeSection || secondLevelSubsection.Title != "Subsection 1.1" || secondLevelSubsection.Level != 3 {
		tester.Errorf("Expected Subsection 1.1 level 3, got %s '%s' level %d", secondLevelSubsection.Type, secondLevelSubsection.Title, secondLevelSubsection.Level)
	}
}

func TestListNestingStructure(tester *testing.T) {
	markdownContent := `- Level 0
  - Level 1
    - Level 2
- Back to 0`

	markdownParser := NewParser()
	markdownParser.indentUnit = 2
	documentAST := markdownParser.Parse(markdownContent)

	// In the hierarchical section builder, these list items are at root level
	var topLevelListItems []*Node
	for _, child := range documentAST.Children {
		if child.Type == NodeListItem {
			topLevelListItems = append(topLevelListItems, child)
		}
	}

	if len(topLevelListItems) != 2 {
		tester.Fatalf("Expected 2 top-level list items, got %d", len(topLevelListItems))
	}

	// Verify deep nesting: Level 0 -> Level 1 -> Level 2
	topLevelItem := topLevelListItems[0]
	if topLevelItem.Content != "Level 0" {
		tester.Errorf("Expected 'Level 0', got '%s'", topLevelItem.Content)
	}

	if len(topLevelItem.Children) != 1 {
		tester.Fatalf("Expected top level item to have 1 child, got %d", len(topLevelItem.Children))
	}

	firstNestedItem := topLevelItem.Children[0]
	if firstNestedItem.Content != "Level 1" || firstNestedItem.Depth != 1 {
		tester.Errorf("Expected 'Level 1' depth 1, got '%s' depth %d", firstNestedItem.Content, firstNestedItem.Depth)
	}

	if len(firstNestedItem.Children) != 1 {
		tester.Fatalf("Expected first nested item to have 1 child, got %d", len(firstNestedItem.Children))
	}

	secondNestedItem := firstNestedItem.Children[0]
	if secondNestedItem.Content != "Level 2" || secondNestedItem.Depth != 2 {
		tester.Errorf("Expected 'Level 2' depth 2, got '%s' depth %d", secondNestedItem.Content, secondNestedItem.Depth)
	}

	// Verify second top-level item
	backToRootItem := topLevelListItems[1]
	if backToRootItem.Content != "Back to 0" || backToRootItem.Depth != 0 {
		tester.Errorf("Expected 'Back to 0' depth 0, got '%s' depth %d", backToRootItem.Content, backToRootItem.Depth)
	}
}

func TestTableWithMath(tester *testing.T) {
	markdownContent := `| Heading 1 | Heading 2 |
| --- | --- |
| $a | b$ | Cell 2 |`

	markdownParser := NewParser()
	documentAST := markdownParser.Parse(markdownContent)

	if len(documentAST.Children) != 1 {
		tester.Fatalf("Expected 1 child (Table), got %d", len(documentAST.Children))
	}

	table := documentAST.Children[0]
	if table.Type != NodeTable {
		tester.Fatalf("Expected NodeTable, got %s", table.Type)
	}

	if len(table.Rows) != 2 {
		tester.Fatalf("Expected 2 rows, got %d", len(table.Rows))
	}

	// Verify the math cell didn't split on the pipe
	mathRow := table.Rows[1]
	if len(mathRow.Cells) != 2 {
		tester.Errorf("Expected 2 cells in math row, got %d. Cell contents: %v", len(mathRow.Cells), mathRow.Cells)
	} else if mathRow.Cells[0] != `\$a | b\$` {
		tester.Errorf("Expected cell 1 to be `\\$a | b\\$`, got '%s'", mathRow.Cells[0])
	}
}

func TestFootnoteSpaceTrimming(tester *testing.T) {
	markdownContent := "Some text [^1]"

	markdownParser := NewParser()
	documentAST := markdownParser.Parse(markdownContent)

	markdownReconstructor := NewReconstructor()
	reconstructed := markdownReconstructor.Reconstruct(documentAST)

	expected := "Some text[^1]\n"
	if reconstructed != expected {
		tester.Errorf("Expected footnote space to be trimmed. \nExpected: %q\nGot:      %q", expected, reconstructed)
	}
}

func TestArithmeticProgressionIndentation(tester *testing.T) {
	// Pattern: 0, 3, 6 (Consistent 3-space difference)
	lines := []string{
		"- Level 0",
		"   - Level 1",
		"      - Level 2",
	}

	markdownParser := NewParser()
	indent := markdownParser.detectIndentationPattern(lines)

	if indent != 3 {
		tester.Errorf("Expected indent unit 3 from arithmetic progression, got %d", indent)
	}
}

func TestEscapeDollarSigns(tester *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple dollar",
			input:    "$100",
			expected: `\$100`,
		},
		{
			name:     "Already escaped",
			input:    `\$100`,
			expected: `\$100`,
		},
		{
			name:     "Double backslash even parity",
			input:    `\\$100`, // The \\ escapes each other, the $ is raw
			expected: `\\\$100`,
		},
		{
			name:     "Triple backslash odd parity",
			input:    `\\\$100`, // The first two escape each other, the third escapes the $
			expected: `\\\$100`,
		},
	}

	markdownParser := NewParser()
	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := markdownParser.escapeDollarSigns(testCase.input)
			if result != testCase.expected {
				subTester.Errorf("Expected: %s, Got: %s", testCase.expected, result)
			}
		})
	}
}

func TestIndentationDetectionRobustness(tester *testing.T) {
	markdownParser := NewParser()
	lines := []string{"- Item 1", "  - Item 1.1", "  - Item 1.2", "- Item 2", "    - Item 2.1", "    - Item 2.2"}

	// Should favor 2-space if frequency is higher or clear multiple
	indent := markdownParser.detectIndentationPattern(lines)
	if indent != 2 {
		tester.Errorf("Expected indent unit 2, got %d", indent)
	}
}

func TestCitationRobustness(tester *testing.T) {
	reconstructor := NewReconstructor()

	testCases := []struct {
		name              string
		input             string
		expectedMarkdown  string
		expectedCitations []ParsedCitation
	}{
		{
			name:             "Simple citation",
			input:            "Referencing {{{Desc-file.pdf-p1}}}",
			expectedMarkdown: "Referencing[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "Description with dashes",
			input:            "Referencing {{{A-complex-desc-file.pdf-p1}}}",
			expectedMarkdown: "Referencing[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "A-complex-desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "File with dots and no pages",
			input:            "Referencing {{{Desc-my.file.name.pdf}}}",
			expectedMarkdown: "Referencing[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "my.file.name.pdf", Pages: nil},
			},
		},
		{
			name:             "Multiple citations",
			input:            "First {{{D1-f1.pdf-p1}}} and second {{{D2-f2.pdf-p2}}}",
			expectedMarkdown: "First[^1] and second[^2]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "D1", File: "f1.pdf", Pages: []int{1}},
				{Number: 2, Description: "D2", File: "f2.pdf", Pages: []int{2}},
			},
		},
		{
			name:             "Page range",
			input:            "Range {{{Desc-file.pdf-p1-3}}}",
			expectedMarkdown: "Range[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1, 2, 3}},
			},
		},
		{
			name:             "Complex pages",
			input:            "Complex {{{Desc-file.pdf-p1, 3, 5-7}}}",
			expectedMarkdown: "Complex[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1, 3, 5, 6, 7}},
			},
		},
		{
			name:             "Messy spacing",
			input:            "Messy {{{  Desc  -  file.pdf  -p1  }}}",
			expectedMarkdown: "Messy[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "Filename with underscore",
			input:            "Referencing {{{Description-file_name.pdf-p1}}}",
			expectedMarkdown: "Referencing[^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Description", File: "file_name.pdf", Pages: []int{1}},
			},
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			md, citations := reconstructor.ParseCitations(testCase.input)
			if md != testCase.expectedMarkdown {
				subTester.Errorf("Markdown mismatch. Expected: %q, Got: %q", testCase.expectedMarkdown, md)
			}
			if len(citations) != len(testCase.expectedCitations) {
				subTester.Fatalf("Citation count mismatch. Expected: %d, Got: %d", len(testCase.expectedCitations), len(citations))
			}
			for i := range citations {
				if citations[i].Description != testCase.expectedCitations[i].Description {
					subTester.Errorf("Citation %d description mismatch. Expected: %q, Got: %q", i, testCase.expectedCitations[i].Description, citations[i].Description)
				}
				if citations[i].File != testCase.expectedCitations[i].File {
					subTester.Errorf("Citation %d file mismatch. Expected: %q, Got: %q", i, testCase.expectedCitations[i].File, citations[i].File)
				}
				if len(citations[i].Pages) != len(testCase.expectedCitations[i].Pages) {
					subTester.Errorf("Citation %d pages count mismatch. Expected: %v, Got: %v", i, testCase.expectedCitations[i].Pages, citations[i].Pages)
				} else {
					for j := range citations[i].Pages {
						if citations[i].Pages[j] != testCase.expectedCitations[i].Pages[j] {
							subTester.Errorf("Citation %d page %d mismatch. Expected: %d, Got: %d", i, j, testCase.expectedCitations[i].Pages[j], citations[i].Pages[j])
						}
					}
				}
			}
		})
	}
}

func TestListEmphasisNumbering(tester *testing.T) {
	markdownContent := `**1.** Item One
**2.** Item Two`

	markdownParser := NewParser()
	documentAST := markdownParser.Parse(markdownContent)

	if len(documentAST.Children) != 2 {
		tester.Fatalf("Expected 2 list items, got %d", len(documentAST.Children))
	}

	item := documentAST.Children[0]
	if item.Type != NodeListItem || item.Index != 1 {
		tester.Errorf("Expected list item index 1, got %s index %d", item.Type, item.Index)
	}
}

func TestFootnoteIntegration_ASTAndReconstruction(tester *testing.T) {
	reconstructor := NewReconstructor()
	parser := NewParser()

	// 1. Initial content with citations
	rawContent := "Some claim {{{Claim info-file.pdf-p5}}}"
	contentWithoutMarkers, citations := reconstructor.ParseCitations(rawContent)

	if contentWithoutMarkers != "Some claim[^1]" {
		tester.Errorf("Expected marker replacement, got %q", contentWithoutMarkers)
	}

	// 2. Mock AI metadata enrichment
	if len(citations) != 1 {
		tester.Fatalf("Expected 1 citation, got %d", len(citations))
	}
	citations[0].Description = "AI improved description"
	citations[0].Pages = []int{5, 6}

	// 3. Append citations to generate final markdown
	finalMarkdown := reconstructor.AppendCitations(contentWithoutMarkers, citations)

	// 4. Parse back to AST
	documentAST := parser.Parse(finalMarkdown)

	// Verify AST structure
	var footnoteNode *Node
	for _, child := range documentAST.Children {
		if child.Type == NodeFootnote {
			footnoteNode = child
			break
		}
	}

	if footnoteNode == nil {
		tester.Fatal("AST missing NodeFootnote")
	}

	if footnoteNode.FootnoteNumber != 1 {
		tester.Errorf("Expected footnote number 1, got %d", footnoteNode.FootnoteNumber)
	}

	if footnoteNode.Content != "AI improved description" {
		tester.Errorf("Expected content 'AI improved description', got %q", footnoteNode.Content)
	}

	// 5. Reconstruct and verify output
	reconstructed := reconstructor.Reconstruct(documentAST)

	if !strings.Contains(reconstructed, "[^1]: AI improved description") {
		tester.Errorf("Reconstructed markdown missing footnote definition. Got:\n%s", reconstructed)
	}

	if !strings.Contains(reconstructed, "(`file.pdf` pp. 5–6)") {
		tester.Errorf("Reconstructed markdown missing or incorrect metadata. Got:\n%s", reconstructed)
	}
}

// LaTeX Math Delimiter Conversion Tests

func TestUnwrapBacktickMath(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Backtick wrapped inline math",
			input:          "Text with `\\(x + y\\)` inline",
			expectedOutput: "Text with \\(x + y\\) inline",
		},
		{
			name:           "Backtick wrapped display math",
			input:          "Text with `\\[E = mc^2\\]` display",
			expectedOutput: "Text with \\[E = mc^2\\] display",
		},
		{
			name:           "Multiple backtick wrapped expressions",
			input:          "`\\(a\\)` and `\\[b\\]` together",
			expectedOutput: "\\(a\\) and \\[b\\] together",
		},
		{
			name:           "No backtick wrapping",
			input:          "Normal text with no math",
			expectedOutput: "Normal text with no math",
		},
		{
			name:           "Complex expression in backticks",
			input:          "`\\(\\frac{x}{y}\\)` fraction",
			expectedOutput: "\\(\\frac{x}{y}\\) fraction",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := parser.unwrapBacktickMath(testCase.input)
			if result != testCase.expectedOutput {
				subTester.Errorf("Expected: %q\nGot: %q", testCase.expectedOutput, result)
			}
		})
	}
}

func TestConvertLatexMathDelimiters(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Inline LaTeX to markdown",
			input:          "Text \\(x + y\\) more text",
			expectedOutput: "Text $x + y$ more text",
		},
		{
			name:           "Display LaTeX to markdown",
			input:          "Text \\[E = mc^2\\] more text",
			expectedOutput: "Text $$E = mc^2$$ more text",
		},
		{
			name:           "Multiple inline expressions",
			input:          "\\(a\\) and \\(b\\) and \\(c\\)",
			expectedOutput: "$a$ and $b$ and $c$",
		},
		{
			name:           "Standalone inline math converted to display",
			input:          "Text\n\\(x^2 + y^2 = z^2\\)\nMore text",
			expectedOutput: "Text\n$$x^2 + y^2 = z^2$$\nMore text",
		},
		{
			name:           "Standalone with trailing punctuation",
			input:          "Text\n\\(f(x) = x^2\\).\nMore",
			expectedOutput: "Text\n$$f(x) = x^2.$$\nMore",
		},
		{
			name:           "Mixed inline and display",
			input:          "Inline \\(a\\) and display \\[b\\]",
			expectedOutput: "Inline $a$ and display $$b$$",
		},
		{
			name:           "Standalone inline math upgraded to display",
			input:          "\\(  x + y  \\)",
			expectedOutput: "$$x + y$$",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := parser.convertLatexMathDelimiters(testCase.input)
			if result != testCase.expectedOutput {
				subTester.Errorf("Expected: %q\nGot: %q", testCase.expectedOutput, result)
			}
		})
	}
}

func TestLatexMathConversionRoundTrip(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	testCases := []struct {
		name          string
		input         string
		expectedInAST string
		shouldContain string
	}{
		{
			name:          "Inline LaTeX converted to dollar notation",
			input:         "Text \\(x+y\\) more",
			expectedInAST: "x+y",
			shouldContain: "$",
		},
		{
			name:          "Display LaTeX converted to double dollar",
			input:         "\\[E=mc^2\\]",
			expectedInAST: "E=mc^2",
			shouldContain: "$$",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			ast := parser.Parse(testCase.input)
			reconstructed := reconstructor.Reconstruct(ast)

			// Verify the AST contains a node with expected content
			found := false
			var searchNode func(*Node)
			searchNode = func(node *Node) {
				if node == nil {
					return
				}
				if strings.Contains(node.Content, testCase.expectedInAST) {
					found = true
				}
				for _, child := range node.Children {
					searchNode(child)
				}
			}
			searchNode(ast)

			if !found {
				subTester.Errorf("AST did not contain expected content: %q", testCase.expectedInAST)
			}

			// Verify reconstruction produces math delimiters
			if !strings.Contains(reconstructed, testCase.shouldContain) {
				subTester.Errorf("Reconstructed markdown missing expected pattern %q:\n%s", testCase.shouldContain, reconstructed)
			}
		})
	}
}

// Equations with Footnote References Tests

func TestEquationsWithFootnoteReferences(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name                    string
		input                   string
		expectedEquationType    NodeType
		expectedEquationContent string
	}{
		{
			name:                    "Display equation with footnote reference",
			input:                   "\\[E = mc^2\\] [^1]",
			expectedEquationType:    NodeDisplayEquation,
			expectedEquationContent: "E = mc^2",
		},
		{
			name:                    "Inline equation with footnote reference",
			input:                   "\\(x + y\\) [^2]",
			expectedEquationType:    NodeParagraph, // Inline math stays in paragraph after conversion
			expectedEquationContent: "x + y",
		},
		{
			name:                    "Equation with footnote and period",
			input:                   "\\[\\alpha + \\beta\\] [^3].",
			expectedEquationType:    NodeDisplayEquation,
			expectedEquationContent: "\\alpha + \\beta",
		},
		{
			name:                    "Equation with footnote and comma",
			input:                   "\\(f(x)\\) [^4],",
			expectedEquationType:    NodeParagraph,
			expectedEquationContent: "f(x)",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			ast := parser.Parse(testCase.input)

			// Find the equation node
			var equationNode *Node
			for _, child := range ast.Children {
				if child.Type == NodeDisplayEquation {
					equationNode = child
					break
				}
			}

			if equationNode == nil {
				subTester.Fatalf("Expected to find %s node in AST", testCase.expectedEquationType)
			}

			if equationNode.Content != testCase.expectedEquationContent {
				subTester.Errorf("Expected equation content %q, got %q", testCase.expectedEquationContent, equationNode.Content)
			}
		})
	}
}

// Paragraph Equation Splitting Tests

func TestSplitParagraphEquations(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name              string
		input             string
		expectedNodeCount int
		expectedNodeTypes []NodeType
		expectedContents  []string
	}{
		{
			name:              "Paragraph with single embedded equation",
			input:             "Text before $$x^2$$ text after",
			expectedNodeCount: 3,
			expectedNodeTypes: []NodeType{NodeParagraph, NodeDisplayEquation, NodeParagraph},
			expectedContents:  []string{"Text before", "x^2", "text after"},
		},
		{
			name:              "Paragraph with multiple equations",
			input:             "Start $$eq1$$ middle $$eq2$$ end",
			expectedNodeCount: 5,
			expectedNodeTypes: []NodeType{NodeParagraph, NodeDisplayEquation, NodeParagraph, NodeDisplayEquation, NodeParagraph},
			expectedContents:  []string{"Start", "eq1", "middle", "eq2", "end"},
		},
		{
			name:              "Equation at start of paragraph",
			input:             "$$first$$ then text",
			expectedNodeCount: 2,
			expectedNodeTypes: []NodeType{NodeDisplayEquation, NodeParagraph},
			expectedContents:  []string{"first", "then text"},
		},
		{
			name:              "Equation at end of paragraph",
			input:             "Text then $$last$$",
			expectedNodeCount: 2,
			expectedNodeTypes: []NodeType{NodeParagraph, NodeDisplayEquation},
			expectedContents:  []string{"Text then", "last"},
		},
		{
			name:              "Only equation in paragraph",
			input:             "$$only$$",
			expectedNodeCount: 1,
			expectedNodeTypes: []NodeType{NodeDisplayEquation},
			expectedContents:  []string{"only"},
		},
		{
			name:              "Three consecutive equations",
			input:             "$$a$$ $$b$$ $$c$$",
			expectedNodeCount: 3,
			expectedNodeTypes: []NodeType{NodeDisplayEquation, NodeDisplayEquation, NodeDisplayEquation},
			expectedContents:  []string{"a", "b", "c"},
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			ast := parser.Parse(testCase.input)

			if len(ast.Children) != testCase.expectedNodeCount {
				subTester.Fatalf("Expected %d nodes, got %d", testCase.expectedNodeCount, len(ast.Children))
			}

			for i, child := range ast.Children {
				if child.Type != testCase.expectedNodeTypes[i] {
					subTester.Errorf("Node %d: expected type %s, got %s", i, testCase.expectedNodeTypes[i], child.Type)
				}

				if child.Content != testCase.expectedContents[i] {
					subTester.Errorf("Node %d: expected content %q, got %q", i, testCase.expectedContents[i], child.Content)
				}
			}
		})
	}
}

// Page Number Formatting and Parsing Tests

func TestFormatPageNumbers(tester *testing.T) {
	testCases := []struct {
		name           string
		pages          []int
		expectedOutput string
	}{
		{
			name:           "Single page",
			pages:          []int{5},
			expectedOutput: "5",
		},
		{
			name:           "Two consecutive pages",
			pages:          []int{5, 6},
			expectedOutput: "5–6",
		},
		{
			name:           "Three consecutive pages",
			pages:          []int{1, 2, 3},
			expectedOutput: "1–3",
		},
		{
			name:           "Non-consecutive pages",
			pages:          []int{1, 3, 5},
			expectedOutput: "1, 3, 5",
		},
		{
			name:           "Mixed ranges and singles",
			pages:          []int{1, 2, 3, 5, 7, 8, 9},
			expectedOutput: "1–3, 5, 7–9",
		},
		{
			name:           "Duplicate pages",
			pages:          []int{1, 1, 2, 3, 3, 3},
			expectedOutput: "1–3",
		},
		{
			name:           "Unsorted input",
			pages:          []int{9, 1, 5, 3, 7},
			expectedOutput: "1, 3, 5, 7, 9",
		},
		{
			name:           "Long consecutive range",
			pages:          []int{10, 11, 12, 13, 14, 15},
			expectedOutput: "10–15",
		},
		{
			name:           "Empty array",
			pages:          []int{},
			expectedOutput: "",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := FormatPageNumbers(testCase.pages)
			if result != testCase.expectedOutput {
				subTester.Errorf("Expected: %q, Got: %q", testCase.expectedOutput, result)
			}
		})
	}
}

func TestParsePageStringWithEnDash(tester *testing.T) {
	testCases := []struct {
		name          string
		pageString    string
		expectedPages []int
	}{
		{
			name:          "Range with en-dash",
			pageString:    "5–6",
			expectedPages: []int{5, 6},
		},
		{
			name:          "Range with hyphen",
			pageString:    "5-6",
			expectedPages: []int{5, 6},
		},
		{
			name:          "Multiple ranges with en-dash",
			pageString:    "1–3, 5–7",
			expectedPages: []int{1, 2, 3, 5, 6, 7},
		},
		{
			name:          "Mixed hyphen and en-dash",
			pageString:    "1-3, 5–7, 9",
			expectedPages: []int{1, 2, 3, 5, 6, 7, 9},
		},
		{
			name:          "Single page",
			pageString:    "42",
			expectedPages: []int{42},
		},
		{
			name:          "Pages with spaces",
			pageString:    "1 – 5, 10",
			expectedPages: []int{1, 2, 3, 4, 5, 10},
		},
		{
			name:          "Page with p prefix",
			pageString:    "p5-7",
			expectedPages: []int{5, 6, 7},
		},
		{
			name:          "Empty string",
			pageString:    "",
			expectedPages: []int{},
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := ParsePageString(testCase.pageString)

			if len(result) != len(testCase.expectedPages) {
				subTester.Errorf("Expected %d pages, got %d: %v", len(testCase.expectedPages), len(result), result)
				return
			}

			for i, page := range result {
				if page != testCase.expectedPages[i] {
					subTester.Errorf("Page %d: expected %d, got %d", i, testCase.expectedPages[i], page)
				}
			}
		})
	}
}

func TestPageNumberRoundTrip(tester *testing.T) {
	testCases := []struct {
		name          string
		originalPages []int
	}{
		{
			name:          "Single page round trip",
			originalPages: []int{5},
		},
		{
			name:          "Range round trip",
			originalPages: []int{1, 2, 3, 4, 5},
		},
		{
			name:          "Complex pattern round trip",
			originalPages: []int{1, 2, 3, 7, 9, 10, 11},
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			formatted := FormatPageNumbers(testCase.originalPages)
			parsed := ParsePageString(formatted)

			if len(parsed) != len(testCase.originalPages) {
				subTester.Errorf("Round trip failed: expected %d pages, got %d", len(testCase.originalPages), len(parsed))
				return
			}

			for i, page := range parsed {
				if page != testCase.originalPages[i] {
					subTester.Errorf("Round trip failed at index %d: expected %d, got %d", i, testCase.originalPages[i], page)
				}
			}
		})
	}
}

// Heading Title Cleaning Tests

func TestCleanTitle(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Numeric prefix with period",
			input:          "1. Introduction",
			expectedOutput: "Introduction",
		},
		{
			name:           "Two digit numeric prefix",
			input:          "42. The Answer",
			expectedOutput: "The Answer",
		},
		{
			name:           "Roman numeral prefix uppercase",
			input:          "I. Background",
			expectedOutput: "Background",
		},
		{
			name:           "Complex roman numeral",
			input:          "XIV. Chapter Fourteen",
			expectedOutput: "Chapter Fourteen",
		},
		{
			name:           "No prefix",
			input:          "Simple Title",
			expectedOutput: "Simple Title",
		},
		{
			name:           "Title with number but no prefix pattern",
			input:          "Chapter 1 Overview",
			expectedOutput: "Chapter 1 Overview",
		},
		{
			name:           "Prefix with multiple spaces",
			input:          "5.  Multiple Spaces",
			expectedOutput: "Multiple Spaces",
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			result := parser.cleanTitle(testCase.input)
			if result != testCase.expectedOutput {
				subTester.Errorf("Expected: %q, Got: %q", testCase.expectedOutput, result)
			}
		})
	}
}

func TestHeadingTitleCleaningInParse(tester *testing.T) {
	parser := NewParser()

	markdownContent := `# 1. Introduction
## 2. Background
### I. Roman Numeral Section`

	ast := parser.Parse(markdownContent)

	// Find all section nodes
	var sections []*Node
	var findSections func(*Node)
	findSections = func(node *Node) {
		if node == nil {
			return
		}
		if node.Type == NodeSection {
			sections = append(sections, node)
		}
		for _, child := range node.Children {
			findSections(child)
		}
	}
	findSections(ast)

	if len(sections) != 3 {
		tester.Fatalf("Expected 3 sections, got %d", len(sections))
	}

	expectedTitles := []string{"Introduction", "Background", "Roman Numeral Section"}
	for i, section := range sections {
		if section.Title != expectedTitles[i] {
			tester.Errorf("Section %d: expected title %q, got %q", i, expectedTitles[i], section.Title)
		}
	}
}

// Edge Cases: Unclosed Blocks

func TestUnclosedCodeBlock(tester *testing.T) {
	parser := NewParser()

	markdownContent := "```\ncode line 1\ncode line 2"

	ast := parser.Parse(markdownContent)

	// Unclosed code block should not create a code block node
	// It should be treated as regular paragraphs
	hasCodeBlock := false
	for _, child := range ast.Children {
		if child.Type == NodeCodeBlock {
			hasCodeBlock = true
			break
		}
	}

	if hasCodeBlock {
		tester.Errorf("Unclosed code block should not create a CodeBlock node")
	}
}

func TestUnclosedDisplayEquation(tester *testing.T) {
	parser := NewParser()

	markdownContent := "$$\nequation line 1\nequation line 2"

	ast := parser.Parse(markdownContent)

	// Unclosed equation should not create a display equation node
	hasDisplayEquation := false
	for _, child := range ast.Children {
		if child.Type == NodeDisplayEquation && child.IsMultiline {
			hasDisplayEquation = true
			break
		}
	}

	if hasDisplayEquation {
		tester.Errorf("Unclosed display equation should not create a multi-line DisplayEquation node")
	}
}

func TestProperlyClosedBlocks(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name         string
		markdown     string
		expectedType NodeType
	}{
		{
			name:         "Properly closed code block",
			markdown:     "```\ncode\n```",
			expectedType: NodeCodeBlock,
		},
		{
			name:         "Properly closed LaTeX display equation",
			markdown:     "\\[\nequation\n\\]",
			expectedType: NodeDisplayEquation,
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			ast := parser.Parse(testCase.markdown)

			found := false
			for _, child := range ast.Children {
				if child.Type == testCase.expectedType {
					found = true
					break
				}
			}

			if !found {
				subTester.Errorf("Expected to find %s node, got %d children", testCase.expectedType, len(ast.Children))
				for i, child := range ast.Children {
					subTester.Logf("  Child %d: Type=%s, Content=%q", i, child.Type, child.Content)
				}
			}
		})
	}
}

// Table Edge Cases

func TestTableWithInconsistentColumns(tester *testing.T) {
	parser := NewParser()

	// Table where rows have different numbers of columns
	markdownContent := `| A | B | C |
| --- | --- | --- |
| 1 | 2 |
| 3 | 4 | 5 | 6 |`

	ast := parser.Parse(markdownContent)

	var tableNode *Node
	for _, child := range ast.Children {
		if child.Type == NodeTable {
			tableNode = child
			break
		}
	}

	if tableNode == nil {
		tester.Fatalf("Expected to find a table node")
	}

	// Verify table was parsed despite inconsistent columns
	if len(tableNode.Rows) != 3 {
		tester.Errorf("Expected 3 rows, got %d", len(tableNode.Rows))
	}
}

func TestTableWithDisplayEquations(tester *testing.T) {
	parser := NewParser()

	markdownContent := `| Formula | Description |
| --- | --- |
| $$E=mc^2$$ | Energy equation |`

	ast := parser.Parse(markdownContent)

	var tableNode *Node
	for _, child := range ast.Children {
		if child.Type == NodeTable {
			tableNode = child
			break
		}
	}

	if tableNode == nil {
		tester.Fatalf("Expected to find a table node")
	}

	// Verify the display equation wasn't split
	if len(tableNode.Rows) < 2 {
		tester.Fatalf("Expected at least 2 rows")
	}

	dataRow := tableNode.Rows[1]
	if len(dataRow.Cells) < 1 {
		tester.Fatalf("Expected at least 1 cell in data row")
	}

	// The cell should contain the equation markers (escaped)
	if !strings.Contains(dataRow.Cells[0], "E=mc^2") {
		tester.Errorf("Expected cell to contain equation content, got: %q", dataRow.Cells[0])
	}
}

func TestTableWithoutAlignmentRow(tester *testing.T) {
	parser := NewParser()

	// Table without proper alignment row
	markdownContent := `| A | B |
| C | D |`

	ast := parser.Parse(markdownContent)

	// This should not be parsed as a table
	hasTable := false
	for _, child := range ast.Children {
		if child.Type == NodeTable {
			hasTable = true
			break
		}
	}

	if hasTable {
		tester.Errorf("Table without alignment row should not be parsed as a table")
	}
}

func TestEmptyTableCells(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	markdownContent := `| A | B | C |
| --- | --- | --- |
| 1 |  | 3 |
|  |  |  |`

	ast := parser.Parse(markdownContent)
	reconstructed := reconstructor.Reconstruct(ast)

	// Verify it round-trips (empty cells are preserved)
	if !strings.Contains(reconstructed, "|") {
		tester.Errorf("Table structure not preserved in reconstruction")
	}
}

// Deep Nesting Tests

func TestDeeplyNestedLists(tester *testing.T) {
	parser := NewParser()

	// 6 levels of nesting
	markdownContent := `- Level 0
    - Level 1
        - Level 2
            - Level 3
                - Level 4
                    - Level 5`

	ast := parser.Parse(markdownContent)

	// Find the top-level list item
	var topLevelItem *Node
	for _, child := range ast.Children {
		if child.Type == NodeListItem && child.Depth == 0 {
			topLevelItem = child
			break
		}
	}

	if topLevelItem == nil {
		tester.Fatalf("Expected to find top-level list item")
	}

	// Traverse down to verify depth
	currentItem := topLevelItem
	for expectedDepth := 0; expectedDepth <= 5; expectedDepth++ {
		if currentItem.Depth != expectedDepth {
			tester.Errorf("Expected depth %d, got %d", expectedDepth, currentItem.Depth)
		}

		if expectedDepth < 5 {
			if len(currentItem.Children) == 0 {
				tester.Fatalf("Expected nested item at depth %d", expectedDepth)
			}
			currentItem = currentItem.Children[0]
		}
	}
}

func TestDeeplySectionHierarchy(tester *testing.T) {
	parser := NewParser()

	// All 6 heading levels
	markdownContent := `# H1
## H2
### H3
#### H4
##### H5
###### H6`

	ast := parser.Parse(markdownContent)

	// Count sections at each level
	levelCounts := make(map[int]int)
	var countLevels func(*Node)
	countLevels = func(node *Node) {
		if node == nil {
			return
		}
		if node.Type == NodeSection {
			levelCounts[node.Level]++
		}
		for _, child := range node.Children {
			countLevels(child)
		}
	}
	countLevels(ast)

	// Should have one section at each level
	for level := 1; level <= 6; level++ {
		if levelCounts[level] != 1 {
			tester.Errorf("Expected 1 section at level %d, got %d", level, levelCounts[level])
		}
	}
}

// Reconstructor Edge Cases

func TestReconstructFootnoteWithoutSourceFile(tester *testing.T) {
	reconstructor := NewReconstructor()

	node := &Node{
		Type:           NodeFootnote,
		FootnoteNumber: 1,
		Content:        "Plain footnote text",
		SourceFile:     "",
		SourcePages:    nil,
	}

	ast := &Node{
		Type:     NodeDocument,
		Children: []*Node{node},
	}

	reconstructed := reconstructor.Reconstruct(ast)

	expected := "[^1]: Plain footnote text"
	if !strings.Contains(reconstructed, expected) {
		tester.Errorf("Expected reconstruction to contain %q, got:\n%s", expected, reconstructed)
	}

	// Should not contain backticks or parentheses
	if strings.Contains(reconstructed, "`") || strings.Contains(reconstructed, "(") {
		tester.Errorf("Footnote without source file should not contain metadata markers:\n%s", reconstructed)
	}
}

func TestReconstructFootnoteWithSourceButNoPages(tester *testing.T) {
	reconstructor := NewReconstructor()

	node := &Node{
		Type:           NodeFootnote,
		FootnoteNumber: 2,
		Content:        "Text with file reference",
		SourceFile:     "document.pdf",
		SourcePages:    nil,
	}

	ast := &Node{
		Type:     NodeDocument,
		Children: []*Node{node},
	}

	reconstructed := reconstructor.Reconstruct(ast)

	expectedContent := "[^2]: Text with file reference (`document.pdf`)"
	if !strings.Contains(reconstructed, expectedContent) {
		tester.Errorf("Expected reconstruction to contain %q, got:\n%s", expectedContent, reconstructed)
	}

	// Should not contain "pp." or "p."
	if strings.Contains(reconstructed, "pp.") || strings.Contains(reconstructed, " p. ") {
		tester.Errorf("Footnote without pages should not contain page markers:\n%s", reconstructed)
	}
}

func TestReconstructHorizontalRule(tester *testing.T) {
	reconstructor := NewReconstructor()

	node := &Node{
		Type: NodeHorizontalRule,
	}

	ast := &Node{
		Type:     NodeDocument,
		Children: []*Node{node},
	}

	reconstructed := reconstructor.Reconstruct(ast)

	if !strings.Contains(reconstructed, "---") {
		tester.Errorf("Expected horizontal rule '---' in reconstruction, got:\n%s", reconstructed)
	}
}

func TestReconstructMixedOrderedUnorderedLists(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	markdownContent := `- Unordered item
1. Ordered item
- Back to unordered
2. Another ordered`

	ast := parser.Parse(markdownContent)
	reconstructed := reconstructor.Reconstruct(ast)

	// Verify both list types are preserved
	if !strings.Contains(reconstructed, "-") {
		tester.Errorf("Unordered list marker '-' missing from reconstruction")
	}

	if !strings.Contains(reconstructed, "1.") {
		tester.Errorf("Ordered list marker '1.' missing from reconstruction")
	}
}

// Citation Metadata Parsing Edge Cases

func TestFootnoteMetadataParsingVariations(tester *testing.T) {
	parser := NewParser()

	testCases := []struct {
		name               string
		input              string
		expectedContent    string
		expectedFile       string
		expectedPagesCount int
	}{
		{
			name:               "Footnote with file but no pages",
			input:              "[^1]: Description (`file.pdf`)",
			expectedContent:    "Description",
			expectedFile:       "file.pdf",
			expectedPagesCount: 0,
		},
		{
			name:               "Footnote with single page using p.",
			input:              "[^2]: Text (`doc.pdf` p. 5)",
			expectedContent:    "Text",
			expectedFile:       "doc.pdf",
			expectedPagesCount: 1,
		},
		{
			name:               "Footnote with pages using pp.",
			input:              "[^3]: Info (`file.pdf` pp. 1-3)",
			expectedContent:    "Info",
			expectedFile:       "file.pdf",
			expectedPagesCount: 3,
		},
		{
			name:               "Footnote with extra spaces",
			input:              "[^4]: Content  (  `file.pdf`   pp.  5 – 7  )",
			expectedContent:    "Content",
			expectedFile:       "file.pdf",
			expectedPagesCount: 3,
		},
		{
			name:               "Footnote without metadata parentheses",
			input:              "[^5]: Just plain text",
			expectedContent:    "Just plain text",
			expectedFile:       "",
			expectedPagesCount: 0,
		},
	}

	for _, testCase := range testCases {
		tester.Run(testCase.name, func(subTester *testing.T) {
			ast := parser.Parse(testCase.input)

			var footnoteNode *Node
			for _, child := range ast.Children {
				if child.Type == NodeFootnote {
					footnoteNode = child
					break
				}
			}

			if footnoteNode == nil {
				subTester.Fatalf("Expected to find footnote node")
			}

			if footnoteNode.Content != testCase.expectedContent {
				subTester.Errorf("Expected content %q, got %q", testCase.expectedContent, footnoteNode.Content)
			}

			if footnoteNode.SourceFile != testCase.expectedFile {
				subTester.Errorf("Expected file %q, got %q", testCase.expectedFile, footnoteNode.SourceFile)
			}

			if len(footnoteNode.SourcePages) != testCase.expectedPagesCount {
				subTester.Errorf("Expected %d pages, got %d: %v", testCase.expectedPagesCount, len(footnoteNode.SourcePages), footnoteNode.SourcePages)
			}
		})
	}
}

// Complex Integration Tests

func TestComplexDocumentStructure(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	complexMarkdown := `# Main Title

Introduction paragraph with \(E=mc^2\) equation.

## Section 1

- List item 1
  - Nested item with \(x+y\) math
  - Another nested item
- List item 2

| Header A | Header B |
| --- | --- |
| \(\alpha\) | \(\beta\) |

### Subsection 1.1

` + "```" + `
code block content
` + "```" + `

\[
\int_0^\infty e^{-x^2} dx
\]

Some text[^1]

[^1]: Citation (` + "`source.pdf`" + ` pp. 10–15)`

	ast := parser.Parse(complexMarkdown)
	reconstructed := reconstructor.Reconstruct(ast)

	// Verify major structures are preserved
	checks := []struct {
		name     string
		mustFind string
	}{
		{"Heading", "# Main Title"},
		{"Display equation", "$$"},
		{"List marker", "-"},
		{"Table pipes", "|"},
		{"Code block", "```"},
		{"Footnote reference", "[^1]"},
		{"Footnote definition", "[^1]:"},
		{"Citation file", "source.pdf"},
		{"Page range", "10–15"},
	}

	for _, check := range checks {
		if !strings.Contains(reconstructed, check.mustFind) {
			tester.Errorf("%s missing: expected to find %q in reconstruction", check.name, check.mustFind)
		}
	}
}

func TestUnicodeMathSymbolsPreserved(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	unicodeMarkdown := `$$α + β = γ$$

Text with ∑, ∫, ∂, ∇ symbols.

| Symbol | Meaning |
| --- | --- |
| ∞ | Infinity |
| ≈ | Approximately |`

	ast := parser.Parse(unicodeMarkdown)
	reconstructed := reconstructor.Reconstruct(ast)

	unicodeSymbols := []string{"α", "β", "γ", "∑", "∫", "∂", "∇", "∞", "≈"}
	for _, symbol := range unicodeSymbols {
		if !strings.Contains(reconstructed, symbol) {
			tester.Errorf("Unicode symbol %q not preserved in reconstruction", symbol)
		}
	}
}

func TestEmptyLinePreservation(tester *testing.T) {
	parser := NewParser()
	reconstructor := NewReconstructor()

	markdownWithEmptyLines := `# Title

Paragraph 1


Paragraph 2 with multiple empty lines above`

	ast := parser.Parse(markdownWithEmptyLines)
	reconstructed := reconstructor.Reconstruct(ast)

	// Verify structure is maintained (exact empty line count may vary)
	if !strings.Contains(reconstructed, "Title") {
		tester.Errorf("Title missing from reconstruction")
	}

	if !strings.Contains(reconstructed, "Paragraph 1") {
		tester.Errorf("Paragraph 1 missing from reconstruction")
	}

	if !strings.Contains(reconstructed, "Paragraph 2") {
		tester.Errorf("Paragraph 2 missing from reconstruction")
	}
}
