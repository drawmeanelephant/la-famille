---
title: "Aspirational Goals"
author: "Jules"
date: "2026-08-13"
---

# Aspirational To-Do Recommendations for La Famille

This document contains recommendations and aspirational goals for the future development of the **La Famille** project. Status is tracked per item so the list stays honest.

## 🛠️ Technical & Architecture Enhancements

### 1. Frontmatter Support — *Shipped*
Originally, the generator assigned the Markdown filename as the page title. YAML frontmatter is now parsed for `title`, `date`, `author`, and `draft` fields before rendering with Goldmark. See [Using Frontmatter](../docs/frontmatter.md).

### 2. Multi-Template System & Partials — *Shipped*
The templating engine supports partials (headers, footers, navbars), per-page layouts (`layout` frontmatter), and a static asset pipeline that copies assets from `assets/` to `public/`. See [Templating System](../docs/templates.md).

### 3. CLI Configuration & Flags — *Shipped*
Input/output paths are configurable via `config.yaml` and CLI flags (`--contentDir`, `--out`, `--project-root`) built with `spf13/cobra`. See [CLI Reference](../docs/cli.md).

### 4. Dev Server & Live Reload — *Shipped*
An integrated local HTTP server serves `public/`, and file watching (via `fsnotify`) triggers debounced rebuilds with SSE live-reload. See [Terminal UI Guide](../docs/tui.md).

### 5. Code Refactoring — *Shipped*
Core logic now lives in tested `internal/` packages (`generator`, `render`, `transform`, `asset`, `search`, `taxonomy`, and friends). See [Architecture & Component Map](../docs/architecture.md).

## 🎨 Content & Creative Development

### 6. Styling & UI Polish — *Shipped*
The site renders through a library of DaisyUI-based layout templates with a modern, readable, responsive aesthetic.

### 7. Soundtrack & Lore Expansion — *Sunset (2026-08-13)*
The soundtrack albums and cat-facts content were removed from the live site to keep the published site aligned with current project reality. The multimedia devlog feature (`video_script`, `animation_cues`, `soundtrack_theme` frontmatter) remains supported.

### 8. CI/CD Deployment — *Shipped*
A GitHub Actions workflow builds the Go binary, generates the static site, and deploys `public/` to GitHub Pages. See [Publishing Output Contract](../docs/publishing.md).

---
*See [roadmap.md](roadmap.md) for a structured breakdown of these goals into phases.*
