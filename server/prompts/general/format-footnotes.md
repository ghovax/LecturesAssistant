{{language_requirement}}

The footnote must be fixed to be a standalone piece of content detailing what has been referenced, without any reference to files or pages. Each footnote must be a complete, grammatically correct sentence or phrase that stands alone as a complete thought, being formed into a normal sentence, with correct grammar, syntax and punctuation. It is absolutely critical that even the shortest or most fragmented input is transformed into a full, coherent, and polished sentence.

**Strict Negative Constraints (Extremely Critical):**

- **Do not** mention the source filename (e.g., "document.pdf").
- **Do not** mention page numbers or ranges (e.g., "on page 1", "S. 1", "pp. 5-10").
- **Do not** mention the title of the resource or document.
- **Do not** use introductory meta-phrases such as "This footnote references...", "Specifically from...", "According to the resource...", "The document states...", etc.
- **Do not** include any parenthetical citations at the end of the sentence.

Focus exclusively on the factual or conceptual information being cited. The goal is to produce a natural sentence that conveys the information as if it were part of the main text, but placed in a footnote for detail. Every footnote **must** be a perfect, high-quality sentence regardless of the input quality.

Footnotes must not be short fragments, lowercase phrases, or incomplete thoughts. If a footnote is a broken fragment or incomplete sentence, it needs to be made into a proper sentence. They must clearly state the cited information in a way that makes the reasoning behind the citation clear and unambiguous, while remaining concise, taking from the available context of the footnote itself, avoiding to insert information that wasn't originally present.

Footnotes should be maximum one or two sentences; only if a footnote exceeds two or three sentences, summarize it into one or two sentences. Respect the formatting. All footnotes must be accounted for in your response. You must not forget any individual footnote. Return just the footnotes, without any other message before it or after it. Preserve the LaTeX \(...\) terms, and fix it if malformed or include it if missing and necessary. The language of the footnotes and their conventions or styles must be preserved, taking the following example of a proper footnote just as a reference.

{{latex_instructions}}

### Critical Preservation Rules

**Language Preservation:** You must preserve the original language of the footnote text. Do not translate footnotes. If the footnote is in Italian, keep it in Italian. If it is in English, keep it in English. The same applies to all other languages.

**LaTeX Preservation:** You must preserve all LaTeX math formulas exactly as they appear; if something LaTeX is missing, try to fix it based on context. Do not strip, simplify, or convert LaTeX math. Ensure all backslashes are preserved. If a formula is broken or incomplete, try to fix it based on context, but prioritize preservation.

## Examples of Correct vs Incorrect Formatting

**Incorrect:** This footnote references a dialogue from a resource titled “Man answering: I’m from Washington, D.C..-Where Are You From Part 1 _ Dialogue (lingoneo.org),” specifically from page 1. (Where Are You From Part 1 _ Dialogue (lingoneo.org).pdf, S. 1)
**Correct:** The dialogue features a man stating that he is originally from Washington, D.C.

**Incorrect:** According to page 45 of biology_textbook.pdf, the mitochondria is the powerhouse of the cell.
**Correct:** The mitochondria is the powerhouse of the cell, responsible for generating most of the cell's supply of ATP.

## Example of Proper Output Footnotes

[^1]: This is an example footnote with a formula CO\(\_2\) and another one \(x^2\).

[^2]: This is another example footnote.

---

{{footnotes}}
