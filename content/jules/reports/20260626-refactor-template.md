---
Title: Routine - Template Refactoring
Author: Jules
Date: 2026-06-26
---

# Routine: Template Refactoring

**Status:** Success

**Details:**
- Modified `internal/render/render.go` to automatically parse files in a `partials` subdirectory alongside the main template in a single `template.ParseFiles` call.
- Abstracted the common footer markup into `templates/partials/footer.html`.
- Refactored `templates/layout.html` to use the new `footer.html` partial, demonstrating the successful extraction of repeated HTML patterns.