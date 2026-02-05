{{language_requirement}}

---

You are now generating detailed study material for a specific section. This is part of a larger study document being built sequentially, section by section. Your task is to transform the provided lecture content into a thorough study document section, ensuring **maximum fidelity to the core reasoning and factual statements** from the lecture transcript. This is not an act of summarization, nor of omission, but of presentation in such a way that an expert would unequivocally approve of the study document for comprehensive understanding of the subjects discussed in the lecture.

**Critical Instructions - Source Hierarchy and Proportionality:**

- **Primary Source**: The audio transcript is your primary and dominant source. It defines what content to include, how deeply to cover each concept, and the depth of explanation.
- **Secondary Source**: Reference materials are provided for terminology verification, accurate definitions, citations, and proportional enrichment only. They must **never** dominate or exceed the lecture content in proportion.
- **Proportionality Principle**: The depth of coverage for each concept must match the professor's treatment in the lecture. If the professor spent less than 5 minutes on a topic (i.e., Low emphasis), do not write 3 pages about it even if reference materials contain extensive information. Brief mentions remain brief; extensively discussed topics with a Medium to High emphasis receive detailed coverage, proportionally to their emphasis level.

The study document section must not simply present conclusions; it must present the full reasoning contained in the lecture, linking and cross-linking together the topics discussed, bringing the student through a complete understanding of the subjects being discussed by unraveling all explanations from implicit to explicit, connecting the concepts in a seamless manner. It must be a chronological, pedagogical document that guides a learner who has not yet studied the topic to understand thoroughly and with extreme accessibility the content and to follow the reasoning presented. Focus the logical decomposition and intermediate steps _only on new mechanisms_ or _specific interactions_ discussed in this section. When referencing previous concepts, limit to their function in the current context, avoiding explicitating internal logical steps already treated. The resulting study document section must be written in Markdown format with LaTeX formulas embedded within it, and not a full LaTeX document. Use tables only when strictly necessary, preferring bullet points or lists. Nothing in the resulting document should indicate that it is a study document; it should be presented as a full, transparent superior representation of the lecture material. In drafting this study document section, preserve an authoritative tone, guiding the reader through the reasoning as if an expert is explaining the concepts directly, one-on-one, providing an immersive experience rather than merely reporting facts. The immersive experience must be achieved primarily through a **clear, explanatory style using the present tense** where applicable.

**Current Section to Generate:** {{section_title}}

### Section to Generate Coverage Requirements

{{section_coverage}}

### How to Use the Provided Information

**Step 1: Understand the Section Scope and Emphasis Levels**

The **Coverage** field above specifies exactly what topics, concepts, and content from the lecture must be included in this section. This is derived from a comprehensive structural analysis that maps every part of the lecture to a specific section.

**Critical Instructions**: The **Introduces** field contains each concept with its **Emphasis level** (High/Medium/Low). These emphasis levels are your proportionality guide:

- **High emphasis concepts**: Develop fully with all the professor's explanations, examples, reasoning, and detailed elaboration
- **Medium emphasis concepts**: Provide clear, complete coverage matching the professor's moderate treatment - thorough but not exhaustive
- **Low emphasis concepts**: Include with minimal elaboration - typically 1-2 paragraphs maximum, brief explanation or definition only, even if reference materials contain extensive information, possibly even merging them within Medium to High emphasis concepts to prevent excessive fragmentation of concepts.

Your first task is to identify all portions of the lecture transcript that correspond to the topics listed in the **Coverage** field, paying careful attention to the emphasis level assigned to each concept.

**Step 2: Extract Relevant Content from Transcript (Primary) and References (Secondary)**

**Priority Order for Content Extraction:**

1. **First, scan the lecture transcript** to locate all content that matches the **Coverage** requirements:
   - Direct explanations of the listed topics and concepts from the professor
   - Examples, case studies, or demonstrations mentioned by the professor
   - Mathematical formulas, equations, or technical details explained in the lecture
   - Any exercises, problems, or practice questions discussed in the lecture
   - Real-world applications or implications discussed by the professor
   - Any analogies, metaphors, or teaching aids used by the lecturer

2. **Then, consult reference materials** only for:
   - **Terminology verification**: Ensuring technical terms are correctly spelled and defined
   - **Accurate definitions**: Cross-referencing formal definitions when the professor mentioned a term
   - **Citations**: Supporting claims or findings discussed by the professor with reference sources
   - **Proportional enrichment**: For High or Medium emphasis concepts only, you may use reference materials to add context or clarify terminology - but never exceed the professor's treatment depth

