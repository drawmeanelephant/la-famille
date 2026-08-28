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
  and `/tags/` lists every tag.
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

Pages using `tags:` or `categories:` link their terms to the archives, and the
site navigation gains **Tags** / **Categories** links automatically, so every
archive stays reachable without editing a template. `la-famille new --tags a,b`
writes the same frontmatter from the command line.

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
