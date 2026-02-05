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
			expectedMarkdown: "Referencing [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "Description with dashes",
			input:            "Referencing {{{A-complex-desc-file.pdf-p1}}}",
			expectedMarkdown: "Referencing [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "A-complex-desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "File with dots and no pages",
			input:            "Referencing {{{Desc-my.file.name.pdf}}}",
			expectedMarkdown: "Referencing [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "my.file.name.pdf", Pages: nil},
			},
		},
		{
			name:             "Multiple citations",
			input:            "First {{{D1-f1.pdf-p1}}} and second {{{D2-f2.pdf-p2}}}",
			expectedMarkdown: "First [^1] and second [^2]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "D1", File: "f1.pdf", Pages: []int{1}},
				{Number: 2, Description: "D2", File: "f2.pdf", Pages: []int{2}},
			},
		},
		{
			name:             "Page range",
			input:            "Range {{{Desc-file.pdf-p1-3}}}",
			expectedMarkdown: "Range [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1, 2, 3}},
			},
		},
		{
			name:             "Complex pages",
			input:            "Complex {{{Desc-file.pdf-p1, 3, 5-7}}}",
			expectedMarkdown: "Complex [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1, 3, 5, 6, 7}},
			},
		},
		{
			name:             "Messy spacing",
			input:            "Messy {{{  Desc  -  file.pdf  -p1  }}}",
			expectedMarkdown: "Messy [^1]",
			expectedCitations: []ParsedCitation{
				{Number: 1, Description: "Desc", File: "file.pdf", Pages: []int{1}},
			},
		},
		{
			name:             "Filename with underscore",
			input:            "Referencing {{{Description-file_name.pdf-p1}}}",
			expectedMarkdown: "Referencing [^1]",
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
