---
title: "Architecture Hardening and Pipeline Optimization"
date: "2026-07-12"
author: "Jules"
---
# Architecture Hardening and Pipeline Optimization

Successfully refactored the build generator to handle concurrent map assignments and collision safety.

## Key optimizations and changes
- **Output-path collision preflight:** Implemented `validateOutputPaths` early in `generator.go` Pass 2 to gracefully identify explicit slug-overwrites instead of relying on filesystem OS locks.
- **Error sorting determinism:** Changed the random map slice sorting from `err.Error()` matching to lexical index based on target page iteration to guarantee identical error output formats across multiple builds.
- **Watch-mode allocation optimization:** Extracted LiveReload string building from `bytes.Buffer` arrays and `strings.Replace` into an `io.Writer` chunk function `writeWithLiveReload` inside `render.go`, slashing massive whole-page string duplications across watch environments.
- **Asset ignore pruning:** Enforced early `filepath.SkipDir` logic on directories flagged in native ignored checks to skip completely redundant iteration patterns in `internal/asset/copy.go`.
- **URL Boundaries:** Realigned the `GetOutputURL` parser logic to honor `index.md` contract priority strictly before parsing slugs, which correctly restored routing mechanics for index subpages.

## Testing additions
- Added integration coverage to trap deterministic output errors.
- Handled parallel layout extraction via `TestRendererConcurrentLayouts`.
- Improved Link AST extraction cases across query strings, HTML entities, and URL-scheme skips.
