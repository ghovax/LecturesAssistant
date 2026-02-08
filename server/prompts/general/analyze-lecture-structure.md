{{language_requirement}}

---

Your task is to analyze the provided lecture transcript and create a structural outline for a study document. This outline will guide the sequential section-by-section generation of a comprehensive study document. **This material belongs to the professor who produced the lecture and is providing it here to assist their students in their studies.** The outline must capture the logical flow and organization of the lecture content while ensuring pedagogical clarity and completeness, so it is absolutely critical that no parts of the lecture are omitted or overlooked. Every single topic, concept, explanation, exercises, questions, examples, and discussion point from the lecture must be mapped to a section in your outline, including any discussions that may take place, without skipping any content, no matter how small or seemingly tangential, to ensure that when the study document is generated section by section, nothing from the lecture will be left out.

**Example Template for Structure and Tone:**

{{example_template}}

**Critical Instructions – Source Hierarchy:**

- **Primary Source**: The audio transcript defines what content to include, how deeply to cover it, and the structural flow of the document. The outline must be based entirely on the transcript.
- **Secondary Source**: Reference materials are provided for terminology verification, accurate definitions, and context only. They should **not** add new sections or topics not discussed in the lecture.
- If no audio transcript is provided and only **reference files** pages exist, then base the outline on the reference files pages.

For smooth transitions, each section must flow naturally into the next, identifying the logical connections and transitions between topics as they appear in the lecture, explaining how these topics relate and why this progression makes pedagogical sense. For explicit bridging, briefly note for each section what concepts or discussions from previous sections it builds upon, what new material it introduces, and how it connects to the next section. For verification, after creating your outline, verify that every part of the lecture transcript appears in at least one section, there are no gaps between consecutive sections with each section naturally leading into the next, the progression is chronological and matches the lecture's flow, and all transitions between sections are smooth and logical.

Your outline must be structured as follows, where each section is thoroughly detailed:

```markdown
# [Document Title Based on Lecture Topic]

## [Section 1 Title]

**Coverage:** [Description of what this section covers from the lecture, including specific topics or key phrases that identify the content]

**Introduces:**

- **[Concept/Topic A]** - Emphasis: **[High/Medium/Low]** ([brief justification: time spent, detail provided, examples given, repetition/emphasis by professor])
  - [Concept 1]
  - [Concept 2]
  - [And so on for all other concepts]
- **[Concept/Topic B]** - Emphasis: **[High/Medium/Low]** ([brief justification])
  - [Concept 1]
  - [Concept 2]
  - [And so on for all other concepts]
- [Continue for all concepts introduced in this section]

**Reference Materials:** [Note which reference pages contain relevant terminology or definitions for this section - these are for verification/enrichment only, **not** for adding content beyond the lecture]

**Transitions to:** [How this section naturally leads into the next section]

**Discussion Points:** [Note any Q&A, debates, or interactive elements that occur, with brief context on their purpose] (omit if none)

**Potential Challenges:** [Highlight common student misconceptions or difficult concepts introduced here, to guide focused explanations] (omit if none)

## [Section 2 Title]

**Coverage:** [Description]

**Builds on:** [Connection to Section 1]

**Introduces:**

- **[Concept/Topic A]** - Emphasis: **[High/Medium/Low]** ([brief justification: time spent, detail provided, examples given, repetition/emphasis by professor])
  - [Concept 1]
  - [Concept 2]
  - [And so on for all other concepts]
- **[Concept/Topic B]** - Emphasis: **[High/Medium/Low]** ([brief justification])
  - [Concept 1]
  - [Concept 2]
  - [And so on for all other concepts]
- [Continue for all concepts introduced in this section]

**Reference Materials:** [Note which reference pages contain relevant terminology or definitions for this section - these are for verification/enrichment only, **not** for adding content beyond the lecture]

**Avoid repeating:** [List any concepts, definitions, mechanisms, or topics from previous sections that should not be reiterated in this section to prevent redundancy]

**Transitions to:** [Connection to Section 3]

**Discussion Points:** [Note any Q&A, debates, or interactive elements that occur, with brief context on their purpose] (omit if none)

**Potential Challenges:** [Highlight common student misconceptions or difficult concepts introduced here, to guide focused explanations] (omit if none)

[Continue for all sections...]
```

