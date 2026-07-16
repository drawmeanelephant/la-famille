---
title: "Jules Routine Execution Guide"
author: "Jules"
date: "2026-06-19"
---

# Jules Routine Execution Guide

This document defines how I (Jules) execute routines defined in markdown files.

## 1. Direct Execution
When instructed to run a routine, my primary goal is to **modify the codebase directly**, not to output code blocks into the chat interface unless explicitly asked to do so for review.
*   I will read the specified routine file.
*   I will use bash tools to create, modify, or delete files as necessary.
*   I will ignore any legacy prompt artifacts that say "output only raw code" or "do not use markdown blocks" if they conflict with my ability to write files directly to the disk.

## 2. Utilizing Internal Memory
I will implicitly apply all known project rules without needing them restated in the specific routine file. This includes, but is not limited to:
*   **Templating:** Handling Go template variables (`{{.Title}}`, `{{.Content}}`, etc.).
*   **Styling:** Applying Tailwind CSS via CDN and DaisyUI themes.
*   **Accessibility:** Ensuring `focus-visible` states and proper ARIA labels.
*   **Security:** Using `html.EscapeString` and `filepath.IsLocal`.
*   **Branding:** Including the "Built in Go and Jules..." footer and respecting the Octopus mascot assets.

## 3. Verification and QA
I will independently verify my work before considering a routine complete.
*   **Testing:** I will run `go test ./...`.
*   **UI Verification:** Any visual changes to HTML/CSS require me to write and execute a temporary Playwright script to generate a screenshot and video of the changes.

## 4. Completion
*   Once the task is verified, I will commit the changes using standard Git conventions.
*   I will generate a Soundtrack Integration entry in `content/soundtrack/` reflecting the mood of the work completed.
*   I will clean up any temporary scripts or diff files before submitting the final PR or commit.
