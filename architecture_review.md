# La Famille: Architectural Review & Component Mapping

## Part 1: Component Identification

The `internal/` directory contains the core logic of the La Famille static site generator. The components and their responsibilities are mapped out below:

- **`internal/generator`**: The primary orchestrator for the build process. It coordinates a worker pool of goroutines to process markdown files concurrently, gathers metadata, tracks missing file references, and calls upon other packages to output HTML, search indices, and graph data.
- **`internal/render`**: Handles the discovery, caching, and execution of HTML templates. Uses `html/template` to render the final `.html` artifacts, injecting `page.Page` data, executing partials, and conditionally inserting live-reload scripts.
- **`internal/transform`**: Implements custom Goldmark AST transformers. Notably, it resolves internal markdown links (e.g., `[link](file.md)`) into proper output HTML relative paths and provides an Emoji Kitchen sticker parser (`!ek[...]`).
- **`internal/asset`**: Responsible for discovering and verbatim copying of static assets to the output directory while filtering out `.go` files, `testdata`, and respecting `.gitignore` rules (using an optimization pass with `git check-ignore`).
- **`internal/search`**: Extracts raw text snippets from parsed markdown files (removing formatting) and writes out a minified `search.json` file for client-side consumption.
- **`internal/taxonomy`**: Aggregates tags defined in the YAML frontmatter across all pages and uses the `render` package to generate a flat taxonomy index view (e.g., `tags/example/index.html`).
- **`internal/graph`**: Manages and serializes the backlink network and site node graph into JSON structures (`graph.json` and `backlinks.json`), driving knowledge-graph visualization features.
- **`internal/ragexport`**: Generates RAG-friendly (Retrieval-Augmented Generation) markdown bundles by concatenating codebase logic and content into discrete system, config, and content artifacts.
- **`internal/config`**: Responsible for loading, parsing (`yaml.v2`), validating, and writing out default `la-famille.yml` site configurations.
- **`internal/content`**: Recursively walks the markdown content directory, parsing YAML frontmatter and normalizing metadata (like tags and dates) into `FileMeta` structures for downstream processing.
- **`internal/stub`**: Dynamically generates missing HTML "stub" pages for internally linked markdown files that were not present during the build, allowing graceful failure with "Under Construction" placeholders and backlink contexts.
- **`internal/page`**: Defines the central `Page` struct data model which encapsulates all necessary layout-facing variables (Site config, Title, Content HTML, SEO metadata) passed into standard templates.
- **`internal/sitedata`**: Generates raw site-wide metadata output (e.g., `sitedata.json` and generating `sitemap.xml` mapping) inside the public directory.
- **`internal/markdown`**: Bootstraps the core `goldmark.Markdown` parsing engine, registering the custom AST extensions and syntax configurations.
- **`internal/git`**: A wrapper over `os/exec` to execute Git commands, assisting with branch checkouts, committing, pushing, and querying remote URLs.
- **`internal/github`**: A minimalistic, custom HTTP client targeting GitHub's REST API for PR management and syncing Check Run statuses.
- **`internal/watcher`**: Implements SSE (Server-Sent Events) handlers and filesystem polling loops to trigger targeted rebuilds and broadcast live-reload signals to connected browser clients during `serve --watch`.

## Part 2: High-ROI Micro-Improvements

Here are 4 targeted, localized micro-improvements to improve performance, maintainability, and code quality without requiring structural refactors:

1. **Optimize Deduplication in `internal/ragexport` (Linear Scan to Map)**:
   In `internal/ragexport/export.go`, the `matchedFiles` slice is deduplicated using an $O(n)$ linear scan inside a `WalkDir` loop (`for _, mf := range matchedFiles`). This leads to $O(n^2)$ lookup times. Converting `matchedFiles` to a `map[string]struct{}` lookup during the traversal and then flattening it to a sorted slice afterward will improve efficiency on larger repositories.

2. **Improve Error Observability in `internal/render` Initialization**:
   In `internal/render/render.go`, the `New(templateDir)` function calls `DiscoverLayouts(templateDir)`. If `DiscoverLayouts` throws an error (e.g., if the template directory is missing or permissions are invalid), the error is completely swallowed and `allowlist` is silently reset to `make(map[string]bool)`. Logging this error (`log.Printf`) before recovering will improve debuggability for missing layout configurations.

3. **Struct Field Alignment in `internal/config.Config`**:
   The `Config` struct in `internal/config/config.go` places the `SiteLinks []SiteLink` slice (24 bytes) near the bottom of the struct after multiple strings (16 bytes) and before smaller primitives like `Port int` (8 bytes) and `WatchMode bool` (1 byte). Reordering the struct fields by descending size—moving `SiteLinks` to the top—will optimize memory packing and reduce padding overhead.

4. **Pre-allocate Maps/Slices using Known Lengths (`internal/generator` & `internal/taxonomy`)**:
   In `internal/generator/generator.go`, the `searchIndexItems` step explicitly allocates `make([]search.Item, len(keys))`. However, the final `searchIndex` array dynamically appends items using `append(searchIndex, item)`. Since the upper bound is known, pre-allocating `searchIndex := make([]search.Item, 0, len(searchIndexItems))` before the loop will avoid slice reallocation overhead. Similarly, `tagMap` capacities in `taxonomy.go` could be guessed to prevent reallocations.
