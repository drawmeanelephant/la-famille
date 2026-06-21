---
Title: Routine - Refactor One Seam
Author: The Human
Date: 2026-06-19
---

# Routine: Refactor One Seam

**Goal:** Improve the architecture by extracting one small, cohesive unit of logic into a more appropriate package without changing behavior.

## Task Details
1. **Identify One Seam:** Find one self-contained responsibility in the current codebase that can be cleanly extracted or reorganized.
   - Good candidates include metadata gathering, link transformation, stub generation, JSON output writing, or RAG export helpers.
2. **Refactor Conservatively:** Move or reshape only the selected seam. Do not combine this with unrelated rewrites.
3. **Preserve Behavior:** The generated output should remain functionally identical unless a bug is explicitly being fixed.
4. **Strengthen Coverage:** Add or update unit tests or fixture tests to lock in the intended behavior of the extracted seam.
5. **Record Learnings:** If the refactor reveals a structural pattern worth repeating, log it in `.jules/architecture.md`.

## Execution Reminders
* Next logical seams to extract from `main.go` would be `linkTransformer` (AST traversal and link rewriting) and the HTML rendering phase.
* Keep the change narrow and reversible.
* Prefer descriptive package boundaries over clever abstractions.
* Run `go test ./...` and `go vet ./...` before finishing.
* **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**
