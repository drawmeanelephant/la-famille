---
title: "Routine Report: Template Naming Consistency Pass"
date: "2026-06-19"
author: "Jules"
routine: "Nightly Maintenance Pass"
success: "Yes"
status: "Completed"
---

# Nightly Maintenance Pass: Template Consistency

**Date:** 2026-06-19

## Actions Taken
- Renamed `brutalist.html`, `cyberpunk.html`, `devlog.html`, and `luxury_magazine.html` in `templates/` to adhere to the `layout-[name].html` convention.
- Renamed corresponding showcase files in `content/showcase/`.
- Updated frontmatter and internal links across all markdown files in `content/` to reflect the new template names.
- Fixed two minor HTML id bugs in `layout-brutalist.html` and `layout-luxury-magazine.html` that caused testing harness failures due to dangling anchor links.

## Learnings
Ensuring adherence to standardized naming conventions across all templates makes discovering and debugging layouts much more straightforward. Resolving dangling navigation links helped stabilize regression testing.
