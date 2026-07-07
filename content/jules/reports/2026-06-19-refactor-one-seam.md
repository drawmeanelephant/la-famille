---
title: Routine Report - Refactor One Seam
date: 2026-06-19
author: "Jules"
---

# Routine: Refactor One Seam - Markdown Engine Extraction

**Status:** Success

**Task Details:**
1. **Identify One Seam:** The core `Build` pipeline in `internal/generator/generator.go` contained the inline configuration and instantiation of the `goldmark` markdown engine, increasing coupling and length inside the core generator loop.
2. **Refactor Conservatively:** The engine instantiation logic was extracted into a new package `internal/markdown` in `internal/markdown/markdown.go`.
3. **Preserve Behavior:** The output static site behavior remains identical, as the exact same configuration was moved into `markdown.NewEngine(transformer)`.
4. **Strengthen Coverage:** A new unit test `internal/markdown/markdown_test.go` was added to verify the `NewEngine` returns a properly functional `goldmark.Markdown` instance.
5. **Record Learnings:** Extracting the markdown configuration reduces the cognitive load of `internal/generator/generator.go` and makes testing markdown conversion logic more isolated.

**Learnings & Suggestions:**
* `content/jules/refactor-one-seam.md` listed `linkTransformer` and HTML rendering as next steps, but those had already been completed by previous runs. We extracted `markdown` instead. The routine document should be updated periodically to point to valid seams, such as moving the web server (`serveCmd` logic) out of `main.go`.
