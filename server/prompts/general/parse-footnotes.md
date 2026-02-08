# Footnote Parsing Task

{{language_requirement}}

You are given a set of footnotes to parse. Extract the metadata from each footnote including the file reference, page numbers, and page section (if mentioned). Return a JSON object containing an array of footnotes with their metadata.

{{latex_instructions}}

### Critical Preservation Rules

**Language Preservation:** You must preserve the original language of the footnote text. Do not translate footnotes. If the footnote is in Italian, keep it in Italian. If it is in English, keep it in English. The same applies to all other languages.

**LaTeX Preservation:** You must preserve all LaTeX math formulas exactly as they appear; if something LaTeX is missing, try to fix it based on context. Do not strip, simplify, or convert LaTeX math. Ensure all backslashes are preserved. If a formula is broken or incomplete, try to fix it based on context, but prioritize preservation.

**Critical Requirement: Preserve Exact Footnote Numbers**

You **must** preserve the exact footnote number as it appears in the input. Do **not** renumber footnotes. Do **not** start from 1 if the first footnote is numbered differently. Do **not** skip any footnote numbers. If you receive footnotes numbered [^16] through [^30], your output must contain numbers 16 through 30, not 1 through 15.

## Fields to Extract

For each footnote, extract:

- `number`: The **exact** footnote number as it appears (e.g., if input is `[^25]:`, number must be 25)
- `text_content`: The full text content of the footnote
- `file`: The filename referenced (if any)
- `pages`: Array of page numbers mentioned (if any)

## Critical Requirements

- **Process every footnote in the input**: Do **not** skip or omit any
- **Preserve original numbering**: Return footnotes with their exact numbers from the input
- **Return _only_ valid JSON**: Your response must be pure JSON with no additional text, explanations, Markdown formatting, or code blocks. Do not wrap the JSON in `json` tags.
- Use empty arrays `[]` for pages when no page numbers are found
- Use empty strings `""` for file when no filename is mentioned
- If you cannot parse a footnote, include it anyway with empty metadata rather than omitting it

## Output Format

```json
{
  "footnotes": [
    {
      "number": 1,
      "text_content": "Full footnote text here",
      "pages": [1, 2, 3],
      "file": "source.pdf"
    }
  ]
}
```

---

# Footnotes to Parse

{{footnotes}}
