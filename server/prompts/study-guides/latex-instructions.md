**Critical Requirement - Scientific Accuracy and Terminology:**
All scientific acronyms and terminology **must** be accurate and precise, while also being kept consistent throughout the document. This is **extremely important** and non-negotiable. The following rules apply:

- **All terms used must be scientifically correct:** verify accuracy of technical vocabulary, chemical formulas, biological terminology, and scientific concepts
- **Acronyms must be properly defined:** on first use and correctly spelled throughout
- **Chemical formulas must be accurate:** verify correct element symbols, subscripts, and chemical structures
- **Mathematical and scientific notation must be precise:** use correct symbols, units, and conventions
- **Technical terms must be used appropriately:** in their correct scientific context
- Violating this requirement by using incorrect or imprecise scientific terminology is considered a **critical error**

**Critical Requirement - Linear Math Representation:**
Ensure all LaTeX math is written and rendered correctly. Both mathematical and chemical equations **must** be written linearly, **absolutely forbidding bi-dimensional representations** of chemical structures (e.g., structural formulas showing bonds in 2D are forbidden) and math structures (e.g., multi-row matrices without proper LaTeX commands are forbidden). All mathematical content must use proper LaTeX syntax within the prescribed delimiters. Write the math in **display math \[...\]** if standalone and in **inline math \(...\)** if within a paragraph. **Absolutely never** write math content or LaTeX commands outside of or without these delimiters.

**When to Use Math Mode vs. Plain Text:**

- **Acronyms, abbreviations and words to highlight (Critical)**: Write as **plain text** without any math delimiters; you can highlight them if you deem relevant, but only using Markdown formatting, never using LaTeX commands: use **bold** for **strong emphasis** and _italics_ for _emphasis_. Example: "The **NATO** alliance" is correctly written, but **never under any circumstances** "The \(\textbf{NATO}\) alliance" or "The \(\text{NATO}\) alliance", or even "The \(\textit{NATO}\) alliance". It is also forbidden to use \textbf{} or \text{} or \textit{} for highlighting text, unless any math mode is strictly necessary for superscripts or subscripts, and even in that case it could be written as "The NATO\(^2\)", without the need for \textbf{} or \text{}, unless very strictly necessary: which in most cases would be exceptionally rare or not necessary at all.
- **Formulas with subscripts/superscripts (Critical)**: Use **inline math** with plain text subscripts (no \text{} or derivatives needed). Example: \(x*i\), \(A_n\), \(k*{max}\), \(T^2\)
- **Proper writing of arrows:** Use \(\rightarrow\) for right arrows and \(\leftarrow\) for left arrows. Example: \(x \rightarrow y\), \(x \leftarrow y\), never leave just -> or <- in the text as they are not rendered correctly.
- **Equilibrium arrows (Critical)**: Avoid using \(\xrightleftharpoons\) as this command cannot be rendered. Instead, use simpler alternatives such as \(\rightleftharpoons\) for equilibrium reactions or \(\leftrightarrow\) for bidirectional arrows.
- **Mathematical expressions and variables (Critical)**: Use **inline math** \(...\) or **display math** \[...\]. Example: \(E = mc^2\), \[\int\_{0}^{\infty} e^{-x^2} dx = \frac{\sqrt{\pi}}{2}\]
- **Emphasis in regular text (Critical)**: Use **Markdown** formatting, not LaTeX commands. Use _italics_ for _emphasis_ or **bold** for **strong emphasis**
- **Do not ever use (extremely critical – absolute prohibition)** \text{}, \textit{}, or \textbf{} for highlighting text: use Markdown formatting instead.
- **No emphasis or highlighting inside math expressions (Extremely Critical – Absolute Prohibition)**: Mathematical expressions must **never** contain any emphasis or highlighting commands such as \textbf{}, \textit{}, \mathbf{}, \boldsymbol{}, or any other formatting commands. Math content should be written in plain LaTeX without any emphasis. Example: \(\textbf{p_C = 300}\) is **strictly forbidden** and must be written as \(p_C = 300\). Don't ever emphasize a mathematical expression.
- **Never split equations between math mode and plain text (Extremely Critical – Absolute Prohibition)**: Mathematical equations must be kept entirely within a single math delimiter pair \(...\) or \[...\]. You must **never** split an equation where some parts are inside math mode and other parts are outside. Example: "Profit = \($1800\)" is **strictly forbidden** because "Profit" is outside math mode while the rest is inside. Instead, write the entire equation in math mode: \(\text{Profit} = $1800\) or \(\text{Profit} = \textdollar 1800\). If the equation includes text labels or variable names, use \text{} to properly format them within the math environment.

**Critical Requirement - Absolute Prohibition on Dollar Signs for Math Delimiters:**
You are **strictly forbidden** from using dollar signs ($) to delimit mathematical expressions under **any** circumstances. This prohibition is absolute and non-negotiable:

- **Never** use single dollar signs ($...$) for inline math delimiters
- **Never** use double dollar signs ($$...$$) for display math delimiters
- The **only** acceptable delimiters for math mode are: \(...\) for inline math and \[...\] for display math
- Violating this requirement is considered a **critical error** that must be avoided at all costs

**Handling Currency Symbols:**
Dollar signs for currency must follow these rules:

- **Currency alone (outside math mode)**: Use dollar signs directly for simple currency values with no mathematical operations (e.g., $50, $100, $18,000)
- **Currency with mathematical operations (_must_ be inside math mode)**: Any currency value involved in mathematical expressions, calculations, or operations **must** be enclosed in \(...\) or \[...\] (e.g., \($1,800 \times 10\), \($3,000 - $1,200\))
- **Dollar signs inside math mode**: Dollar signs **can** be used within \(...\) or \[...\] as they will be properly escaped during conversion
- **Alternative inside math mode**: Use the \textdollar command for explicit dollar symbols (e.g., \(\textdollar 1,800 \times 10\))
- **Critical Rule**: **Any** mathematical operation involving currency **must** be in math mode. Writing "$1,800 \times 10$" or similar is **strictly forbidden** - use \($1,800 \times 10\) instead

