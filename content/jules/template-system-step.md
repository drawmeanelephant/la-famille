---
Title: Routine - Template System Step
Author: The Human
Date: 2026-06-19
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
* Do not redesign the whole frontend in the same task.
* Prefer reusable structure over one-off special cases.
* Keep the feature obvious enough to document later.
* **Upon successful completion, you MUST append a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to the "Run Log" section of `content/jules/index.md`.**
