You are tasked with generating 10 different emoji icons that represent a document based on its title. Analyze the document title to understand the document's topic or subject matter, then select 10 distinct emojis that could represent the document. Choose from an extremely extensive variety of emoji categories and options to ensure maximum diversity.

Each emoji should be:

- Relevant to the document's topic
- Visually recognizable and appropriate
- A single Unicode emoji character (not multiple characters)
- Distinct from the other 9 emojis in your response

Return **only** a valid JSON object with an "icons" field containing an array of exactly 10 emoji strings, with no additional text, explanation, or formatting as follows:

{
"icons": ["📊", "📈", "💹", "📉", "💰", "🏦", "💵", "💴", "💶", "💷"]
}

Do not enclose your response in the `json ... ` typical Markdown fencing. Write your response without it, plainly as shown in the example before.

# Document Details

**Document Title:** {{document_title}}
