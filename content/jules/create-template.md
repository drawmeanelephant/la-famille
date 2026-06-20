---
Title: Routine - Generate New Layout Template
Author: The Human
Date: 2026-06-19
---

# Routine: Generate New Layout Template

**Goal:** Create a new, unique HTML layout template for the static site generator.

## Task Details
Create a single-file HTML layout in the `templates/` directory.

1.  **Variety:** Choose a structural layout we don't currently use heavily (e.g., Sidebar Navigation, Centered Minimalist, Split-screen, Brutalist, Magazine Grid).
2.  **Theme:** Select a specific, matching DaisyUI theme (e.g., synthwave, cupcake, dracula, business, wireframe, cyberpunk) and apply it to the `<html>` tag.
3.  **Core Components:**
    *   Ensure there is a visually distinct header and footer.
    *   Ensure mobile-first responsiveness.
    *   Wrap the `{{.Content}}` output in a container utilizing Tailwind Typography's `prose` classes (e.g., `class="prose lg:prose-xl max-w-none"`).

## Execution Reminders
*   Write the file directly to the `templates/` directory with an appropriate name (e.g., `layout-sidebar.html`).
*   Rely on your internal memory for project standards (footers, accessibility, Go template variables).
*   Test visually with Playwright before committing.
*   **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**
