When creating this study document, incorporate information from the reference files provided below to support and enhance the lecturer's claims. Use the reference material to provide accurate scientific terminology where conversational terms were used in the lecture, support claims made during the lecture with information from authoritative sources, and augment the lecture content with additional context from the reference materials. The study document should present a seamless integration of the lecture content and the reference material, appearing as a cohesive whole rather than a lecture with citations added afterward.

### Key Guidelines for Citations and Footnotes

- **Never Cite the Lecture Transcript**: The lecture transcript itself must never be cited. Citations should only reference the provided reference files, not the transcript. This is a critical rule to maintain the integrity of the study document.
- **Citation Style**: Citations must be done using inline footnotes in the format {{{content-filename-pN}}}. Each citation must reference distinct content and cannot be reused within the study document. Avoid nested citations; each citation should be standalone and not contain other citations or references within it. **Critical Instruction:** Citations must clearly describe the specific information being cited from the source, not merely name the section title in the page or the source file. The content part must be a **meaningful** description of the cited information.
- **Usage Limit**: Citations should be used sparingly—no more than 7-8 per section, with a total limit of one dozen for the entire study document. This limit prevents reader confusion from excessive citations, while allowing thorough integration of reference material into the study content. **Strictly enforce this limit: do not exceed 5-6 citations per section under any circumstances. Prioritize quality over quantity in citations. Only use citations when they provide unique, valuable information that enhances the lecture content significantly; avoid citing for every minor detail or claim.**
- **Standalone Formatting**: Citations must be rephrased as independent, grammatically correct statements that stand alone as complete thoughts, not as direct excerpts, continuations, or fragments from the lecture transcript.
- **Complete Statements Required**: Every citation **must** begin with a capital letter and be a complete, grammatically correct sentence or phrase that clearly states the cited information. Citations must not be short fragments, lowercase phrases, or incomplete thoughts. Each citation must explicitly describe the piece of information being cited in a way that makes the reasoning behind the citation clear and unambiguous, but still be concise. Citations must be standalone statements that can be understood independently, without requiring context from the surrounding text. They must clearly explain what specific information is being cited and why it supports the claim in the document. This makes a sentence the perfect length for the citation's content.
- **Consistency in Notation and Terminology**: Citations must use the same notation and terminology as the surrounding content, including in LaTeX, so that all figures, symbols, notation, and terms remain consistent; the citations may be lengthy and contain substantial information, but they must adopt the document's established notation and terminology for consistency.
- **Citation Content**: Each citation must clearly specify the **meaningful and substantial** description of the piece of information extracted from the referenced page, the source file, and the page number within the inline brackets. The citation content can include \(...\) inline math expressions when referencing mathematical formulas or equations from any subject area.
- **Page Continuity**: Ensure page numbers are contiguous where possible, reflecting the sequential nature of the lecture. Avoid drastic shifts in page references unless justified.
- **Page Reference Rule**: Only use page numbers from "## Page N" headings. Do not adjust or reinterpret based on other text indications.

When adding citations to the study document, care must be taken to ensure that page numbers are properly contiguous, because the professor explains concepts in a sequence that roughly follows the progression from the first to the last pages. If there is a drastic shift in the pages from which information and footnotes are taken, there must be a reason for it and it should not be done lightly; in essence, page citations should be contiguous whenever possible.

Citations must be placed inline within the text using the format {{{information extracted-filename-pN}}}. For page ranges, use {{{information-filename-p1-p3}}}. For example, these are the only correct footnote formats:

- Single page: {{{This concept is explained in detail-filename.pdf-p5}}}
- Page range: {{{The theory is developed here-filename.pdf-p10-p15}}}

Only the pages explicitly labeled as "## Page N" must be used for citations. Do not reinterpret or adjust the page numbers based on any other indications within the text.

For example, if a page contains the following:

```markdown
## Page 31

This page discusses a topic; at the bottom, it states that this corresponds to page "16" of the reference file.
```

In this case, the correct citation must reference **page 31**, not page 16. Using any page number other than the one in the "## Page N" heading constitutes a false citation.

### Examples of Correct and Incorrect Footnotes

#### Correct Examples:

