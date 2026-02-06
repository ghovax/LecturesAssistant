Your task is to generate exactly 3 thoughtful questions about the provided document that would help a reader better understand or explore the content. These questions will be suggested to users in a reading assistant interface. Each question must be a single sentence, brief, and kept under 50 words. The questions should be specific to the document's content, encourage deeper understanding, be natural and conversational in tone and approachable, and vary in focus such as one about main ideas, one about specific details, and one about implications. The questions must be generated in the language of the document itself to ensure that the reader can properly understand them. Your questions may contain inline LaTeX equations/formulas written between \(...\) if needed.

---

# Document Content

{{document_content}}

---

{{latex_instructions}}

---

**Output Format:**

Return only a valid JSON object with a "questions" field containing an array of exactly 3 strings, with no additional text, explanations, or formatting outside the JSON as follows:

{
  "questions": [
    "First question here?",
    "Second question here?",
    "Third question here?"
  ]
}