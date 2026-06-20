---
Title: Jules Routines Index
Author: Jules (AI)
Date: 2026-06-19
---

# Jules Routines

This directory contains the standard, executable routines that guide my (Jules) automated workflows. They define bounded, recurring tasks that improve the codebase incrementally.

## Available Routines

*   [Generate New Layout Template](create-template.md)
*   [Implement Micro-UX Improvement](micro-ux-improvement.md)
*   [Implement Security Enhancement](security-enhancement.md)
*   [Refactor One Seam](refactor-one-seam.md)
*   [Close One Stub](close-one-stub.md)
*   [Docs Reality Pass](docs-reality-pass.md)
*   [Asset Pipeline Step](asset-pipeline-step.md)
*   [Template System Step](template-system-step.md)
*   [Metadata Feature Step](meta-feature-step.md)
*   [Serve/Watch Step](serve-watch-step.md)
*   [Test Fixture Expansion](test-fixture-expansion.md)
*   [Improve Missing Page Stub](stub-page-polish.md)
*   [Taxonomy Step](taxonomy-step.md)
*   [Search Step](search-step.md)
*   [Nightly Maintenance Pass](nightly-maintenance.md)

### Meta Routines
*   [Self-Improvement Pass](routine-self-improvement-pass.md)

## Suggested Schedule Mix

For a healthy and consistent codebase evolution, try running these in this rotation:

*   **Nightly/Regular:**
    *   `refactor-one-seam.md`
    *   `docs-reality-pass.md`
    *   `test-fixture-expansion.md`
    *   `close-one-stub.md`
    *   `nightly-maintenance.md`
*   **Every Few Days:**
    *   `template-system-step.md`
    *   `asset-pipeline-step.md`
    *   `stub-page-polish.md`
    *   `routine-self-improvement-pass.md`
*   **Less Frequent but Strategic:**
    *   `serve-watch-step.md`
    *   `search-step.md`
    *   `taxonomy-step.md`
    *   `meta-feature-step.md`

---

## Run Log

*(Routines will automatically append their execution results, notes, and suggested improvements here upon completion. These logs will be periodically analyzed and cleared by the Self-Improvement Pass routine.)*

*   **2026-06-20** - `docs-reality-pass` - SUCCESS. Updated `README.md` to document the existing `init` and `rag` CLI commands, and aligned `content/meta/roadmap.md` to reflect that the CLI and RAG tasks are already complete. Also started generating Volume 2 for the routine tasks soundtrack. No major changes needed to the routine itself.
- **2026-06-19 - Refactor One Seam - Success**
  Extracted `RAG export` logic from `cmd/la-famille/main.go` into a dedicated `internal/ragexport` package. This keeps `main.go` cleaner and encapsulates the RAG bundling functionality.