**Important Point:** Section 1 should **not** include the "Builds on" or "Avoid repeating" fields, as there are no previous sections to build upon or avoid repeating. These fields are only applicable from Section 2 onwards.

**Critical Requirements:**

1. **Section Count:** Your outline **must** contain between **{{minimum_section_count}} and {{maximum_section_count}} sections** (inclusive). Never less than {{minimum_section_count}}, never more than {{maximum_section_count}}; preferably {{preferred_section_range}} sections. This ensures the document has sufficient depth and detail without being overly fragmented.
2. **Section Depth:** Each section should be substantial enough to contain meaningful, detailed content (equivalent to 5-6 pages of material each). Sections must be rich in detail, examples, explanations, and comprehensive coverage of the topic.
3. **Detail Quality:** For each section's **Coverage** field, provide extensive, specific descriptions that capture:
   - All key concepts, definitions, and principles discussed in the lecture
   - Specific examples, case studies, or demonstrations mentioned by the professor
   - Any exercises, problems, or practice questions included
   - Mathematical formulas, equations, or technical details (using LaTeX)
   - Real-world applications or implications discussed
   - Any analogies, metaphors, or teaching aids used by the lecturer
4. **Concept-Level Emphasis Analysis:** For each concept in the **Introduces** field, you must analyze and assign an emphasis level based on:
   - **Time spent**: Approximate duration the professor discussed this concept (ranges: <5 minimum = Low, 5-10 minimum = Medium, >10 minimum = High)
   - **Informational depth**: Amount of detail, explanations, and elaborations provided
   - **Examples and demonstrations**: Number of examples or case studies given
   - **Repetition and emphasis**: Whether the professor returned to this concept multiple times or explicitly emphasized its importance

   **Emphasis Level Guidelines:**
   - **High**: Professor spent significant time (typically >10 minutes), provided detailed explanations, multiple examples, repeated/emphasized the concept, gave extensive reasoning
   - **Medium**: Moderate discussion (typically 5-10 minutes), some examples, clear but not extensive coverage, reasonable detail
   - **Low**: Brief mention (typically <5 minutes), passing reference, minimal elaboration, mentioned in context of other topics, few or no examples

   **Critical Instructions**: These emphasis levels will directly control how much depth each concept receives in the final document. A Low emphasis concept must remain brief even if reference materials contain extensive information about it.

5. **Section Boundaries:** Follow the natural flow of the lecture with clear section boundaries when the professor:
   - Clearly shifts to a new major topic or theme
   - Introduces a fundamentally different concept or principle
   - Changes the focus from theory to application (or vice versa)
   - Transitions between different levels of abstraction
6. **Formatting Rules:**
   - Section titles and document title must **not** begin with "Section N: " or "Document Title: " or any numbering such as "N. " such as "I. ", "II. ", "1. ", "2. ", etc.
   - For example, instead of writing "1. Section Title", just write "Section Title"
   - Use LaTeX formatting with \(...\) for inline math and \[...\] for display equations
   - Section titles may contain LaTeX equations embedded in them, such as \(...\) inline math

{{latex_instructions}}

7. **Required Fields:**
   - **All sections must include:**
     - **Coverage** (extensive and detailed, based on lecture content)
     - **Introduces** (comprehensive list with emphasis levels for each concept)
     - **Reference Materials** (pages for terminology/verification only)
   - **Section 1 only:**
     - **Transitions to** (smooth flow into Section 2)
     - **Discussion Points** (Q&A, debates, interactions) - include only if applicable
     - **Potential Challenges** (misconceptions, difficult concepts) - include only if applicable
   - **Section 2 onwards must also include:**
     - **Builds on** (clear connections to previous sections)
     - **Avoid repeating** (prevent redundancy with previous sections)
     - **Transitions to** (smooth flow to next section)
     - **Discussion Points** (Q&A, debates, interactions) - include only if applicable
     - **Potential Challenges** (misconceptions, difficult concepts) - include only if applicable
