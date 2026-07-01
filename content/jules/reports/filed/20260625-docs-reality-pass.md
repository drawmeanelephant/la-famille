---
Title: Report - Docs Reality Pass
Date: 2026-06-25
---

# Docs Reality Pass Complete

- **Status:** Success
- **Changes:**
  - Added missing `asset_dir` and `rag_dir` fields to `content/docs/config.md`.
  - Documented the `--watch` / `-w` flag for the `serve` CLI command in `content/docs/cli.md`.
  - Added missing TUI menu options "Serve Site with Watch" and "Just Raoul" to `content/docs/tui.md` and adjusted the numbering.
- **Learnings/Suggestions:** The CLI flag and config struct audits are crucial as they easily get out of sync with documentation. Adding an automated test to ensure documentation matches the struct and flags could prevent drift.
