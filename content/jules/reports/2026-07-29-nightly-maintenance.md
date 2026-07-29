---
title: "Routine - Nightly Maintenance Pass"
author: "Jules"
date: "2026-07-29"
routine: "nightly-maintenance"
status: "success"
---

# Routine: Nightly Maintenance Pass

**Goal:** Perform one bounded maintenance pass that improves project cleanliness, consistency, or readiness without introducing broad new scope.

## Theme: Template Naming Consistency

## Execution Result
- Renamed 4 inconsistent template files to strictly follow the `layout-[name].html` naming convention:
  - `brutalist.html` -> `layout-brutalist.html`
  - `cyberpunk.html` -> `layout-cyberpunk.html`
  - `devlog.html` -> `layout-devlog.html`
  - `luxury_magazine.html` -> `layout-luxury-magazine.html`
- Updated frontmatter references across 6 markdown content files and documentation to ensure layouts map correctly.
- Updated `template_contract_harness_test.go` to test the new layout names instead of old ones.
- Discovered and fixed an invalid `layout: "report"` reference in `2026-07-27-architectural-review.md` that caused silent fallback warnings during build generation, standardizing it to `layout: "layout"`.

## Learnings
Ensuring all layout templates share a common prefix (e.g. `layout-`) makes them easier to discover and reduces confusion around partials versus full page layouts. Using the La Famille CLI output to check for fallback warnings is a great way to identify stale or invalid frontmatter configurations.
