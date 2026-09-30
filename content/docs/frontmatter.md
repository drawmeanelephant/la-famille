---
title: "Frontmatter Guide"
author: "Jules"
date: "2026-06-18"
---

# Using YAML Frontmatter

La Famille supports optional YAML frontmatter at the top of your `.md` files.

## Supported Fields

Here are the currently supported fields:

* `title`: The title of the page. If omitted, it falls back to the filename.
* `author`: The author of the post.
* `date`: A date string formatted as `YYYY-MM-DD` (e.g., `date: "2023-10-27"`).
* `tags`: An array of strings grouping the page under tag archives
  (e.g., `tags: [go, test]`). Each tag generates a `/tags/<tag>/` archive page,
  and `/tags/` lists every tag. Body hashtags such as `#ceramics` join the same
  tags; hashtags in headings, code, links, and URLs are ignored.
* `categories`: An array of strings grouping the page under category archives
  (e.g., `categories: [blog]`), generating `/categories/` pages the same way.
* `render`: A boolean (`true` or `false`).
* `slug`: A custom URL path for the page.
* `layout`: To specify a custom layout, provide the filename *without* the
  `.html` extension (e.g., `layout: "layout-brutalist"`).

```yaml
---
title: "Hello"
date: "2026-08-27"
tags: [go, test]
categories: [blog]
---
# Hello
```

Pages using `tags:`, `categories:`, or body hashtags link their terms to the
archives, and the site navigation gains **Tags** / **Categories** links
automatically, so every archive stays reachable without editing a template.
For example, a prose sentence can contain `#ceramics`, or a body can have a
standalone `#ceramics` line. Use no space after `#` so the line is not a
Markdown heading. `la-famille new --tags a,b` writes the same frontmatter from
the command line.

Taxonomy terms can use native-language letters and digits, for example
`tags: [起始]` and `categories: [说明]` generate `/tags/起始/` and
`/categories/说明/`. Public links percent-encode those path segments.
Spaces and most punctuation are removed, but hyphens are kept; uppercase
letters are lowercased.
`check` reports an error when a term would disappear altogether or its
normalized path component would exceed 255 bytes.

### The `render` Flag

If you set `render: false` in the frontmatter, La Famille will *not* convert the file to HTML. Instead, it will simply copy the raw `.md` file directly to the `public/` folder. This is useful for exposing raw assets or documentation you want visitors to download rather than view.

```yaml
---
title: "Secret Config"
render: false
---
# This will stay as Markdown!
```

This ensures we have maximum flexibility with how our content is processed.
