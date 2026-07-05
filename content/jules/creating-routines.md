---
title: Creating Effective Routines for Jules
author: "The Human"
date: 2026-06-19
---

# Creating Effective Routines

This document is a guide for the human operator on how to write effective markdown routines for Jules, the AI maintainer of *La Famille*.

## The Golden Rule: No Dev Theatre
Avoid performative prompt engineering. Jules does not need to be told "You are a Senior UX Expert" or "DO NOT MAKE MISTAKES." These constraints handcuff the system and create unnecessary friction.

**Instead of this (Dev Theatre):**
> Role: You are an expert UI/UX developer and web designer specializing in Tailwind CSS, DaisyUI.
> Task: Create a complete, single-file HTML layout... Output ONLY the raw HTML code... Do not include markdown formatting blocks...

**Write this (Direct Action):**
> Task: Create a new `.html` file in `templates/` with a unique name. It should feature a unique layout and a DaisyUI theme. Test it, and commit it.

## Key Principles for Writing Routines

1.  **Assume Project Context:** Jules already knows the core project rules via internal memory. You **do not** need to restate:
    *   The requirement for the "Built in Go and Jules..." footer.
    *   Accessibility rules (like `focus-visible` or `aria-hidden`).
    *   The use of Tailwind CSS and DaisyUI.
    *   Security practices (`html.EscapeString`, `filepath.IsLocal`).
    *   The project structure (`cmd/`, `internal/`, `pkg/`, `templates/`, `content/`).
2.  **Focus on the "What," Not the "How":** Specify the desired outcome, the input files, and the expected output location. Trust Jules to handle the mechanics of file creation, bash execution, and Git commits.
3.  **Define the Trigger:** A routine should be triggered by simply asking Jules to "run the routine in `content/jules/your-routine-name.md`."
4.  **Expect Direct File Modification:** Do not ask for chat output or raw code blocks. Routines should instruct Jules to modify the codebase directly.

## Structure of a Routine File

A good routine file should contain:
*   **Goal:** A clear, one-sentence description of the task.
*   **Inputs:** What files or variables need to be considered.
*   **Outputs:** Where the result should be saved.
*   **Specifics:** Any unique constraints for *this specific task* (e.g., "Use a dark theme," or "Ensure it supports a sidebar").
