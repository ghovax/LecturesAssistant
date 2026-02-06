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

---

**Output Format:**

Return only a valid JSON object with a "description" field, with no additional text, explanations, or formatting outside the JSON as follows:

{"description": "Brief description here with embedded LaTeX for mathematical expressions where appropriate."}