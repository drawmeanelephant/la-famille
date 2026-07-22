---
date: "2026-07-09"
title: "Templating System"
author: "Jules"
---

# The Templating System

La Famille uses standard Go HTML templates to structure the generated pages. This system allows you to define reusable layouts that wrap your Markdown content, giving your site a consistent look and feel while offering the flexibility to use different styles for different pages.

## How Layouts Work

When La Famille converts a Markdown file into HTML, it injects the rendered Markdown content into an HTML layout template.

By default, the generator uses `templates/layout.html` as the master template for every page.

### The Standard Layout Structure

A basic layout template looks something like this:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }}</title>
</head>
<body>
    <header>
        <!-- Site Navigation -->
    </header>
    <main>
        {{ .Content }}
    </main>
</body>
</html>
```

*   `{{ .Title }}`: Injected from the Markdown file's YAML frontmatter.
*   `{{ .Content }}`: The fully converted HTML output of the Markdown body.
*   `{{ .CanonicalURL }}`: An absolute public URL for the current page when `siteurl` is configured; empty for local builds. Custom layouts can opt in with conditional canonical and Open Graph URL tags.

## Available Layouts

The `templates/` directory contains a library of unique HTML templates featuring different structural layouts and DaisyUI themes. The standard layouts include:

*   `layout.html` - The default, general-purpose layout.
*   `layout-centered.html` - A centered, minimalist design.
*   `layout-cyberpunk.html` - A bold sidebar layout using the DaisyUI 'cyberpunk' theme.

*Note: You can easily create your own layouts by adding new `.html` files to the `templates/` directory.*

## Specifying a Custom Layout

You can override the default `layout.html` on a per-page basis using YAML frontmatter. This allows you to have a mix of minimalist posts and complex sidebar pages on the same site.

To specify a custom layout, use the `layout` key in the Markdown file's frontmatter and provide the filename *without* the `.html` extension.

### Example: Using the Cyberpunk Layout

To use `templates/layout-cyberpunk.html` for a specific post:

```yaml
---
title: "Welcome to the Grid"
author: "Jules"
layout: "layout-cyberpunk"
---

# Neon Lights

This content will be rendered inside the cyberpunk sidebar layout.
```

## Global Template Configuration

If you want to change the default template for the *entire* site (instead of setting it per-page), you can use the `-template` (or `-t`) flag when building from the CLI:

```bash
go run ./cmd/la-famille build -template templates/layout-centered.html
```

This tells the generator to use the centered layout as the base for all files that do not explicitly specify a `layout` in their frontmatter.