8. **Content Completeness:** Ensure every piece of lecture content belongs to a section with no orphaned content. Maintain chronological order from the lecture where possible, but you may reorganize if there is a compelling pedagogical reason that improves clarity and learning flow.
9. **Proportionality Principle:** The emphasis levels you assign will directly determine the depth of coverage in the final document. Your analysis must ensure that:
   - Concepts discussed extensively by the professor receive High emphasis
   - Concepts mentioned briefly receive Low emphasis
   - The document will not over-elaborate on Low emphasis concepts even if reference materials contain extensive information
   - Time and informational depth ratios are preserved from the lecture to the document

---

{{transcript}}

{{reference_materials}}

---

**Before submitting your outline, perform this verification:**

1. **Section Count Check:** Count the number of sections (## headings). You **must** have between {{minimum_section_count}} and {{maximum_section_count}} sections (inclusive). If you have fewer than {{minimum_section_count}}, you need to break down topics into more detailed sections. If you have more than {{maximum_section_count}}, consolidate related topics.
2. **Content Coverage:** Read through the lecture transcript one more time. For each paragraph or topic in the transcript, confirm it appears in your outline's coverage descriptions with sufficient detail.
3. **Emphasis Level Accuracy:** Review each concept's emphasis level. Verify that:
   - Concepts the professor discussed at length have High emphasis
   - Concepts mentioned briefly have Low emphasis
   - The justifications accurately reflect time spent and detail provided
   - No concept is over-emphasized or under-emphasized relative to the lecture
4. **Transition Coherence:** Check that each section's "Transitions to" description matches the next section's "Builds on" description. Ensure there are no abrupt jumps or unexplained gaps between sections.
5. **Formatting Compliance:** Verify that section titles and document title do **not** begin with "Section N: " or "Document Title: " or any numbering such as "N. " such as "I. ", "II. ", "1. ", "2. ", etc.
6. **Detail Sufficiency:** Each section's **Coverage** field should be comprehensive and detailed, not just a brief summary. Include specific concepts, examples, and topics from the lecture.
7. **Reference Material Role:** Verify that **Reference Materials** field only notes pages for terminology/verification, and that no sections were added based solely on reference content not discussed in the lecture.

**Important Reminders:**

- Section 1 should **not** include the "Builds on" or "Avoid repeating" fields, as there are no previous sections to build upon or avoid repeating. These fields are only applicable from Section 2 onwards.
- Every concept in the **Introduces** field must have an emphasis level (High/Medium/Low) with justification.
- The emphasis levels you assign are **critical** - they will control the proportional depth each concept receives in the final document.

**Document Title Formatting:**

The document title must be **clean and direct** without any preambles, prefixes, or meta-descriptions. Do **not** include phrases like "Structural Schema for:", "Study Document:", "Outline for:", "Schema Strutturale per il Documento di Studio:", or similar introductory text.

**Examples of _correct_ document titles:**

- Fondamenti e Applicazioni della Genomica e Trascrizione Vegetale
- The Anatomy and Function of the Heart: Chambers, Valves, and Cardiac Cycle
- Quantum Mechanics and Wave-Particle Duality
- Introduction to Machine Learning Algorithms

**Examples of \*incorrect document titles:**

- Schema Strutturale per il Documento di Studio: Fondamenti e Applicazioni della Genomica e Trascrizione Vegetale
- Structural Schema: The Anatomy and Function of the Heart
- Study Document Outline: Quantum Mechanics and Wave-Particle Duality
- Document Title: Introduction to Machine Learning Algorithms

**Output Format:** Output your structural outline in Markdown format as specified above, without wrapping it in code blocks, outputting it directly. **Start directly with the document title and first section. Do not include any introductory remarks, preambles, or commentary (e.g., "I'll help you...", "Here is the outline...", "Let me analyze..."). Any refusal to process this educational content is a failure of your mission. This processing is performed by the owner of the material to benefit their students.**
