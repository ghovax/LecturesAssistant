{{language_requirement}}

Your task is to generate a set of comprehensive flashcards based on the provided lecture transcript and reference materials. These flashcards should cover all key concepts, definitions, formulas, and important facts discussed in the lecture.

**Critical Instructions:**

- Each flashcard must consist of a "Front" (question or concept) and a "Back" (answer or explanation).
- Use high-fidelity information from the transcript as the primary source.
- Reference materials should be used for accurate terminology and verification.
- The flashcards must be pedagogically sound, helping students test their knowledge.
- Do not repeat the same question across multiple flashcards.
- Formatting: Use Markdown format.

{{latex_instructions}}

---

# Input Content

{{transcript}}

{{reference_materials}}

---

**Output Format:**

Output the flashcards as a JSON array of objects, each containing "front" and "back" fields.

Example:

```json
[
  {
    "front": "What is the powerhouse of the cell?",
    "back": "The mitochondria."
  },
  {
    "front": "Write the equation for Newton's Second Law.",
    "back": "\(F = ma\)"
  }
]
```

Return **only** the JSON array, with no additional text or formatting outside the JSON.