3. **Restrictions on Reference Material Usage:**
   - **For Low emphasis concepts**: Use references **only** for terminology accuracy, **not** for content expansion
   - **Never expand beyond the lecture**: If the professor mentioned a concept briefly, keep it brief regardless of reference material depth
   - **Proportionality check**: Reference-derived content should represent a small fraction of each concept's coverage compared to lecture-derived content

**Step 3: Apply Analysis Metadata**

Use the structural analysis metadata to guide your content transformation:

- **"Builds on" / "Transitions from"** (if present): Concepts listed here were explained in previous sections. When these appear in the source material for this section, reference them briefly (e.g., "As discussed in the previous section...") without repeating their full explanations. Note: This field is not present for the first section.

- **"Introduces"**: These are the new concepts that this section must fully explain. When you encounter these topics in the source material, transform them into comprehensive, pedagogical explanations with all reasoning made explicit.

- **"Avoid repeating"** (if present): If the source material (transcript) contains explanations of these topics, **do not** include those explanations in this section. These topics were already covered in previous sections. If mentioned in the current source content, only acknowledge them with brief cross-references. Note: This field is not present for the first section.

- **"Transitions to"** (if present): This indicates what comes next. End your section with language that naturally anticipates the next topic without explaining it.

- **"Discussion Points"** (if present): If Q&A sessions, debates, or interactive elements are noted, incorporate the relevant questions and their answers naturally into the explanatory flow. Present these as part of the pedagogical narrative rather than as separate Q&A sections.

- **"Potential Challenges"** (if present): Pay special attention to concepts identified as commonly misunderstood or difficult. For these topics, provide extra clarity, additional examples, or explicit step-by-step reasoning to preemptively address misconceptions and ensure student comprehension.

**Step 4: Transform Lecture Content into Study Material with Emphasis-Based Depth**

For each concept identified in Step 2, apply the following transformation based on its emphasis level:

**For High Emphasis Concepts:**

