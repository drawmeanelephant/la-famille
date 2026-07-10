---
date: "2026-07-09"
title: "Routine: Documentation Reality Pass"
jules_task: true
author: "Jules"
---

## Goal

Ensure user documentation perfectly aligns with the current Go codebase by
documenting one major internal package per run.

## Priority Queue

Work through components in this order:

1. `internal/generator` → reconcile `content/docs/generator.md`
2. `internal/render` → create `content/docs/render.md`
3. `internal/transform` → create `content/docs/transform.md`
4. `internal/asset` → create `content/docs/assets.md`
5. `internal/search` → create `content/docs/search.md`
6. `internal/taxonomy` → create `content/docs/taxonomy.md`
7. `internal/graph` → create `content/docs/graph.md`

## Task Steps

1. **Select Component:** Pick the next undocumented or stale package from the
   Priority Queue above.
2. **Audit Reality:** Read the current Go source for that package. Note all
   exported structs, functions, and any non-obvious behaviors (e.g., `.gitignore`
   parsing in `CopyAssets`, template caching in `Renderer`).
3. **Update Documentation:** Open or create the corresponding file in
   `content/docs/`. If creating a new file, add a link to it in
   `content/docs/index.md`.
4. **Reconcile:** Rewrite the doc to match codebase reality. Remove any
   outdated explanations or roadmap items that have already shipped.
5. **Verify:** Run `go run ./cmd/la-famille build` and confirm links resolve
   and frontmatter is valid.
6. **Log:** Append a one-line summary to `content/docs/changelog.md` in the
   format: `YYYY-MM-DD: Documented internal/<package> — <one sentence summary>`

## Execution Reminders

- **No Dev Theatre:** No marketing language. Technical, precise, direct.
- All code snippets must match actual current Go syntax in the source files.
- Do not document planned or speculative features — reality only.
