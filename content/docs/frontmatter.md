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
* `date`: A date string.
* `render`: A boolean (`true` or `false`).

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