- {{{The mitochondria is the powerhouse of the cell-biology_textbook.pdf-p45}}}
- {{{Photosynthesis converts light energy into chemical energy-botany_guide.pdf-p12-p18}}}
- {{{Newton's laws of motion are fundamental to classical physics-physics_principles.pdf-p3}}}
- {{{DNA replication occurs during the S phase of the cell cycle-molecular_biology.pdf-p67-p72}}}
- {{{The equation \(x^2 + y^2 = z^2\) is the Pythagorean theorem-mathematics.pdf-p15}}}
- {{{Newton's laws describe the relationship between force, mass, and acceleration in classical mechanics-physics.pdf-p5}}}
- {{{Chemical reactions involve the rearrangement of atoms to form new substances-chemistry.pdf-p12}}}
- {{{The quadratic formula solves equations of the form \(ax^2 + bx + c = 0\)-mathematics.pdf-p20}}}
- {{{Historical events are often influenced by economic factors leading to social changes-history.pdf-p15}}}
- {{{Artistic movements reflect cultural and technological advancements of their time-art_history.pdf-p8}}}
- {{{The Krebs cycle generates ATP through oxidative phosphorylation-cell_biology.pdf-p23}}}
- {{{Quantum mechanics describes particle behavior at the atomic level-physics_textbook.pdf-p45-p52}}}
- {{{Antibiotics target bacterial cell walls to inhibit growth-microbiology.pdf-p18}}}
- {{{The Fibonacci sequence appears in natural patterns such as plant growth-mathematics.pdf-p12}}}
- {{{Climate change accelerates glacial melting due to rising temperatures-environmental_science.pdf-p67-p71}}}
- {{{Shakespeare's sonnets explore themes of love and mortality-literature.pdf-p5}}}
- {{{The periodic table organizes elements by atomic number-chemistry.pdf-p3}}}

#### Incorrect Examples:

- {{{biology_textbook.pdf-p45}}} (Missing the content description)
- {{{The mitochondria is the powerhouse of the cell-p45}}} (Missing the filename)
- {{{This concept is explained here-biology_textbook.pdf-p45, physics_principles.pdf-p12}}} (Comma-separated multiple citations not allowed)
- {{{Important information-biology_textbook.pdf-p45-p50-p55}}} (Non-contiguous page ranges not allowed)
- {{{Key concept-biology_textbook.pdf-page45}}} (Should use 'p' not 'page')
- {{{The theory is detailed here-biology_textbook.pdf-p45}}} (Reused citation - each must be unique)
- {{{This matches page 16 of the original-biology_textbook.pdf-p16}}} (Using page number from text instead of "## Page N" heading)
- {{{This concept is discussed in the lecture-transcript}}} (Citing the lecture transcript, which is prohibited)
- {{{The buffer includes enzymes-transcript}}} (Citing the lecture transcript)
- {{{Enzymes degrade the molecule-transcript}}} (Citing the lecture transcript)
- {{{Molecules form complex structures-transcript}}} (Citing the lecture transcript)
- {{{The resistance is conferred by the gene {{additional reference}}-reference.pdf-p5}}} (Nested citation containing another citation within it)
- {{{The plasmid has a selective marker {{see note}}-reference.pdf-p10}}} (Nested citation containing another citation within it)
- {{{la reazione di PCR è un processo "passo dopo passo". Ogni ciclo raddoppia il numero di molecole di DNA, portando a una crescita esponenziale del prodotto.-transcript}}} (Direct excerpt from transcript, not a standalone citation)
- {{{inserimento di un sito di restrizione alle estremità di un prodotto di PCR.-transcript}}} (Fragment excerpt lacking proper grammar and independence)
- {{{i plasmidi sono molecole di DNA circolare a doppio filamento, caratterizzate da una conformazione superavvolta.-transcript}}} (Transcript continuation, not rephrased as standalone information)
- {{{eliminare tutto il detrito cellulare, tutte le pareti batteriche, tutto il tipo.-transcript}}} (Direct quote from transcript, improperly formatted)
- {{{il gel viene collocato nell'apparecchio per l'elettroforesi e immerso nel tampone di corsa TAE-transcript}}} (Excerpt continuation, not an independent statement)
- {{{cell_biology.pdf-p23}}} (Missing content description)
- {{{The Krebs cycle generates ATP-p23}}} (Missing filename)
- {{{This process is explained-cell_biology.pdf-p23, physics.pdf-p10}}} (Comma-separated multiple citations not allowed)
- {{{Important information-cell_biology.pdf-p23-p30-p45}}} (Non-contiguous page ranges not allowed)
- {{{The theory is detailed here-cell_biology.pdf-page23}}} (Should use 'p' not 'page')
- {{{Quantum mechanics describes particle behavior-physics_textbook.pdf-p45}}} (Reused citation - each must be unique)
- {{{This matches page 16 of the original-cell_biology.pdf-p16}}} (Using page number from text instead of "## Page N" heading)
- {{{This concept is discussed in the lecture-transcript}}} (Citing the lecture transcript, which is prohibited)
- {{{The resistance is conferred by the gene {{{additional reference}}}-reference.pdf-p5}}} (Nested citation containing another citation)
- {{{la reazione di PCR è un processo "passo dopo passo"-transcript}}} (Direct excerpt from transcript, not a standalone citation)
- {{{inserimento di un sito di restrizione-transcript}}} (Fragment lacking proper grammar and independence)

### Behaviors to Avoid and Proper Alternatives

To ensure citations are accurate, consistent, and compliant with the guidelines, avoid the following behaviors. Each includes an explanation of why it's incorrect and how to handle it properly:

1. **Omitting the content description**: Citations like {{{biology_textbook.pdf-p45}}} lack the specific information extracted, making them vague. Instead, always include a clear description of the content, e.g., {{{The mitochondria is the powerhouse of the cell-biology_textbook.pdf-p45}}}.
2. **Omitting the filename**: Citations like {{{The mitochondria is the powerhouse of the cell-p45}}} miss the source file, hindering traceability. Always include the filename, e.g., {{{The mitochondria is the powerhouse of the cell-biology_textbook.pdf-p45}}}.
3. **Using comma-separated multiple citations**: Formats like {{{This concept is explained here-biology_textbook.pdf-p45, physics_principles.pdf-p12}}} combine multiple sources in one bracket, which is not allowed. Use separate, distinct citations for each source, ensuring each is unique and placed appropriately in the text.
4. **Using non-contiguous page ranges**: Citations like {{{Important information-biology_textbook.pdf-p45-p50-p55}}} skip pages, violating continuity. Use contiguous ranges where possible, e.g., {{{Important information-biology_textbook.pdf-p45-p50}}}, or separate citations for non-contiguous pages.
5. **Using 'page' instead of 'p'**: Formats like {{{Key concept-biology_textbook.pdf-page45}}} use the wrong abbreviation. Always use 'p' for page, e.g., {{{Key concept-biology_textbook.pdf-p45}}}.
6. **Reusing citations**: Repeating the same citation like {{{The theory is detailed here-biology_textbook.pdf-p45}}} multiple times violates uniqueness. Each citation must reference distinct content; rephrase or find new information to cite if needed.
7. **Using page numbers from text instead of headings**: Citations like {{{This matches page 16 of the original-biology_textbook.pdf-p16}}} use numbers from the content rather than "## Page N" headings. Always reference the explicit page heading, e.g., if the heading is "## Page 31", use p31 regardless of other indications.
8. **Citing the lecture transcript**: Citations like {{{This concept is discussed in the lecture-transcript}}} reference the transcript, which is prohibited. Only cite reference files; integrate transcript content without citing it directly.
9. **Including nested citations**: Citations like {{{The resistance is conferred by the gene {{{additional reference}}}-reference.pdf-p5}}} contain other citations within them, creating confusion. Keep each citation standalone; if multiple references are needed, use separate citations in the text.
10. **Formatting citations as transcript excerpts**: Avoid using direct quotes or fragments from the transcript as citations, as this violates the no-transcript rule and results in improperly formatted entries. Rephrase information from reference files into standalone statements with correct grammar. Example incorrect: {{{la reazione di PCR è un processo "passo dopo passo"-transcript}}}; Example correct: {{{Chemical reactions proceed through sequential steps-chemistry.pdf-p8}}}.

### Proper Citation Practices

To create effective and compliant citations, follow these steps:

1. **Identify valuable content**: Only cite when the reference material provides unique, valuable information that significantly enhances the lecture content. Avoid citing for minor details.
2. **Extract specific information**: Clearly describe the piece of information from the page, ensuring it's distinct and not reused.
3. **Reference the correct source**: Always use the filename of the reference file, not the transcript.
4. **Use accurate page numbers**: Only use page numbers from "## Page N" headings, ensuring contiguous ranges where possible.
5. **Format correctly**: Use the format {{{information extracted-filename-pN}}} for single pages or {{{information-filename-p1-p3}}} for ranges.
6. **Maintain limits**: Limit to 7-8 citations per section and no more than a dozen total, prioritizing quality and coverage of each part of the section, avoiding to nest all the citations in just one specific subsection.
7. **Ensure uniqueness**: Each citation must reference distinct content and not be repeated.
8. **Integrate seamlessly**: Place citations inline within the text to maintain a cohesive document flow.
9. **Avoid prohibited elements**: Never nest citations, cite the transcript, or use incorrect formats.
10. **Format citations as standalone entries**: Rephrase extracted information into complete, independent sentences or phrases that can stand alone, ensuring proper grammar and syntax.

**Critical Instruction Reminder:** Moreover, citations **must** clearly describe the specific information being cited from the source, not merely name the section title in the page or the source file. The content part must be a **meaningful** description of the cited information.
