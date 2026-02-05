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
