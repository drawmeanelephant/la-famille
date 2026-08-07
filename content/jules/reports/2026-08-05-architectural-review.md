---
date: "2026-08-05"
title: "Component Mapping & Micro-Improvement Audit: La Famille"
author: "Jules"
---

# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a map of the major components currently residing in the `internal/` directory and their core responsibilities based on the codebase structure:

- **internal/generator**: Orchestrates the main build process. It coordinates concurrent rendering, handles build caches, tracks health/metrics (e.g., top tags), and ensures no conflicts occur during site generation.
- **internal/render**: Handles HTML template execution and management. It resolves layouts based on configuration or frontmatter, ensures layout navigation anchors resolve, and applies templates to transformed content.
- **internal/transform**: Processes markdown abstract syntax trees (ASTs), focusing on resolving and transforming links (like wiki-links and image sources) to ensure they point to correct, generated output paths.
- **internal/asset**: Manages static asset copying and directory structure mapping. It provides utilities for natively evaluating `.gitignore` patterns (without git subprocesses) and tracks output path ownership to prevent collisions.
- **internal/search**: Generates the minified JSON search index by extracting clean snippets (stripping HTML/markdown formatting) and indexing headings from raw content for client-side search functionality.
- **internal/taxonomy**: Generates category and tag index pages by extracting metadata from markdown frontmatter and creating aggregate listings.
- **internal/graph**: Constructs the site's knowledge graph (backlinks) by resolving internal edges into deduplicated, sorted inbound and outbound neighbor lists per node.
- **internal/ragexport**: Handles the extraction and export of project files into clean, RAG-friendly markdown bundles tailored for LLM consumption.
- **internal/config**: Manages site configuration validation, defaults, and struct loading (avoiding partial/invalid loads to strictly differentiate between "no config" and "bad config").
- **internal/content**: Responsible for reading raw markdown files and parsing YAML frontmatter. It handles frontmatter key normalization (e.g., collapsing case-variants) and type coercion (like converting scalars to slices).
- **internal/stub**: Manages the generation of placeholder pages (stubs) for valid internal links that do not have a corresponding source markdown file, writing them only if the target path is unclaimed.
- **internal/page**: Defines the core data models and interfaces representing an individual site page as it passes through the rendering pipeline.
- **internal/sitedata**: Orchestrates the aggregation of site metadata and handles the generation of standard site-level data files like `sitemap.xml`.
- **internal/markdown**: Configures and manages the Goldmark engine and extensions used to convert markdown text into HTML, serving as the core parser for the content pipeline.
- **internal/git**: Provides wrappers for local Git commands and state inspection (e.g., checking for uncommitted changes, reading remotes).
- **internal/github**: Interacts with the GitHub API for operations like syncing, reading PR labels, or evaluating PR policies (used in continuous integration flows).
- **internal/watcher**: Implements the filesystem watching (WatchMode) and debouncing logic. It manages SSE-based live-reloading to coordinate lightweight, incremental site rebuilds during local development.

## Part 2: Micro-Improvements

Based on the architectural review, here are 3 to 5 localized, high-ROI micro-improvements that align with existing conventions and memory profiles:

1. **Optimize Struct Alignment (Memory Packing)**
   Go's `fieldalignment` tool identifies several key structs with poor field alignment, leading to wasted pointer bytes due to padding. Packing fields sequentially by size can significantly reduce garbage collection overhead on heavily instantiated structs. For example:
   - `internal/generator/generator.go`: `SiteGenerator` struct fields could be reordered to save 32 pointer bytes per instance (136 bytes down to 104 bytes).
   - `internal/generator/cache.go`: `BuildCache` struct could be reduced from 176 to 144 pointer bytes.
   - `internal/config/config.go`: `Config` struct could be optimized from 248 to 216 pointer bytes.
   - `internal/retrieval/loader.go`: The `Document` or `Chunk` structs could be packed to reduce memory overhead (96 pointer bytes to 80).
   *Note: Applying field reordering requires manual review to ensure struct documentation comments are not stripped.*

2. **Pre-allocate Maps and Slices During Graph & Taxonomy Building**
   In both `internal/graph/adjacency.go` and `internal/taxonomy/taxonomy.go`, slices of keys are constructed from map iterations. While the slice length is often correctly pre-allocated via `make([]string, 0, len(map))`, the underlying maps constructed during AST traversal (e.g., when discovering links or tags) often start at size 0 and grow dynamically. Pre-allocating maps in `internal/graph` (where the number of edges is often known or bounded by the file size) or pre-allocating slices for deeply nested structs during `internal/content/frontmatter.go` parsing can yield small, measurable performance bumps in large workspaces.

3. **Improve Symlink Path Traversal Checks**
   In `internal/asset/copy.go` (and potentially `internal/content/`), the system must safely copy files while preventing path traversal. Currently, relying solely on `fs.FileMode` checks is insufficient since `FileMode` doesn't provide an `IsSymlink()` method. A micro-improvement is to explicitly apply the bitwise check `info.Mode() & os.ModeSymlink != 0` within `fs.WalkDir` functions to strictly skip symlinks, ensuring robust directory boundaries without relying solely on final-path evaluations.

4. **Enhance Error Wrapping Context**
   In areas handling file parsing and remote API connections, error context can be sparse. For instance, in `internal/content/frontmatter.go` when YAML unmarshaling fails, or in `internal/github/github.go` when API limits are hit, enhancing the errors with `fmt.Errorf("parsing frontmatter for %q: %w", path, err)` or standardizing API error wrappers provides significantly faster localized debugging compared to plain `err.Error()` string checks.

5. **Convert Linear Slice Scans to Map Lookups in Validation**
   When the generator verifies reserved targets (in `internal/generator/generator.go`) or the stub generator checks if a slug is usable (`internal/stub/stub.go`), there is a pattern of converting a map to a slice of keys just to perform existence checks or sorting. By maintaining a centralized `map[string]struct{}` for claimed paths across the build lifecycle, the generator can perform O(1) map lookups instead of repeated O(N) linear array scans, especially valuable when processing thousands of internal tags and page links.
