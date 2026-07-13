---
title: Routine - Refactor One Seam
author: "The Human"
date: "2026-06-19"
---

# Routine: Refactor One Seam

**Goal:** Improve the architecture by extracting one small, cohesive unit of logic into a more appropriate package without changing behavior.

## Task Details
1. **Identify One Seam:** Find one self-contained responsibility in the current codebase that can be cleanly extracted or reorganized.
   - Good candidates include metadata gathering, link transformation, stub generation, JSON output writing, or RAG export helpers.
2. **Refactor Conservatively:** Move or reshape only the selected seam. Do not combine this with unrelated rewrites.
3. **Preserve Behavior:** The generated output should remain functionally identical unless a bug is explicitly being fixed.
4. **Strengthen Coverage:** Add or update unit tests or fixture tests to lock in the intended behavior of the extracted seam.
5. **Record Learnings:** If the refactor reveals a structural pattern worth repeating, log it in `.julesarchitecture.md`.

## Execution Reminders
* Next logical seams to extract would be moving the web server (`serveCmd` logic) out of `main.go` into an `internal/server` package.
* Keep the change narrow and reversible.
* Prefer descriptive package boundaries over clever abstractions.
* Run `go test ./...` and `go vet ./...` before finishing.
* **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**

* **Package Abstraction:** When extracting logic from `main.go`, look for coupled structures (like `Graph` or `Page`). Extracting these structures into their own agnostic packages (e.g., `internal/page`) enables safer sharing of data and clears up the core generation pipeline. Post-processing modules (like stub generation) or entire phases (like graph building or HTML rendering) are excellent candidates for extraction.
