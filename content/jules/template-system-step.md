---
title: Routine - Template System Step
author: "The Human"
date: "2026-06-19"
---

# Routine: Template System Step

**Goal:** Improve the template architecture by adding one reusable capability to the rendering system.

## Task Details
1. **Choose One Capability:** Implement one narrow templating enhancement.
   - Good candidates: partials, section-specific layouts, per-page layout selection, shared header/footer includes, or template fallback rules.
2. **Integrate Cleanly:** The change should reduce duplication or increase layout flexibility.
3. **Preserve Existing Templates:** Existing templates should continue to work unless there is a documented migration reason.
4. **Verify Visually:** Build the site and confirm the new behavior works in rendered output.

## Execution Reminders
* **DO NOT** attempt to modify `.go` files or the core application logic (the system) when executing this task. This routine is meant to be accomplished using the existing template rendering capabilities.
* Do not redesign the whole frontend in the same task.
* Prefer reusable structure over one-off special cases.
* Keep the feature obvious enough to document later.
* **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**
