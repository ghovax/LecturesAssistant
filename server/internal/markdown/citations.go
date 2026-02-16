package markdown

import (
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ParsedCitation represents metadata extracted from a {{{...}}} marker
type ParsedCitation struct {
	Number      int
	Description string
	File        string
	Pages       []int
}

func (reconstructor *Reconstructor) ParseCitations(text string) (string, []ParsedCitation) {
	// Refined regex:
	// 1. {{{
	// 2. (.*) - non-greedy match for everything until the last dash before either a -p or the closing }}}
	// 3. (?:-(p[\d\s\-,]+))? - optional page marker starting with -p
	// 4. \s*}}}
	// Actually, a simpler way to handle "description-file-p1" where description has dashes
	// is to match the whole content and then split it from the end.

	// Match leading whitespace + the marker
	citationRegex := regexp.MustCompile(`\s*\{\{\{(.*?)\s*\}\}\}`)
	matches := citationRegex.FindAllStringSubmatch(text, -1)

	slog.Debug("ParseCitations called", "text_length", len(text), "matches_found", len(matches))
	if len(matches) > 0 {
		slog.Debug("First citation match", "content", matches[0][1])
	}

	var citations []ParsedCitation
	result := text

	for citationIndex, match := range matches {
		fullMatch := match[0]
		content := strings.TrimSpace(match[1])

		// Split logic:
		// We expect [description]-[file]-p[pages] or [description]-[file]
		// Let's find if there is a -p[digits] at the end

		var description, filename, pageString string

		pageRegex := regexp.MustCompile(`-p([\d\s\-,]+)$`)
		pageMatch := pageRegex.FindStringSubmatch(content)

		remaining := content
		if pageMatch != nil {
			pageString = pageMatch[1]
			remaining = content[:len(content)-len(pageMatch[0])]
		}

		// Now remaining should be [description]-[filename]
		// Since we normalized filenames to replace dashes with underscores, the last dash
		// must be the separator between description and filename.
		// However, to be even more robust, we look for the file extension.
		extensionRegex := regexp.MustCompile(`^(.*)-([^\-]+?\.[a-z0-9]+)$`)
		extMatch := extensionRegex.FindStringSubmatch(remaining)

		if extMatch != nil {
			description = strings.TrimSpace(extMatch[1])
			filename = strings.TrimSpace(extMatch[2])
		} else {
			// Fallback to last dash split if regex fails
			lastDash := strings.LastIndex(remaining, "-")
			if lastDash != -1 {
				description = strings.TrimSpace(remaining[:lastDash])
				filename = strings.TrimSpace(remaining[lastDash+1:])
			} else {
				description = remaining
				filename = "unknown"
			}
		}

		citationNumber := citationIndex + 1
		pages := ParsePageString(pageString)

		citations = append(citations, ParsedCitation{
			Number:      citationNumber,
			Description: description,
			File:        filename,
			Pages:       pages,
		})

		// Replace marker (including its leading whitespace) with [^N]
		result = strings.Replace(result, fullMatch, fmt.Sprintf("[^%d]", citationNumber), 1)
	}
	// Move periods/commas before footnote references to before and consolidate duplicates
	// 1. Move all surrounding punctuation to the left and strip whitespace
	result = regexp.MustCompile(`[ \t]*([.,]*)[ \t]*(\[\^\d+\])[ \t]*([.,]*)`).ReplaceAllString(result, "$1$3$2")

	// 2. Remove punctuation/whitespace between consecutive citations: "[^1]. [^2]" -> ".[^1][^2]"
	// Wait, if it's ".[^1][^2]", we need to make sure we don't end up with ".[^1].[^2]"
	result = regexp.MustCompile(`(\[\^\d+\])[ \t.,]*(\[\^\d+\])`).ReplaceAllString(result, "$1$2")

	// 3. Consolidate multiple dots/commas into one (now at the left of the first reference)
	result = regexp.MustCompile(`([.,]{2,})(\[\^\d+\])`).ReplaceAllStringFunc(result, func(match string) string {
		sub := regexp.MustCompile(`([.,]{2,})(\[\^\d+\])`).FindStringSubmatch(match)
		return sub[1][:1] + sub[2]
	})

	return result, citations
}

// ParsePageString converts "1, 2, 5-10" to []int{1, 2, 5, 6, 7, 8, 9, 10}
func ParsePageString(pageString string) []int {
	var pages []int
	if pageString == "" {
		return pages
	}

	parts := strings.Split(pageString, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "p")

		// Handle both hyphen-minus (-) and en-dash (–)
		if strings.Contains(part, "-") || strings.Contains(part, "–") {
			// Replace en-dash with hyphen for consistent splitting
			part = strings.ReplaceAll(part, "–", "-")
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, _ := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, _ := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if start > 0 && end >= start {
					for pageIndex := start; pageIndex <= end; pageIndex++ {
						pages = append(pages, pageIndex)
					}
				}
			}
		} else {
			pageNum, _ := strconv.Atoi(part)
			if pageNum > 0 {
				pages = append(pages, pageNum)
			}
		}
	}
	return pages
}

// FormatPageNumbers converts []int{1, 2, 3, 5} to "1–3, 5"
func FormatPageNumbers(pages []int) string {
	if len(pages) == 0 {
		return ""
	}

	// Deduplicate and sort
	uniqueMap := make(map[int]bool)
	for _, pageNumber := range pages {
		uniqueMap[pageNumber] = true
	}
	var sortedPages []int
	for pageNumber := range uniqueMap {
		sortedPages = append(sortedPages, pageNumber)
	}
	sort.Ints(sortedPages)

	var ranges []string
	if len(sortedPages) == 0 {
		return ""
	}

	rangeStart := sortedPages[0]
	rangeEnd := sortedPages[0]

	for pageIndex := 1; pageIndex < len(sortedPages); pageIndex++ {
		if sortedPages[pageIndex] == rangeEnd+1 {
			rangeEnd = sortedPages[pageIndex]
		} else {
			if rangeStart == rangeEnd {
				ranges = append(ranges, strconv.Itoa(rangeStart))
			} else {
				ranges = append(ranges, fmt.Sprintf("%d–%d", rangeStart, rangeEnd))
			}
			rangeStart = sortedPages[pageIndex]
			rangeEnd = sortedPages[pageIndex]
		}
	}

	if rangeStart == rangeEnd {
		ranges = append(ranges, strconv.Itoa(rangeStart))
	} else {
		ranges = append(ranges, fmt.Sprintf("%d–%d", rangeStart, rangeEnd))
	}

	return strings.Join(ranges, ", ")
}
