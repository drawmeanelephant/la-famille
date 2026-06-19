---
title: "La Famille SSG: Roadmap"
author: "Jules"
date: "2026-06-18"
render: true
---

# Project Roadmap

This document outlines the proposed phases to evolve La Famille from a single-file script into a robust, extensible Go-based Static Site Generator (SSG).

La Famille currently possesses strong fundamentals: Markdown parsing via Goldmark, frontmatter extraction, link transformation, and graph/backlink generation. The next steps focus on refactoring, improving the developer/authoring experience, and modernizing the output.

## Phase 1: Foundation & Architecture
*Focus: Paying off technical debt and establishing a solid codebase for future features.*

- **Code Refactoring:** Break down the monolithic `main.go` file. Move the core rendering, walking, and graph-generation logic into an `internal/generator` package.
- **CLI Configuration Engine:** Replace the standard `flag` package with a robust framework like `cobra` and `viper` to support configuration files (e.g., `la-famille.yaml`) and advanced command-line flags.

## Phase 2: The Authoring Experience
*Focus: Making La Famille enjoyable and efficient for content creators.*

- **Dev Server & Live Reload:** Implement an integrated local HTTP server with file watching (via `fsnotify`). This will automatically trigger a rebuild and refresh the browser when a `.md` or `.html` file is saved.
- **Advanced Templating:** Expand the Go `html/template` integration to support partials (headers, footers) and specific page layouts (e.g., different layouts for `index.html` vs. individual posts).
- **Static Asset Pipeline:** Create a mechanism to seamlessly sync CSS, JavaScript, and image assets from an `assets/` directory directly into the generated `public/` directory during build time.

## Phase 3: Design, UI, and Output Polish
*Focus: Creating a stunning default aesthetic out of the box.*

- **Modern CSS Framework:** Introduce a default, modern Vanilla CSS styling system using CSS variables, responsive typography, and an aesthetic dark/light mode toggle.
- **Enhanced Stub Pages:** Improve the auto-generated "Missing Page" stubs with better UI, providing clear contextual links back to the referencing documents to aid in content mapping.

## Phase 4: Advanced Features & Deployment
*Focus: Scaling the project and making it production-ready.*

- **CI/CD Deployment:** Provide default GitHub Actions workflows (`.github/workflows/deploy.yml`) to automatically build and deploy the static site to GitHub Pages on every push.
- **Client-Side Search:** Leverage the existing `meta.json` output to implement a lightweight, fast, client-side search overlay (e.g., using `fuse.js` or a custom vanilla implementation).
- **Taxonomy & Tagging:** Add support for indexing tags and categories from YAML frontmatter, automatically generating index pages for each tag.
