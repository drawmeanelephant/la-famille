---
date: "2026-07-31"
title: "Architecture Audit"
layout: "routine"
success: "true"
status: "completed"
author: "Jules"
---

# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

The `internal/` directory contains the core domain logic and architectural components of the La Famille framework.

- **internal/generator**: Orchestrates the static site build process, managing output paths, cache, and content health metrics.
- **internal/render**: Renders Markdown content and template data into HTML output.
- **internal/transform**: Transforms Markdown ASTs, specifically handling URL slugs, markdown links, and extensions like the Emoji Kitchen.
- **internal/asset**: Manages the copying of static assets to the output directory while respecting `.gitignore` configurations natively.
- **internal/search**: Extracts and minifies searchable text, snippets, and headings from Markdown to build the `search.json` index.
- **internal/taxonomy**: Generates taxonomy and tagging pages based on content metadata.
- **internal/graph**: Computes and writes the knowledge graph and backlink adjacency data for the site.
- **internal/ragexport**: Generates RAG (Retrieval-Augmented Generation) formatted Markdown archives from the site content.
- **internal/config**: Handles site configuration parsing, validation, and defaults.
- **internal/content**: Parses Markdown frontmatter and extracts metadata into structured types.
- **internal/stub**: Generates placeholder pages for missing content referenced in the graph.
- **internal/page**: Defines the data models used for rendering HTML templates.
- **internal/sitedata**: Writes aggregated metadata and sitemaps for the output site.
- **internal/markdown**: Configures and instantiates the Goldmark markdown engine with necessary extensions.
- **internal/git**: Handles local Git status checks (e.g., detecting uncommitted changes).
- **internal/github**: Orchestrates GitHub API interactions, such as syncing pull requests and enforcing merge policies.
- **internal/watcher**: Provides file system watching and SSE-based live-reloading for the local development server.

## Part 2: Micro-Improvements

Here are some localized, high-ROI enhancements to address technical debt and improve efficiency:

1. **Struct Field Alignment (Memory Packing)**
   Several structs have suboptimal field alignment, leading to wasted padding bytes. Using `fieldalignment` from `golang.org/x/tools`, these can be optimized:
   - `LoadOptions` in `internal/retrieval/loader.go`
   - `ServerOptions` or similar in `internal/ask/server.go`
   - Structs in `internal/generator/cache.go` and `internal/generator/generator.go` (e.g., `outputClaims` or `outputOwner`)
   - Reordering fields (e.g., packing `bool`s and pointers together) reduces struct sizes and GC pressure.

2. **Slice Pre-allocation**
   In components generating lists (like `internal/search/search.go` when processing headings or snippets, or `internal/graph/write.go`), slices are sometimes grown dynamically in loops. Pre-allocating the slice capacity `make([]T, 0, expectedSize)` where the size is known (e.g., based on the number of nodes or lines) will reduce memory allocations.

3. **Map Lookups Over Linear Scans**
   In areas resolving nodes, links, or processing tags across a large dataset (such as graph resolution in `internal/graph` or deduplicating links in `internal/transform/link_transformer.go`), replacing nested loops or linear slice scans with map lookups will improve `O(N)` behavior to `O(1)`. This is especially beneficial for large sites.

4. **Error Wrapping Context**
   Many internal components construct errors without `fmt.Errorf("...: %w", err)` or custom error types. Enriching localized error returns (e.g., in `internal/asset/copy.go` when parsing `.gitignore` patterns or file permission issues) to include file paths and context using `%w` will significantly improve debugging traceability without altering core logic.