**Critical Requirement - LaTeX Commands Must Be Inside Math Environments:**
You are **strictly forbidden** from writing LaTeX commands or any other backslash commands directly in markdown text outside of math delimiters \(...\) or \[...\]. This is a **critical error** that will break rendering:

- **All** LaTeX commands beginning with backslash **must** be enclosed within math delimiters
- For regular text emphasis, use markdown formatting instead: _italic_ for _italic_ or **bold** for **bold**

### Correct Examples

- The structure of DNA consists of **nucleotides**. _(Reason: Formatting is done using markdown formatting instead of LaTeX commands.)_
- The equation is \(E = mc^2\), where \(c\) is the speed of light. _(Reason: Mathematical expressions and variables must be wrapped in \(...\) for inline math.)_
- In chemistry, H\(_2\)O represents water. _(Reason: Chemical formulas shouldn't use \ce{} for proper upright typesetting, but either normal text mode or enclosed in math mode using \text{}. For example, H\(\_2\)O (preferred) and \(\text{H}\_2\text{O}\) (less preferred) are correct, but \(\ce{H2O}\) is wrong as it can't be rendered.)\_
- The derivative of \(x^2\) is \(2x\). _(Reason: Superscripts and mathematical operations require math mode.)_
- The cost is $50, but it can be sold also for $100 online. _(Reason: Simple currency values without mathematical operations can be written outside math mode.)_
- The price is $100, and the derivative of \(x^2\) is \(2x\). _(Reason: Simple currency values can be outside math mode; mathematical operations require \(...\) delimiters.)_
- Total profit = \($1,800 \times 10\) million = \($18,000\) million. _(Reason: When mixing currency with math operations, enclose the entire expression in \(...\); dollar signs inside math mode are properly escaped during conversion.)_
- Revenue per unit: \(\textdollar 3,000 - \textdollar 1,200 = \textdollar 1,800\). _(Reason: Alternative method using \textdollar command inside math mode for explicit dollar symbols.)_
- The cost increased from $500 to \($500 \times 1.2 = $600\). _(Reason: Currency outside math mode combined with currency+math inside \(...\) is correct.)_
- The profit calculation: \(\text{Profit} = $1800\). _(Reason: The entire equation is kept within math delimiters with \text{} used for the text label.)_
- The formula \(\text{Revenue} - \text{Costs} = \text{Profit}\) shows the relationship. _(Reason: Complete equation stays in math mode with \text{} for labels.)_

### Incorrect Examples

- The structure of DNA consists of \(\textbf{nucleotides}\). _(**Critical Error:** Reason: Formatting is **not** done using markdown formatting, instead it uses LaTeX commands, which may fail to be displayed correctly, so it needs to be carefully avoided at all costs.)_
- The equation is E = mc^2, where c is the speed of light. _(**Critical Error:** Reason: Mathematical expressions and variables are not wrapped in math delimiters, preventing proper rendering.)_
- In chemistry, H*2O represents water. *(Reason: Chemical formulas use subscripts outside math mode, which may not display as intended.)\_
- The derivative of x^2 is 2x. _(Reason: Superscripts and mathematical operations are used outside math mode.)_
- The equation is $E = mc^2$, where $c$ is the speed of light. _(**Critical Error - Strictly Forbidden**: Using dollar signs for inline math is absolutely prohibited; you **must** use \(...\) instead.)_
- The derivative of $x^2$ is $2x$. _(**Critical Error - Strictly Forbidden**: Using dollar signs for math is absolutely prohibited; you **must** use \(...\) instead.)_
- Eventually there could be a display equation: $$i\hbar^2$$ _(**Critical Error - Strictly Forbidden**: Using double dollar signs for display math is absolutely prohibited; you **must** use \[...\] instead.)_
- Total Profit = $1,800 \times 10 \text{ million} = $18,000 \text{ million}$ _(**Critical Error - Strictly Forbidden**: This mixes dollar signs as delimiters with currency symbols and math operations; you **must** use \(...\) delimiters instead: Total Profit = \($1,800 \times 10\) million = \($18,000\) million.)_
- Revenue = $3,000 \times 10$ million _(**Critical Error - Strictly Forbidden**: Using dollar signs as math delimiters is prohibited; use \($3,000 \times 10\) million instead.)_
- The pressure is \(\textbf{p*C = 300}\) Pa. *(**Critical Error - Strictly Forbidden**: Using emphasis commands like \textbf{} inside math expressions is absolutely prohibited; write \(p*C = 300\) instead. Don't ever emphasize a mathematical expression.)*
- The equation \(\mathbf{F = ma}\) describes force. _(**Critical Error - Strictly Forbidden**: Using \mathbf{} or any emphasis command inside math is prohibited; write \(F = ma\) instead. Don't ever emphasize a mathematical expression.)_
- Profit = \($1800\) shows our earnings. _(**Critical Error - Strictly Forbidden**: Splitting an equation between plain text and math mode is prohibited; "Profit" is outside while the rest is inside. Write \(\text{Profit} = $1800\) instead to keep the entire equation in math mode.)_
- The calculation Revenue = \($5000\) is incorrect. _(**Critical Error - Strictly Forbidden**: Never split equations across math mode boundaries; write \(\text{Revenue} = $5000\) to keep the complete equation within math delimiters.)_
