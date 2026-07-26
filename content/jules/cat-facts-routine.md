---
title: "Routine - Generate Cat Facts"
author: "The Human"
date: "2026-06-20"
---

# Routine: Generate Cat Facts

**Goal:** Generate a document containing 5 interesting facts about cats and document the workflow pipeline.

## Task Details
1. **Generate Content:** Use your internal knowledge to generate five unique, interesting facts about cats. Do not use external APIs or scripts.
2. **Create Markdown File:** Create a new markdown file in `content/catfacts/`.
   - Name the file using the format: `<unix-epoch>-catfact.md` (e.g., `1718899200-catfact.md`).
   - Include YAML frontmatter with `Title` (a descriptive title under 60 characters), `Author` (@jules), and `Date` (YYYY-MM-DD format).
   - Write the 5 cat facts in the body of the markdown file.
3. **Log the Run:** Write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).
4. **Create a Report:** Write a short markdown report in `content/jules/reports/` (e.g., `[date]-cat-facts-routine.md`) summarizing the run.

## Execution Reminders
* Generating thematic facts (e.g., historical cats) adds great variety compared to standard biological trivia.
* Ensure the target directory `content/catfacts/` exists before attempting to write files to it.
* Commit the generated files as part of the routine execution.

* **Thematic Facts:** To add variety, consider requiring thematic cat facts (e.g., historical cats, biological quirks, famous internet cats) instead of just general facts.
* **Expansion Opportunity:** Consider expanding this routine to also generate an accompanying `assets/img` prompt for a feline illustration to complement the text.
