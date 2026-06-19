---
title: "Aspirational Goals"
author: "Jules"
date: "2026-06-18"
render: true
---

# Aspirational To-Do Recommendations for La Famille

This document contains recommendations and aspirational goals for the future development of the **La Famille** project.

## 🛠️ Technical & Architecture Enhancements

### 1. Frontmatter Support
Currently, the generator assigns the Markdown filename as the page title.
- **Action:** Integrate a YAML frontmatter parser to extract metadata like `Title`, `Date`, `Author`, and `Draft` status from the top of the `.md` files before rendering with Goldmark. *(Note: partially implemented)*

### 2. Multi-Template System & Partials
The generator currently only relies on a single `layout.html`.
- **Action:** Expand the templating engine to support partials (e.g., headers, footers, navbars) and specific page templates (e.g., `post.html` vs. `index.html`).
- **Action:** Implement a static asset pipeline to seamlessly copy CSS, JavaScript, and image files from an `assets/` folder to the `public/` directory.

### 3. CLI Configuration & Flags
The input/output paths are currently hardcoded in `cmd/la-famille/main.go`.
- **Action:** Implement command-line flags (via the standard `flag` package or a library like `cobra`/`viper`) so users can execute `go run ./cmd/la-famille/main.go --contentDir ./docs --out ./dist`.

### 4. Dev Server & Live Reload
To improve the authoring experience:
- **Action:** Add an integrated local HTTP server (`net/http`) to serve the `public/` directory.
- **Action:** Integrate file watching (e.g., `fsnotify`) to automatically trigger a rebuild when a `.md` or `.html` file is saved.

### 5. Code Refactoring (Following GEMINI.md)
The `GEMINI.md` file specifies `internal/` and `pkg/` directories, but core logic currently lives in `main.go`.
- **Action:** Refactor `processFile` and `run` into an `internal/generator/` package to improve modularity and testability.

## 🎨 Content & Creative Development

### 6. Styling & UI Polish
The generated output is currently raw HTML.
- **Action:** Create a CSS stylesheet (Vanilla CSS or a micro-framework) to give the generated site a modern, readable, and responsive aesthetic. Update `layout.html` to include it.

### 7. Soundtrack & Lore Expansion (In Progress)
The soundtrack files have been moved to the `content/` directory and expanded with new track listings (e.g., "go.mod (The Ledger)" and "Unit Test Blues").
- **Next Step:** Further expand the lore and lyrics within these files to bridge the engineering and musical narratives.

### 8. CI/CD Deployment
To make the project accessible to the world:
- **Action:** Set up a GitHub Actions workflow (`.github/workflows/deploy.yml`) to automatically build the Go binary, generate the static site, and deploy the `public/` folder to GitHub Pages.

---
*See [roadmap.md](roadmap.md) for a structured breakdown of these goals into phases.*
