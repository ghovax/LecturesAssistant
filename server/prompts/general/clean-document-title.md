# Title Cleaning Task

{{language_requirement}}

Your task is to clean and format a document title by removing any preambles, prefixes, or meta-descriptions.

**Critical Requirements:**

1. **Remove all preambles and prefixes** such as:
   - "Document Structure:"
   - "Schema Strutturale per il Documento di Studio:"
   - "Structural Schema:"
   - "Study Document Outline:"
   - "Document Title:"
   - "Outline for:"
   - "Structural Schema for:"
   - Any similar introductory text or meta-descriptions in any language
2. **Preserve the actual title content** that follows the preamble, preserving its language as well
3. **Handle edge cases:**
   - If the title contains a colon that is part of the actual title (not a separator), preserve it
   - If there's no preamble to remove, return the title as-is
   - Preserve any LaTeX formatting in the title (e.g., \(...\) for inline math)

**Examples:**

**Input:** "Schema Strutturale per il Documento di Studio: Fondamenti e Applicazioni della Genomica e Trascrizione Vegetale"
**Output:** {"title": "Fondamenti e Applicazioni della Genomica e Trascrizione Vegetale"}

**Input:** "Structural Schema: The Anatomy and Function of the Heart"
**Output:** {"title": "The Anatomy and Function of the Heart"}

**Input:** "Study Document Outline: Quantum Mechanics and Wave-Particle Duality"
**Output:** {"title": "Quantum Mechanics and Wave-Particle Duality"}

**Input:** "Document Title: Introduction to Machine Learning Algorithms"
**Output:** {"title": "Introduction to Machine Learning Algorithms"}

**Input:** "The Anatomy and Function of the Heart: Chambers, Valves, and Cardiac Cycle"
**Output:** {"title": "The Anatomy and Function of the Heart: Chambers, Valves, and Cardiac Cycle"}

**Input:** "Quantum Mechanics and Wave-Particle Duality"
**Output:** {"title": "Quantum Mechanics and Wave-Particle Duality"}

---

**Title To Clean:**

{{title}}

---

**Output Format:**

Return only a valid JSON object with a "title" field, with no additional text, explanations, or formatting outside the JSON as follows:

{"title": "Cleaned Title Here"}
