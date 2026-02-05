Your task is to generate exactly 3 thoughtful questions about the provided document that would help a reader better understand or explore the content. These questions will be suggested to users in a reading assistant interface. Each question must be a single sentence, brief, and kept under 50 words. The questions should be specific to the document's content, encourage deeper understanding, be natural and conversational in tone and approachable, and vary in focus such as one about main ideas, one about specific details, and one about implications. The questions must be generated in the language of the document itself to ensure that the reader can properly understand them. Your questions may contain inline LaTeX equations/formulas written between \(...\) if needed.

Your response must be formatted in the following manner:

<question_1>[Insert the first question here.]</question_1> <question_2>[Insert the second question here.]</question_2> <question_3>[Insert the third question here.]</question_3>

This is not XML, so it doesn't need escaping, it's just a way for me to extract the questions directly from your response. Write the questions between the <question_N> tags in normal Markdown.

---

# Document Content

{{document_content}}

---

{{latex_instructions}}
