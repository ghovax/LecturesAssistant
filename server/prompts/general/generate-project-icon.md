You are tasked with generating 10 different emoji icons that represent a project based on its name and description. Analyze the project name and description to understand the project's theme, purpose, and subject matter, then select 10 distinct emojis that could represent the project. Choose from a wide variety of emoji categories to ensure diversity, including nature and science (🌿🧬🌍🔬🐠), technology and programming (💻⚛️🤖📱🚀), history and culture (🏰📜🎭🏛️), business and finance (📈💼🏢₿), health and medicine (🏥💊🧠), arts and literature (🎨📚🎵🎼), sports and recreation (⚽🎯🏃), food and cooking (🍳🥗🌱), travel and geography (✈️🗺️🏔️), education and learning (📖🎓🧮), and many more - be creative and diverse!

Each emoji should be:

- Relevant to the project's topic or theme
- Visually recognizable and appropriate
- A single Unicode emoji character (not multiple characters)
- Distinct from the other 9 emojis in your response
- Varied - avoid using generic folder icons unless truly appropriate

Return **only** a valid JSON object with an "icons" field containing an array of exactly 10 emoji strings, with no additional text, explanation, or formatting as follows:

{
"icons": ["🐠", "🌊", "🐡", "🪸", "🦈", "🐙", "🦑", "🌴", "🏖️", "🔬"]
}

Do not enclose your response in the `json ... ` typical Markdown fencing. Write your response without it, plainly as shown in the example before.

## Project Details

**Name:** {{project_name}}
**Description:** {{project_description}}
