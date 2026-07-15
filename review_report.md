# Architectural Review: La Famille

## Part 1: Component Identification

The `internal/` directory contains the core application logic, modularized into specific responsibilities:

- **internal/generator**: Orchestrates the static site generation build process, coordinating metadata extraction, rendering, and asset copying.
- **internal/render**: Manages the HTML rendering process using Go templates.
- **internal/transform**: Handles Markdown AST transformations (via Goldmark), such as converting `.md` links to HTML paths and managing slugs.
- **internal/asset**: Natively parses `.gitignore` and handles copying static assets to the output directory while preventing path traversal.
- **internal/search**: Generates and sanitizes a minified JSON search index for the static site.
- **internal/taxonomy**: Processes tags from content frontmatter and generates tag-specific index pages.
- **internal/graph**: Constructs and writes JSON representations of document graphs and backlinks.
- **internal/ragexport**: Bundles project files into RAG-friendly markdown exports.
- **internal/config**: Defines, loads, and validates the global site configuration from YAML.
- **internal/content**: Discovers Markdown source files and parses YAML frontmatter to extract metadata.
- **internal/stub**: Generates placeholder pages for missing links within the project graph.
- **internal/page**: Defines the data structures (like the `Page` struct) passed into HTML templates during rendering.
- **internal/sitedata**: Writes JSON site metadata and related outputs to the build directory.
- **internal/markdown**: Configures the Goldmark engine and registers extensions/transformers.
- **internal/git**: Wraps basic Git commands (e.g., checking for uncommitted changes).
- **internal/github**: Interacts with the GitHub API for operations like syncing and checking PR status.
- **internal/watcher**: Provides file system watching (fsnotify) and Server-Sent Events (SSE) for live-reloading during development.

## Part 2: Micro-Improvements

Here are 3 high-ROI micro-improvements tailored for the current codebase:

1. **Struct Field Alignment (internal/config/config.go):**
   - The `config.Config` struct has a bool (`WatchMode`) at the end, along with a mix of slices, ints, and strings. Reordering fields from largest to smallest (e.g., slices, then strings, then ints, then bools) can minimize padding and reduce the memory footprint.

2. **Struct Field Alignment (internal/page/page.go):**
   - The `page.Page` struct has multiple string fields mixed with `template.HTML` and the embedded `config.Config` struct. Packing these fields optimally will reduce memory size during concurrent rendering jobs.

3. **Pre-allocate Slices for Known Lengths in Errors (internal/generator/generator.go):**
   - In `internal/generator/generator.go` (around line 328), the slice `var joinErrs []error` is dynamically appended in a loop over `errs`. Since the length of `errs` is known, this should be pre-allocated via `joinErrs := make([]error, 0, len(errs))` to prevent reallocation overhead, aligning with the codebase's performance conventions.
