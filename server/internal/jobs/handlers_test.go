package jobs

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal filename with spaces",
			input:    "Introduction to Python",
			expected: "Introduction to Python",
		},
		{
			name:     "Filename with forward slash",
			input:    "Chapter 1/2",
			expected: "Chapter 1_2",
		},
		{
			name:     "Filename with backslash",
			input:    "Path\\to\\file",
			expected: "Path_to_file",
		},
		{
			name:     "Filename with colon",
			input:    "Chapter 1: Getting Started",
			expected: "Chapter 1_ Getting Started",
		},
		{
			name:     "Filename with asterisk",
			input:    "Quiz *Important*",
			expected: "Quiz _Important_",
		},
		{
			name:     "Filename with question mark",
			input:    "What is Python?",
			expected: "What is Python_",
		},
		{
			name:     "Filename with quotes",
			input:    `The "Best" Guide`,
			expected: "The _Best_ Guide",
		},
		{
			name:     "Filename with angle brackets",
			input:    "Guide <Advanced>",
			expected: "Guide _Advanced_",
		},
		{
			name:     "Filename with pipe",
			input:    "Option A | Option B",
			expected: "Option A _ Option B",
		},
		{
			name:     "Filename with multiple unsafe characters",
			input:    `Chapter 1: "Introduction" / Getting Started?`,
			expected: "Chapter 1_ _Introduction_ _ Getting Started_",
		},
		{
			name:     "Filename with leading and trailing spaces",
			input:    "  Study Guide  ",
			expected: "Study Guide",
		},
		{
			name:     "Filename with leading and trailing dots",
			input:    "..Important..",
			expected: "Important",
		},
		{
			name:     "Filename with newlines",
			input:    "Line 1\nLine 2",
			expected: "Line 1_Line 2",
		},
		{
			name:     "Filename with tabs",
			input:    "Column 1\tColumn 2",
			expected: "Column 1_Column 2",
		},
		{
			name:     "Empty filename",
			input:    "",
			expected: "document",
		},
		{
			name:     "Filename with only spaces",
			input:    "   ",
			expected: "document",
		},
		{
			name:     "Filename with only dots",
			input:    "...",
			expected: "document",
		},
		{
			name:     "Filename with only unsafe characters",
			input:    "/:*?\"<>|",
			expected: "document",
		},
		{
			name:     "Unicode characters preserved",
			input:    "Résumé in 日本語",
			expected: "Résumé in 日本語",
		},
		{
			name:     "Hyphens and underscores preserved",
			input:    "Study-Guide_2024",
			expected: "Study-Guide_2024",
		},
		{
			name:     "Numbers and periods preserved",
			input:    "Version 1.2.3",
			expected: "Version 1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_RealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Academic paper title",
			input:    "Introduction to Machine Learning: A Comprehensive Guide",
			expected: "Introduction to Machine Learning_ A Comprehensive Guide",
		},
		{
			name:     "Quiz with special characters",
			input:    "Quiz #2 - Midterm (Part 1/3)",
			expected: "Quiz _2 - Midterm (Part 1_3)",
		},
		{
			name:     "Study guide with date",
			input:    "Study Guide 2024-01-15",
			expected: "Study Guide 2024-01-15",
		},
		{
			name:     "Flashcards with emoji",
			input:    "Flashcards ✨ Review",
			expected: "Flashcards ✨ Review",
		},
		{
			name:     "Chapter with subsection",
			input:    "Chapter 3.2: Advanced Topics",
			expected: "Chapter 3.2_ Advanced Topics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q; expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
