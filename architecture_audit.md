# La Famille Architecture & Micro-Improvement Audit

## Part 1: Component Identification

The `internal/` directory contains the core application logic, modularized into specific packages:

- **`generator`**: Orchestrates the entire site build process, utilizing worker pools for concurrent markdown conversion and file generation.
- **`render`**: Manages template discovery and HTML rendering, ensuring safe concurrent access to templates during the build.
- **`transform`**: Handles Markdown AST transformations during parsing (via Goldmark), specifically rewriting links, tracking missing files for stubs, and handling custom emoji syntax.
- **`asset`**: Handles parsing local `.gitignore` files and safely copying static assets from the asset directory to the output directory.
- **`search`**: Responsible for generating a minified JSON search index by extracting and sanitizing text snippets from markdown content.
- **`taxonomy`**: Manages the extraction and aggregation of metadata tags into structured taxonomy indexes.
- **`graph`**: Generates JSON representations of the site's link structure (nodes, edges, backlinks) for interactive graph visualizers.
- **`ragexport`**: Bundles source code and markdown content into flat, RAG-friendly (Retrieval-Augmented Generation) archives.
- **`config`**: Parses and validates the site configuration via YAML files using `gopkg.in/yaml.v2`.
- **`content`**: Walks the content directory, parses YAML frontmatter using `github.com/adrg/frontmatter`, and validates/normalizes metadata (e.g., tags, dates).
- **`stub`**: Generates placeholder pages ("stubs") for broken or missing links discovered during AST transformation, utilizing sanitized HTML for inline previews.
- **`page`**: Defines the central `Page` struct model used to pass data to HTML templates during rendering.
- **`sitedata`**: Handles writing aggregated site metadata JSON and generating the `sitemap.xml`.
- **`markdown`**: Configures and instantiates the `goldmark` Markdown parser engine with necessary extensions (GFM, Typographer) and AST transformers.
- **`git`**: Provides utility wrappers around the external `git` CLI for checking uncommitted changes and fetching remote URLs.
- **`github`**: Implements a client for interacting with the GitHub API, handling tasks like pull request creation and branch syncing.
- **`watcher`**: Manages the local development server, handling file system watching, rebuilding, and serving live-reload Server-Sent Events (SSE) to the browser.

## Part 2: High-ROI Micro-Improvements

1. **`search.ExtractSnippet` (internal/search/search.go)**:
   - **Improvement**: Pre-allocate the `runes` slice if possible, or avoid intermediate conversions. The function creates an intermediate `sb.String()`, then splits/joins it, then converts to a `[]rune`. This creates multiple intermediate string allocations. A more optimized approach could build the snippet directly with length tracking to avoid the final `[]rune` cast for truncation.

2. **`content.GatherMetadata` (internal/content/metadata.go)**:
   - **Improvement**: The tag normalization loop uses `strings.Builder` heavily and allocates a new builder per tag. The normalized tag validation checks could be short-circuited. Furthermore, using a `struct{}` map rather than a slice for tracking unique tags per file (or globally) could avoid `O(N^2)` uniqueness checks if implemented later, though currently it normalizes directly into a slice.

3. **`sitedata.Write` (internal/sitedata/write.go)**:
   - **Improvement**: In the sitemap generation loop, it sorts the keys to ensure deterministic output (which is good), but uses a simple string concatenation loop via `strings.Builder`. Pre-allocating `sitemapBuilder.Grow()` based on a heuristic of `len(keys) * avgUrlLength` would eliminate internal buffer reallocations during the loop.

4. **`stub.GenerateStubs` (internal/stub/stub.go)**:
   - **Improvement**: It creates `missingKeys := make([]string, 0, len(missingFiles))` which is excellent (pre-allocated). However, it discovers partials on every invocation via `render.DiscoverPartials(filepath.Dir(cfg.Template))`. If this is called multiple times or repeatedly in watch mode, the partial discovery could be cached or moved up to the `generator` and passed down to avoid redundant filesystem walks.

5. **`generator.Build` (internal/generator/generator.go)**:
   - **Improvement**: Ensure the worker pool slice channels and wait groups are properly sized. If the number of files is known beforehand (which it is, from `content.GatherMetadata`), sizing the worker channels to `len(fileMap)` can prevent goroutine blocking.
