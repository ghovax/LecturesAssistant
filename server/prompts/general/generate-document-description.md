{{language_requirement}}

You are tasked with generating a **single sentence** description for a document based on its Markdown content. Read and analyze the document content provided below, then generate a brief description that summarizes the main topic or purpose. It needs to be a description, not a title.

**Critical Requirements:**

- **Must be exactly one sentence only** (maximum 25-30 words)
- Be direct and concise - no introductory phrases like "this document," "this study," or "this paper", or "the document explains", etc.
- Start directly with the topic or subject matter
- Use professional, clear language

## Document Content

{{document_content}}

---

{{latex_instructions}}

Your response must be formatted in the following manner:

<description>[Insert the single sentence description here.]</description>

This is not XML, so it doesn't need escaping, it's just a way for me to extract the description directly from your response. Write the description between the <description> tags in normal Markdown with embedded LaTeX for mathematical expressions where appropriate. Use standard Markdown syntax and LaTeX math delimiters (\(...\) for inline, \[...\] for display).

### Examples

For a document about enzyme kinetics: <description>Comprehensive guide to enzyme kinetics, covering competitive and non-competitive inhibition mechanisms.</description>

For a document about web development: <description>Introduction to modern web development using React, TypeScript, and component-based architecture principles.</description>

For a document about climate change: <description>Analysis of climate change impacts on global ecosystems and potential mitigation strategies.</description>
