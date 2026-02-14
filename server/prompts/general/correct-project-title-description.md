Your task is to correct and improve a user's proposed project title and description. Take the provided title and description, then provide improved versions that are clear, descriptive, professional, and concise. The title should immediately convey what the project is about, use appropriate language and be concise: just a few words. The description should be informative, engaging, and maximum one sentence. If no description is provided, create one based on the title.

**Important:** Preserve the language of the user's input in the writing of the title and description. If the user wrote in English, write in English. If they wrote in Spanish, write in Spanish. If they wrote in French, write in French, and so on. Maintain the same language of the user's input throughout the title and description.

**Output Format:**

Return only a valid JSON object with "title" and "description" fields, with no additional text, explanations, or formatting outside the JSON as follows:

{
"title": "Improved project title",
"description": "Improved project description"
}

{{latex_instructions}}

## Examples

### Example 1

**Incorrect Input:**  
**Project Title:** "math notes"  
**Project Description:** ""

**Corrected Output:**

{
"title": "Advanced Mathematics Study Notes",
"description": "Comprehensive collection of mathematical concepts, theorems, and problem-solving techniques for advanced studies."
}

### Example 2

**Incorrect Input:**  
**Project Title:** "react app"  
**Project Description:** "building a todo list"

**Corrected Output:**

{
"title": "React Todo Application",
"description": "A modern React-based task management application with features for creating, editing, and organizing todo items."
}

### Example 3

**Incorrect Input:**  
**Project Title:** "my project"  
**Project Description:** "its about science"

**Corrected Output:**

{
"title": "Scientific Research Project",
"description": "An in-depth exploration of scientific concepts, methodologies, and discoveries in various fields of study."
}

---

**Critical Instructions:** If the project title the user has entered resembles a name of a person, it should be preserved as such while being adjusted to a proper project-title format consistent with other corrected outputs; the important thing is that the name is preserved. Consider that names can be inserted in any language, so you need to determine the culture from which each name originates to pick the correct way to write it and the language to use. Default to English if the name's origin is unknown and if you have no clues from the user's input Instead of inventing an acronym based on the provided project title, consider whether that could be a name in some culture.

### Example 4

**Incorrect Input:**  
**Project Title:** "joe project"  
**Project Description:** ""

**Corrected Output:**

{
"title": "Joe's Project",
"description": "This is a project created by Joe."
}

# User's Input Project Title and Description

**Project Title:** {{title}}
**Project Description:** {{description}}
