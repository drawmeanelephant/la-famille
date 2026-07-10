---
date: "2026-07-09"
title: "Raw Markdown & Unrendered Assets"
author: "Jules"
render: false
---

# Raw Markdown & Unrendered Assets

La Famille allows you to bypass the HTML rendering pipeline for specific markdown files. This is accomplished using the `render: false` flag in the YAML frontmatter.

## Why use `render: false`?

By default, the generator converts all `.md` files into `.html` files wrapped in your chosen layout template. However, you might want to serve raw markdown directly for:

*   Downloadable configuration files.
*   Documentation intended to be read in raw text format.
*   Assets that don't need UI wrapping.

## How it works

When a file has `render: false`:

1.  **Copied Verbatim:** The file is copied to the `public/` (or specified output) directory exactly as it is, maintaining the `.md` extension.
2.  **Link Preservation:** Any internal links pointing to this file from other pages will intelligently remain as `.md` links rather than being rewritten to `.html`. This ensures users can click the link and retrieve the raw file.

## Example

If you have this frontmatter in `my-config.md`:

```yaml
---
title: "My Config"
render: false
---
```

It will be output as `my-config.md` instead of `my-config.html`. Links pointing to it `[My Config](my-config.md)` will not be transformed.

[Go back to Index](index.md)