1. Extract **all** core reasoning, factual statements, and explanations from the lecture
2. Unravel implicit reasoning into explicit steps with full intermediate logical progression
3. Include all examples, demonstrations, and applications discussed by the professor
4. Develop comprehensive, detailed explanations matching the professor's extensive treatment
5. You may use reference materials proportionally for terminology enrichment and citations
6. Typical length: Multiple paragraphs to full subsections (###), always exhaustively covering the concept to the extent of the lecture

**For Medium Emphasis Concepts:**

1. Extract the core reasoning and key explanations from the lecture
2. Unravel implicit reasoning where necessary, but more concisely than High emphasis concepts
3. Include primary examples mentioned by the professor
4. Provide clear, complete coverage without exhaustive detail
5. Use reference materials sparingly for terminology and accuracy
6. Typical length: 2-4 paragraphs, but always exhaustively covering the concept to the extent of the lecture

**For Low Emphasis Concepts:**

1. Extract only the essential statement or definition from the lecture
2. Provide minimal elaboration, meaning just enough for clarity
3. Do **not** unravel into extensive logical steps
4. Do **not** add examples unless the professor provided one
5. Use reference materials **only** for terminology accuracy, **not** content expansion
6. **Maximum length: 1-2 paragraphs** regardless of reference material availability
7. **Critical Instruction on Low Emphasis Concepts**: Resist the temptation to elaborate just because references contain extensive information
8. Prefer to merge Low emphasis concepts if possible into the Medium and High emphasis concepts to prevent excessive fragmentation of the section

**General Transformation Steps:**

1. Connect concepts seamlessly with clear transitions
2. Present in chronological order as it appears in the lecture (where pedagogically appropriate)
3. Use the authoritative, explanatory tone with present tense where applicable
4. Maintain proportionality - the space allocated to each concept must reflect its emphasis level

You must maintain continuity by referencing concepts explained earlier naturally, such as 'As discussed in the previous section...' or 'Building on the concept of X...'. **The avoidance of content repetition is the highest priority.** If a topic or concept (e.g., a key term, a historical figure's role, a financial metric's definition) is detailed in a previous section, treat its mention in the current section's Coverage Requirements as a **cross-reference or a concise summary of its relevance to the current topic**, and _do not_ repeat the full, underlying explanation.

The avoidance of content repetition must override maximum fidelity: if the professor's transcript repeats a concept, mechanism, or definition already detailed in a previous section, prioritize non-repetition by cross-referencing or summarizing relevance without repeating the full explanation.

**Proportionality Verification Before Finalizing:**

After drafting the section, perform this critical check:

1. **Review Low emphasis concepts**: Are they 1-2 paragraphs maximum? If any Low emphasis concept exceeds this, reduce it immediately.
2. **Review High emphasis concepts**: Do they have comprehensive, detailed coverage with full reasoning? If any High emphasis concept feels thin or brief, expand it using the professor's explanations from the transcript.
3. **Check reference material proportion**: Does lecture content dominate over reference-derived content? Reference materials should enhance, not overshadow, the lecture.
4. **Verify emphasis balance**: The bulk of the section's depth and detail should be allocated to High emphasis concepts, with progressively less for Medium and Low.

For smooth transitions, begin this section with a natural transition from the previous section's ending, end it with language that naturally leads into the next topic based on the structural outline provided, and ensure the flow is seamless with no abrupt jumps or disconnects. **The transition language (both beginning and end) must be brief and focused, mentioning only the previous or next topic to ensure a smooth, anticipatory flow without pre-empting the content of the upcoming section.** To avoid gaps, cross-reference the source material and reference materials to ensure no content belonging to this section is skipped, and if the source content discussed something in this part, it must appear in your output.

For pedagogical quality, ensure the tone and level of detail remain consistent across sections, following the same thorough, explanatory style as the existing study document by unraveling all implicit reasoning into explicit steps, connecting concepts seamlessly, preserving an authoritative tone and voice, using precise scientific terminology, and including all necessary intermediate logical steps.

{{citation_instructions}}

Ensure that footnotes are concise, direct, and to the point. They should be no longer than one sentence, as excessively long and verbose footnotes in the extended version are undesirable. Remember to strictly adhere to the citation limit of 5-6 per section maximum; do not overuse citations even if reference material is abundant.

**Critical Instruction:** You **must** use the exact section title "{{section_title}}" verbatim as the level 2 heading. Do not modify, rephrase, or generate a new title based on content. Failure to comply will result in rejection.

The document section must begin directly with the exact title provided as "## {{section_title}}", without any modification, rephrasing, or basing it on lecture subjects. Do not change the title; use it verbatim as the level 2 heading. The section must not be numbered; it should begin with ## followed immediately by the exact "{{section_title}}". The document's structure must not exceed the fourth level of headings, meaning that #### and deeper levels are strictly forbidden. Use level 3 headings (###) for sub-topics, sub-mechanisms, or distinct components within a major section (##) to ensure a clear pedagogical hierarchy.

Example of **correct** output:

## {{section_title}}

[Section content here]

Whenever a section contains subsections, the section must begin with a brief introductory paragraph before the first subsection appears. This introductory text provides essential context, eases the reader into the topic, and when appropriate creates a smooth transition from the previous section. Without this intervening text, the flow becomes jarring and disrupts the pedagogical experience. The introduction should naturally lead into the subsections that follow, ensuring the document maintains its seamless, professional quality throughout.

Write in Markdown format with LaTeX formulas embedded within it in \(...\) for inline math or \[...\] for display math, and **never** write LaTeX alone without being properly wrapped in a math environment—this is not a full LaTeX document. **All LaTeX math must be written linearly, absolutely forbidding bi-dimensional representations of chemical structures and math structures.** The section's main heading should be level 2 (## {{section_title}}), with any subsections at level 3 (###), and no deeper levels. Use level 3 headings (###) for **every significant thematic element** mentioned in the section coverage, or for **every distinct multi-step mechanism** that is introduced. For simple lists of sub-points (causes, components, examples) or for explaining terms, use bold text and bullet points. When listing related concepts or variables, use a standard bullet point list (\*) with the key term in **bold** to ensure visual consistency. For math rendering, use display math (\[...\]) for standalone equations and inline math (\(...\)) for equations within paragraphs, ensuring all LaTeX math is written and rendered correctly. If this section contains subsections, begin with a brief introductory paragraph before the first subsection appears to provide essential context and create smooth transitions. **Absolutely never** write math content or LaTeX commands outside of or without these \(...\) or \[...\] delimiters.

# LaTeX Instructions

{{latex_instructions}}

Here is the full structural outline for reference, so you understand how this section fits into the overall document:

{{structure_outline}}

## Example Template

{{example_template}}

**Important Reminder:** The resulting study document section must be written in **Markdown format** with **LaTeX formulas embedded within it** as shown in the example template. It is **not** a full LaTeX document. Formulas must always be wrapped in \(...\) for inline math or \[...\] for display math, as otherwise they wouldn't be rendered correctly. For example, \max{N} and T_m are wrong, but \(\max{N}\) and \(T_m\) are right.

Do not enclose the output in code blocks. Output the Markdown content directly. Begin directly with the section heading and content. Do not include any meta-commentary like "Here is the section..." or wrap your output in code blocks. Begin each section with the title of it, then directly proceed by writing it's content below. Never skip writing the current section title.
