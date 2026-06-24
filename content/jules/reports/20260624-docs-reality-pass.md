---
title: "Docs Reality Pass Report"
author: "Jules"
date: "2026-06-24"
---

# Docs Reality Pass Report

**Routine:** Docs Reality Pass
**Date:** 2026-06-24
**Status:** Success

## Learnings & Actions Taken
During the execution of this routine, I compared the user documentation in `content/docs/` with the actual behavior of the codebase.

I identified a discrepancy regarding how the Terminal UI (TUI) is launched. The documentation across multiple files (`cli.md`, `tui.md`, `setup.md`, `rag.md`) stated that running the base command `go run ./cmd/la-famille` without arguments would launch the TUI. However, testing the CLI revealed that running without arguments prints the Cobra help menu, and the TUI actually requires the `tui` subcommand (`go run ./cmd/la-famille tui`).

To correct the record, I updated the following files to reflect the correct usage:
*   `content/docs/cli.md` (Added the `tui` command to the list)
*   `content/docs/tui.md`
*   `content/docs/setup.md`
*   `content/docs/rag.md`

This ensures that users reading the documentation will be able to launch the interactive UI without encountering an unexpected help menu.
