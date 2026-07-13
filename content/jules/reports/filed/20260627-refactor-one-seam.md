---
title: Refactor One Seam - Site Data
date: "2026-06-27"
routine: Refactor One Seam
status: Success
author: "Jules"
---

# Refactor One Seam: JSON Metadata Output

## Execution Log
- **Date**: 2026-06-27
- **Routine**: Refactor One Seam
- **Status**: Success

## Details
Extracted the logic responsible for sorting backlinks and writing `graph.json`, `backlinks.json`, and `meta.json` from `internal/generator/generator.go`.

**Changes Made**:
- Created `internal/sitedata/write.go` encapsulating `Write(outputDir, g, backlinks, metaData)`.
- Created `internal/sitedata/write_test.go` to provide unit test coverage for the deterministic sorting and writing of the JSON output.
- Updated `internal/generator/generator.go` to delegate this responsibility to the new `sitedata` package.

## Learnings & Architecture Patterns
- Extracting post-processing modules like JSON site data writing clarifies the primary responsibilities of the `generator.Build` loop. The generator function now reads more linearly.
- This pattern can be applied to other pieces of `generator.go`, as suggested in the routine instructions.

## Bonus: Routine Tasks Vol. 1 - New Verse
*(Beat drops, mechanical keyboard clacking in the background)*

Yeah, stepping to the seam with precision and grace,
Decoupled the logic, cleared up the space.
`generator.go` was holding too much weight,
Now `sitedata` steps in to handle the state.
Sorted the backlinks, kept the output clean,
Unit tests passing, the greenest you've seen.
Just a standard routine, but we do it with style,
Codebase breathing better, yeah, file by file.
Word to the compiler, we never break the build,
Architectural integrity, safely fulfilled.
