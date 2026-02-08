package markdown

import (
	"fmt"
	"regexp"
	"sort"
)

// ImageResolver is a function that returns the local file path for a cited page
type ImageResolver func(filename string, pageNumber int) string

// EnrichWithCitedImages walks the AST, identifies cited pages in sections (Level 2 and 3),
// and appends NodeImage nodes to the end of each section where a page is first cited.
func EnrichWithCitedImages(root *Node, resolver ImageResolver) {
	if root == nil || resolver == nil {
		return
	}

	// 1. Map footnote numbers to their source info
	footnoteMap := make(map[int]struct {
		File  string
		Pages []int
	})

	var collectFootnotes func(*Node)
	collectFootnotes = func(node *Node) {
		if node.Type == NodeFootnote {
			footnoteMap[node.FootnoteNumber] = struct {
				File  string
				Pages []int
			}{
				File:  node.SourceFile,
				Pages: node.SourcePages,
			}
		}
		for _, child := range node.Children {
			collectFootnotes(child)
		}
	}
	collectFootnotes(root)

	// 2. Track which (file, page) pairs have been inserted
	insertedPages := make(map[string]bool) // Key: "filename:page"

	// Regex to find [^N] references
	refRegex := regexp.MustCompile(`\[\^(\d+)\]`)

	// 3. Process the document section by section
	var processSection func(*Node)
	processSection = func(node *Node) {
		// We look for any section (Level 2 or Level 3)
		if node.Type == NodeSection && (node.Level == 2 || node.Level == 3) {
			// Find all citations strictly in this section (not its subsections if it's level 2)
			citedInThisSection := make(map[string][]int) // File -> Pages

			var findRefs func(*Node)
			findRefs = func(n *Node) {
				if n.Type == NodeParagraph || n.Type == NodeListItem {
					matches := refRegex.FindAllStringSubmatch(n.Content, -1)
					for _, match := range matches {
						num := 0
						fmt.Sscanf(match[1], "%d", &num)
						if info, ok := footnoteMap[num]; ok && info.File != "" {
							for _, p := range info.Pages {
								citedInThisSection[info.File] = append(citedInThisSection[info.File], p)
							}
						}
					}
				}

				// Recurse into children ONLY if they are not sections themselves
				// This ensures citations in a Level 3 subsection are not attributed to the Level 2 parent
				for _, child := range n.Children {
					if child.Type != NodeSection {
						findRefs(child)
					}
				}
			}
			findRefs(node)

			// Collect pages to insert at the end of this section
			var imagesToInsert []*Node

			var filenames []string
			for f := range citedInThisSection {
				filenames = append(filenames, f)
			}
			sort.Strings(filenames)

			for _, f := range filenames {
				pages := citedInThisSection[f]
				uniquePages := make(map[int]bool)
				for _, p := range pages {
					uniquePages[p] = true
				}
				var sortedPages []int
				for p := range uniquePages {
					sortedPages = append(sortedPages, p)
				}
				sort.Ints(sortedPages)

				for _, p := range sortedPages {
					key := fmt.Sprintf("%s:%d", f, p)
					if !insertedPages[key] {
						imagePath := resolver(f, p)
						if imagePath != "" {
							imagesToInsert = append(imagesToInsert, &Node{
								Type:        NodeImage,
								Content:     imagePath,
								SourceFile:  f,
								SourcePages: []int{p},
							})
							insertedPages[key] = true
						}
					}
				}
			}

			// Append images to the section's children
			node.Children = append(node.Children, imagesToInsert...)

			// Continue processing sections within this one
			for _, child := range node.Children {
				if child.Type == NodeSection {
					processSection(child)
				}
			}
		} else {
			for _, child := range node.Children {
				processSection(child)
			}
		}
	}

	processSection(root)
}
