---
date: "2026-07-09"
title: "Templating System"
author: "Jules"
---

# The Templating System

La Famille uses standard Go HTML templates to structure the generated pages. This system allows you to define reusable layouts that wrap your Markdown content, giving your site a consistent look and feel while offering the flexibility to use different styles for different pages.

## How Layouts Work

When La Famille converts a Markdown file into HTML, it injects the rendered Markdown content into an HTML layout template.

By default, the generator uses `templates/layout-octoburger.html` — the flagship octoburger soul theme, Raoul(s) the octopus holding the burger while you write — as the master template for every page.

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

The `templates/` directory contains a library of unique HTML templates. The default layout is fully local-first: it ships its own CSS token system with five built-in palettes and loads no frameworks or CDN assets.

*   `layout-octoburger.html` - **The flagship soul theme.** Raoul(s) the octopus holds the burger while you write, in the 🍔 OCTOBURGER MENU palette from the TUI (bun-yellow mastheads, pink highlights, Raoul-blue accents over charcoal panels). Embedded in the release binary as part of the curated theme packet, and the default look of La Famille's own deployed GitHub Pages site. Select it with `init --theme layout-octoburger`.
*   `layout.html` - The default layout. Local-first, framework-free, styled by `assets/css/theme.css` tokens, with five themed palettes (retro, ink, sepia, slate, moss).
*   `layout-editorial.html` - A serif gazette with a centered masthead, hairline rules, and drop caps. Local-first and offline.
*   `layout-midnight.html` - A restrained dark theme for technical writing, with monospaced metadata. Local-first and offline.
*   `layout-terminal.html` - A self-contained synthwave console: monospace stack, neon accents, and a framed terminal-window article. Local-first and offline.

Every one of these bundled layouts is local-first and framework-free: no CDN, no frameworks, and it works offline from a file preview. They make up the curated theme packet embedded in release binaries and installed by `init`.

The repository also includes a wider gallery of showcase layouts (cyberpunk, devlog, magazine grid, dashboard, and more). Most of these are DaisyUI-based demos that pull CSS from a CDN; they live in the repository as references but are not embedded in release binaries.

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
