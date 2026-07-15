---
title: "Report - Refactor One Seam (Stub Generator)"
author: "Jules"
date: "2026-06-23"
---

# Routine Execution Log: Refactor One Seam

**Date:** 2026-06-23
**Routine:** Refactor One Seam
**Status:** Success

## Details
- Successfully extracted the stub generation logic from `cmd/la-famille/main.go` into a new `internal/stub` package.
- Relocated the `Page` struct to a new `internal/page` package to be shared between the main logic and the stub generator.
- Abstracted the stub creation logic into `stub.GenerateStubs(cfg, missingFiles, g, p)`.
- Moved `RelPathFromTo` to the `internal/stub` package and exported it for reuse and testing.
- Wrote tests for `RelPathFromTo` and `GenerateStubs` to strengthen coverage.
- Track 52, "Extract the Stub", was added to `content/soundtrack/routine_tasks_vol_2.md` to reflect the seam extraction.

## Learnings
- **Coupling of Page Struct:** The `Page` struct was originally deeply coupled to `main.go`. By extracting it into its own `internal/page` package, it acts as an agnostic data container for templates, enabling other packages like `internal/stub` to safely generate compliant HTML rendering structs.
- **Rethinking Stub Generation:** Stub generation relies heavily on the final state of the `graph.Graph` and missing file data, indicating it fits well as a post-processing module rather than an inline script in `main.go`. Extracting this clears up the core generator pipeline.
