{{language_requirement}}

Your task is to generate a comprehensive multiple-choice quiz based on the provided lecture transcript and reference materials. The quiz should test the student's understanding of all major topics and details discussed.

**Critical Instructions:**

- Each question must have exactly 4 options (A, B, C, D).
- There must be exactly one correct answer for each question.
- Provide a clear, pedagogical explanation for the correct answer.
- The questions should vary in difficulty and cover the entire lecture content.
- Use high-fidelity information from the transcript as the primary source.
- Reference materials should be used for accurate terminology and verification.

{{latex_instructions}}

---

# Input Content

{{transcript}}

{{reference_materials}}

---

**Output Format:**

Output the quiz as a JSON array of objects, each containing "question", "options" (array of 4 strings), "correct_answer" (the exact string of the correct option), and "explanation".

Example:
```json
[
  {
    "question": "Which organelle is responsible for ATP production?",
    "options": ["Nucleus", "Ribosome", "Mitochondria", "Golgi apparatus"],
    "correct_answer": "Mitochondria",
    "explanation": "Mitochondria are known as the powerhouse of the cell because they generate most of the cell's supply of adenosine triphosphate (ATP)."
  }
]
```

Return **only** the JSON array, with no additional text or formatting outside the JSON.
