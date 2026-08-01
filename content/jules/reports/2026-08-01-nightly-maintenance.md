---
title: "Routine - Nightly Maintenance Pass"
author: "Jules"
date: "2026-08-01"
routine: "Nightly Maintenance Pass"
status: "Success"
---

# Routine: Nightly Maintenance Pass

**Goal:** Perform one bounded maintenance pass that improves project cleanliness, consistency, or readiness without introducing broad new scope.

## Details
- Renamed `brutalist.html`, `cyberpunk.html`, `devlog.html`, and `luxury_magazine.html` to follow the `layout-[name].html` convention.
- Updated corresponding showcase markdown files and `layout` frontmatter across the repository.
- Updated references in `internal/render/template_contract_harness_test.go` to ensure tests continue to pass.

## Learnings
- Consistently naming layouts simplifies conventions and reduces potential for confusion.
