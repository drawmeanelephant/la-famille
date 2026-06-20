---
Title: Routine - Generate Cat Facts
Author: The Human
Date: 2026-06-20
---

# Routine: Generate Cat Facts

**Goal:** Generate a document containing 5 interesting facts about cats and create an accompanying soundtrack prompt to document the workflow pipeline.

## Task Details
1. **Generate Content:** Use your internal knowledge to generate five unique, interesting facts about cats. Do not use external APIs or scripts.
2. **Create Markdown File:** Create a new markdown file in `content/catfacts/`.
   - Name the file using the format: `<unix-epoch>-catfact.md` (e.g., `1718899200-catfact.md`).
   - Include YAML frontmatter with `Title` (a descriptive title under 60 characters), `Author` (@jules), and `Date` (YYYY-MM-DD format).
   - Write the 5 cat facts in the body of the markdown file.
3. **Generate Soundtrack Prompt:** Create a new song prompt file in `content/soundtrack/routine-tasks/vol-1/`.
   - Name the file identically to the cat facts file: `<unix-epoch>-catfact.md`.
   - The content should be a music prompt for Flow Music reflecting the completion of the cat facts routine, acknowledging the pipeline and documentation process.
4. **Log the Run:** Write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).
5. **Create a Report:** Write a short markdown report in `content/jules/reports/` (e.g., `[date]-cat-facts-routine.md`) summarizing the run.

## Execution Reminders
* Ensure the target directory `content/catfacts/` exists before attempting to write files to it.
* Commit the generated files as part of the routine execution.
