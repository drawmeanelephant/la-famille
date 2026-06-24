---
Title: Routine - Docs Reality Pass
Author: The Human
Date: 2026-06-21
---

# Routine: Docs Reality Pass

**Goal:** Reconcile the user documentation in `content/docs/` with the actual shipped behavior of the codebase. Ensure that as features are added (like TUI, RAG, CLI), the documentation expands to cover them thoroughly.

## Task Details
1. **Find Missing Polish:** Identify one place in `content/docs/` where a feature is either entirely missing documentation or is documented in a barebones, unhelpful way (e.g., missing setup instructions, missing usage examples).
2. **Correct the Record:** Update or create the relevant markdown file in `content/docs/` so it thoroughly explains the feature, matching the current reality of the codebase. Include examples where appropriate.
3. **Index Updates:** If you create a new documentation file, ensure you add it to the list in `content/docs/index.md`.
4. **Preserve Forward Motion:** If you find a roadmap item or to-do list in the docs that is partially complete, rewrite it as a next-step item instead of deleting it entirely.

## Execution Reminders
*   **Do NOT rewrite `README.md` during this routine.** The `README.md` is strictly the GitHub landing page overview. This routine is explicitly for expanding and polishing the actual user guides inside `content/docs/`.
*   Focus on making the documentation useful for end-users, ensuring it covers things like CLI usage (`go run`), flags, and configuration.
*   Keep edits explicit and ensure they are written in Markdown.
*   **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**

* **Configuration Audits:** Ensure core files like `config.yaml` are well-documented and easily discoverable for users.
* **CLI Flag Audits:** Periodically audit `cmd/la-famille/main.go` flag declarations against the documentation in `cli.md`. Consider using an automated grep test to prevent drift.
