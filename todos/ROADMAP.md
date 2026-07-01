# La Famille Hub Roadmap & TODO 🥖🐙

This document tracks active refactoring tickets, pipeline enhancements, and developer experience milestones for the La Famille project. Since target project directories are ephemeral and refreshed from upstream archives, this file is kept in the central workspace `todos/` directory to ensure persistence.

---

## 📋 Active & Upcoming Tasks

### Phase 1: Core Refactoring & Robustness (Short-Term)
- [x] **Extract Build Pipeline (`internal/generator`)**
  - [x] Extract the two-pass build loop from `cmd/la-famille/main.go`'s `run()` function.
  - [x] Create a new package `internal/generator`.
  - [x] Implement a clean, reusable `Build(cfg config.Config) error` interface.
  - [x] Enable the TUI, watch mode, and RAG exporter to import and invoke `generator.Build`.
- [x] **Normalize Frontmatter Key Casing**
  - [x] Implement a normalization pass in `internal/content` that lowercases frontmatter keys before unmarshaling.
  - [x] Standardize parsing so files using both uppercase keys (e.g., `Title:`, `Author:`) and lowercase keys (e.g., `title:`, `author:`) are parsed correctly.
  - [x] Add unit tests covering mixed-casing frontmatter scenarios to prevent regressions.
- [x] **Implement Config Validation on Load**
  - [x] Add a `Validate() error` method to the `Config` struct in `internal/config/config.go`.
  - [x] Validate that port numbers are in a valid range, paths (like `content_dir`) are non-empty strings, etc.
  - [x] Call `Validate()` immediately after `Load()` in `main.go` to surface early, actionable errors instead of downstream I/O failures.

### Phase 2: Asset & Exporter Pipeline Enhancements (Medium-Term)
- [x] **Add Verbatim Asset Copy Step**
  - [x] Walk the `assets/` directory (or a configured asset directory path) during the build.
  - [x] Copy non-Go-source files (logos, mascot images) verbatim to `public/assets/` during execution.
  - [x] Reference the existing `render: false` copy mechanism as a pattern.
  - [x] Verify templates can reference `/assets/img/jules-logo.png` without broken paths.
- [x] **Formalize `rag-archive/` Output Path**
  - [x] Move `ragexport.RunExport()` output from the hard-coded source directory to a configurable top-level directory (e.g., `rag-archive/` or defined via `config.yaml` as `rag_dir`).
  - [ ] Add further output formats (e.g. JSON RAG output).
  - [x] Add the default or configured RAG export output path to `.gitignore`.
  - [x] Clean up the source tree to ensure RAG exports do not generate git diffs inside `internal/`.

### Phase 3: Developer Experience & Authoring Loop (Long-Term)
- [x] **Implement `--watch` Flag for Local Rebuilds**
  - [x] Add a `--watch` flag to the `serve` command in `cmd/la-famille` or create a new `internal/watcher` package.
  - [x] Integrate `fsnotify` to monitor changes in `content/` and `templates/` directories.
  - [x] Trigger `generator.Build()` automatically upon file changes to enable a live-reload loop.
  - [x] Integrate the watch status display into the TUI's "Serve Site" screen.
