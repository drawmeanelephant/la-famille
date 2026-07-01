---
Title: Routine - Docs Reality Pass Update
Author: Jules
Date: 2026-06-29
---

# Routine: Docs Reality Pass

**Goal:** Reconcile the user documentation in `content/docs/` (and the README) with the actual shipped behavior of the codebase. Ensure that as features are added (like TUI, RAG, CLI), the documentation expands to cover them thoroughly.

## Task Details
1. **Find Missing Polish:** Identified that the `README.md` was missing explicit instructions for TUI navigation and operation, making onboarding harder for beginners. Also noted a lack of diverse feature-rich content to test the layout templates.
2. **Correct the Record:**
    - Updated `README.md` with a new `TUI Navigation & Controls` section, explicitly detailing the `up/down` and `j/k` keys for navigation, `Enter`/`Space` for execution, and `q`/`Esc` to drop back to the main menu. Also explained the "Serve Site" view behavior. Corrected the TUI launch command to `go run ./cmd/la-famille tui`.
    - Created a new test content file `content/showcase/raouls-multi-persona.md` designed to act as a "Multi-Persona Showcase". This file heavily leverages `goldmark` parsing with Prestige Tables, inline relative linking, blockquotes, and structural Go code snippets to stretch formatting features. It was placed under `content/showcase/` to not clutter standard docs.
3. **Index Updates:** The new showcase page is naturally indexed inside the showcase folder.
4. **Preserve Forward Motion:** N/A (No roadmap items were changed)

## Execution Reminders
* Successful build and test validation of the added markdown components via `go run ./cmd/la-famille build`.

**Status:** Success
