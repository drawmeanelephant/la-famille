---
title: "Routine - Search Step"
author: "The Human"
date: "2026-06-19"
---

# Routine: Search Step

**Goal:** Add one small discovery improvement that moves the project toward lightweight site search.

## Task Details
1. **Choose One Step:** Examples include enriching `meta.json`, generating a simple search index, or adding a minimal search UI shell.
2. **Stay Lightweight:** Prefer static and client-side approaches over server-side complexity.
3. **Build on Existing Outputs:** Reuse current metadata whenever possible.
4. **Verify Practical Use:** Confirm the step improves discoverability, not just internal structure.

## Execution Reminders
* Reusing the existing `meta.json` proved to be highly efficient.
* For future routines, consider extending the metadata structure in `meta.json` to include short summaries or tags for even richer search context.
* Avoid large dependencies unless clearly justified.
* Keep the feature optional and noninvasive.
* Write the simplest thing that could later grow into search.
* **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**
