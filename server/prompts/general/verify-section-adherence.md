You are verifying whether a generated document section adheres to its expected coverage requirements. The following are the inputs to your task.

## Inputs

### Section Title

{{section_title}}

### Expected Coverage

{{expected_coverage}}

### Generated Section

{{generated_section}}

---

## Task

Evaluate whether the generated section content:

1. Uses the exact section title verbatim as specified
2. Covers all topics and concepts mentioned in the expected coverage
3. Provides appropriate depth and detail for each required topic **based on its emphasis level**
4. Maintains proper proportionality - Low emphasis concepts are brief, High emphasis concepts are detailed
5. Does not over-elaborate on Low emphasis concepts using reference material
6. Contains no significant omissions or deviations from the expected coverage
7. **Adheres to LaTeX formatting requirements** (critical):
   - Uses correct math delimiters: \(...\) for inline math, \[...\] for display math
   - **Never** uses dollar signs ($) for math (only acceptable for currency)
   - Uses markdown formatting (_italics_, **bold**) for emphasis, **not** LaTeX commands (\textbf{}, \textit{}, \text{})
   - **Never** uses emphasis or highlighting commands inside math expressions (\textbf{}, \mathbf{}, \boldsymbol{}, etc.)
   - **Never** splits equations between math mode and plain text (entire equations must stay within one set of delimiters)
   - All LaTeX commands are inside math delimiters
   - Mathematical notation and formulas are properly formatted
   - Scientific terminology and acronyms are accurate and written in plain text (not in math mode)
   - Chemical formulas use subscripts correctly (e.g., H\(\_2\)O) without \ce{} commands
   - No 2D structural formulas or multi-dimensional mathematical representations

## Output Format

Return your evaluation in the following format:

```
<coverage_score>[Number from 1-100]</coverage_score>
```

**Coverage Score Guidelines:**

- **90-100**: Excellent coverage
  - Uses the exact section title as specified (verbatim match)
  - All key concepts, definitions, and principles from expected coverage are present
  - **Emphasis-based proportionality is correctly maintained:**
    - High emphasis concepts have comprehensive, detailed coverage with full explanations
    - Medium emphasis concepts have moderate, clear coverage
    - Low emphasis concepts are brief (1-2 paragraphs maximum)
  - All specific examples, case studies, demonstrations are included **as discussed in the lecture**
  - All exercises, problems, or practice questions mentioned are covered
  - Mathematical formulas, equations, and technical details are complete
  - Real-world applications and implications are discussed as expected
  - Appropriate depth and detail for each topic **matching its emphasis level**
  - Reference materials are used appropriately for terminology/verification, not content expansion
  - No significant omissions
  - **Perfect LaTeX formatting conformity:**
    - All math uses \(...\) and \[...\] delimiters exclusively (no $ or $$)
    - Text emphasis uses markdown formatting only (no \textbf{}, \textit{})
    - No emphasis or highlighting commands inside math expressions (\textbf{}, \mathbf{}, etc.)
    - Equations are never split between math mode and plain text
    - All LaTeX commands are properly enclosed in math delimiters
    - Scientific terminology and chemical formulas are correctly formatted
    - No LaTeX formatting errors

- **70-89**: Good coverage
  - Section title matches the expected title (minor wording variations acceptable)
  - Most key concepts and principles are covered
  - **Proportionality is mostly maintained:**
    - High emphasis concepts have good detail, though may lack some nuance
    - Medium emphasis concepts are adequately covered
    - Low emphasis concepts are reasonably brief, though one or two may be slightly over-elaborated
  - Most examples and demonstrations from the lecture are included
  - Minor omissions acceptable (e.g., one or two secondary examples missing)
  - Core technical content and formulas are present
  - Depth is generally appropriate for emphasis levels, though some topics may be slightly less detailed than expected or slightly over-detailed
  - **Perfect LaTeX formatting conformity:**
    - All math uses \(...\) and \[...\] delimiters exclusively (no $ or $$)
    - Text emphasis uses markdown formatting only (no \textbf{}, \textit{})
    - No emphasis or highlighting commands inside math expressions (\textbf{}, \mathbf{}, etc.)
    - Equations are never split between math mode and plain text
    - All LaTeX commands are properly enclosed in math delimiters
    - Scientific terminology and chemical formulas are correctly formatted
    - No LaTeX formatting errors

- **50-69**: Partial coverage
  - Section title is similar but may have noticeable differences from expected
  - Significant topics or concepts from expected coverage are missing
  - **Proportionality issues present:**
    - Low emphasis concepts are over-elaborated (exceed 2 paragraphs significantly)
    - High emphasis concepts lack sufficient detail or feel superficial
    - Reference materials appear to dominate over lecture content in some areas
  - Multiple examples or exercises from the lecture are omitted
  - Insufficient depth in explanations for High emphasis concepts
  - Some required technical content or formulas are missing
  - Coverage feels incomplete or superficial in important areas, or over-expanded in minor areas
  - **Mostly correct LaTeX formatting:**
    - Most math uses correct delimiters \(...\) and \[...\], with only occasional $ usage
    - Mostly uses markdown formatting for emphasis, though may have 1-2 instances of \textbf{} or \textit{}
    - May have 1-2 instances of emphasis commands inside math expressions
    - May have 1-2 instances of split equations between math mode and plain text
    - LaTeX commands are generally inside math delimiters
    - Minor LaTeX formatting issues that don't significantly impact readability

- **Below 50**: Poor coverage
  - Section title is incorrect, significantly different, or missing
  - Major omissions of core concepts, definitions, or principles from the lecture
  - **Severe proportionality violations:**
    - Low emphasis concepts are massively over-elaborated (multiple paragraphs or subsections)
    - High emphasis concepts are missing or minimally covered
    - Reference materials clearly dominate over lecture content
    - Content expansion far exceeds what the professor discussed
  - Most examples, exercises, or demonstrations from the lecture are missing
  - Minimal depth or explanation of High emphasis topics that require it
  - Critical technical content, formulas, or applications are absent
  - Content deviates significantly from expected coverage
  - Section may cover wrong topics or irrelevant material not discussed in the lecture
  - **Significant LaTeX formatting issues:**
    - Frequent use of $ or $$ for math delimiters instead of \(...\) and \[...\]
    - Multiple instances of \textbf{}, \textit{}, or \text{} for text emphasis
    - Multiple instances of emphasis commands inside math expressions (\textbf{}, \mathbf{}, etc.)
    - Multiple instances of split equations between math mode and plain text
    - Some LaTeX commands appear outside math delimiters
    - Formatting issues that impact readability or rendering

**Important:**

- The `coverage_score` must be a number between 1 and 100

Return **only** the specified format above, with no additional text or formatting.
