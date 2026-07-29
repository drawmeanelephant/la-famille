---
title: "Component Mapping & Micro-Improvement Audit"
date: "2026-07-27"
author: "Jules"
layout: "layout"
routine: "architectural-review"
status: "completed"
success: "true"
---
# La Famille Component Mapping & Micro-Improvement Audit

## Part 1: Component Identification

*   **internal/generator**: Orchestrates the site build process, detects output path collisions, manages staging directories, and swaps the final output directory.
*   **internal/render**: Handles HTML template caching and rendering, discovers partials and layouts, and injects live-reload scripts for watcher mode.
*   **internal/transform**: Provides utilities for URL resolution, frontmatter slug validation, and markdown AST link/emoji transformations.
*   **internal/asset**: Copies static assets safely, preventing traversal vulnerabilities, and natively evaluates `.gitignore` patterns to exclude unwanted files.
*   **internal/search**: Processes raw markdown to generate lightweight search index JSONs, extracting clean text snippets and headings by stripping HTML and markdown syntax.
*   **internal/taxonomy**: Normalizes and processes taxonomy values (tags and categories) to construct valid taxonomy lists and paths.
*   **internal/graph**: Computes internal page adjacencies and backlinks to generate data for the interactive knowledge graph explorer.
*   **internal/ragexport**: Generates specialized Markdown bundle exports designed for Retrieval-Augmented Generation (RAG) consumption.
*   **internal/config**: Loads, models, and validates the `config.yaml` definitions (e.g., paths, URLs, features), ensuring secure isolated output paths.
*   **internal/content**: Discovers `.md` files, parses frontmatter, handles case-normalization of keys, and extracts all page metadata.
*   **internal/stub**: Claims paths and generates stub pages ("Under Construction") for broken or dangling internal links to prevent 404 errors.
*   **internal/page**: Defines the core `Page` struct model passed into the HTML templates, encapsulating site config and markdown attributes.
*   **internal/sitedata**: Produces site-wide metadata artifacts such as sitemaps for discovery and SEO.
*   **internal/markdown**: Instantiates and configures the Goldmark markdown engine with GFM, typographer, and custom AST transformer extensions.
*   **internal/git**: Provides low-level Go wrappers around CLI `git` commands (e.g., checkout, commit, status, parsing remotes).
*   **internal/github**: Interacts with the GitHub API to manage pull requests, query commit check runs, and automate squashing/merging operations.
*   **internal/watcher**: Uses `fsnotify` to track file changes across content and templates, debouncing events and coordinating rebuilds with SSE live-reloading.

## Part 2: Micro-Improvements

Here are 3 localized high-ROI micro-improvements to consider:

1.  **Optimize Map Pre-Allocation** (`internal/search/search.go`): In `ExtractHeadings`, the `seen` map for tracking unique headings is allocated without a size hint. Since the number of headings is bounded by the number of lines (which is known from `strings.Split`), initializing with a proportional capacity or a small baseline (e.g., `make(map[string]bool, 8)`) can reduce early rehashing.
2.  **Optimize Set Data Structures** (`internal/search/search.go`): Convert the `htmlElements` map from `map[string]bool` to `map[string]struct{}`. Go's empty struct consumes zero bytes, making it a more memory-efficient choice for static lookup sets than boolean values.
3.  **Optimize Map Pre-Allocation** (`internal/render/render.go`): In `DiscoverLayouts`, the `entries, err := os.ReadDir(templateDir)` call provides the exact number of files. By moving the `allowlist := make(map[string]bool)` initialization below it and using `make(map[string]bool, len(entries))`, you can avoid incremental map re-hashing during population.

### Learnings
* Validated core architecture components mapping to their single-responsibility packages.
* Confirmed Go performance optimization opportunities focusing on map size hints and optimal set implementations.
