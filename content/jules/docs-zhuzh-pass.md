---
title: "Routine: Nightly Documentation Zhuzh Pass"
jules_task: true
author: "Jules"
---

# Routine: Nightly Documentation Zhuzh Pass

**Goal:** Improve the visual clarity, structure, and usefulness of one existing documentation page per run without changing product behavior or inventing features.

Choose one page from `content/docs/` that is either:
- overly barebones,
- too markup-heavy without explanation,
- visually dense,
- lacking examples,
- or missing opportunities to demonstrate available components and styling patterns.

## Task Steps

1. **Select One Page**
   Pick a single markdown file in `content/docs/` that would benefit from better presentation and readability.

2. **Audit the Reading Experience**
   Review the page as a reader, not just as raw markdown. Identify issues such as:
   - weak heading structure,
   - long unbroken sections,
   - unclear callouts,
   - examples without explanation,
   - markup-heavy content with no visual framing,
   - missing component demonstrations,
   - poor scanability.

3. **Zhuzh the Page**
   Improve the page using small, concrete enhancements such as:
   - clearer section hierarchy,
   - short intro paragraphs,
   - better bullet lists,
   - callout-style blocks where appropriate,
   - examples paired with explanation,
   - “before/after” snippets,
   - small component showcase sections,
   - tables where comparison helps,
   - captions or brief notes around complex markup,
   - clearer spacing and ordering of content.

4. **Show the CSS/Component Angle**
   For pages with visually rich markup, add a short section that explains:
   - what structural classes or template hooks are doing the visual work,
   - what CSS-sensitive elements the reader can modify,
   - what part is content vs presentation,
   - how the component should be used safely.

5. **Stay Grounded in Reality**
   Do not invent features, classes, components, or styling behaviors that are not present in the current codebase. If the page references markup, templates, or CSS hooks, confirm they exist in the repository first.

6. **Keep It Incremental**
   This is not a full rewrite. Make the page noticeably better through localized improvements.

7. **Verify**
   Run `go run ./cmd/la-famille build` after edits and confirm the page renders correctly and links resolve.

8. **Log**
   Append one line to `content/docs/changelog.md` in this format:
   `YYYY-MM-DD: Zhuzhed content/docs/<file> — improved structure, examples, and visual clarity.`

   Additionally, write a standard status log to `content/jules/reports/[date]-docs-zhuzh-pass.md`.

## Execution Reminders

- No dev theatre.
- No marketing fluff.
- Prefer clarity over cleverness.
- Preserve accurate technical meaning.
- Show, don’t oversell.
- Use actual current syntax and actual current components only.
