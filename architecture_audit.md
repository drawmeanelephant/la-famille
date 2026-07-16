# La Famille Component Mapping & Micro-Improvement Audit

## Part 1: Component Identification

The `internal/` directory contains the core logic of the La Famille static site generator. Here is a mapping of the major components based on current code structure:

* **internal/generator**: Acts as the main orchestrator for the static site build process. It coordinates multi-pass processing, gathers metadata, manages concurrent rendering of markdown to HTML, and handles output file generation.
* **internal/render**: Manages HTML template rendering using Go's `html/template`. It uses caching and `sync.RWMutex` for efficient template processing and is responsible for injecting live-reload scripts when in WatchMode.
* **internal/transform**: Integrates with the Goldmark Markdown engine to mutate the AST. It handles relative/absolute link resolution, detects broken links (identifying missing files), and populates graph backlinks during parsing.
* **internal/asset**: Handles static asset synchronization. It traverses the asset directories, natively reads local `.gitignore` rules to filter ignored files, and validates output destinations to prevent path traversal vulnerabilities.
* **internal/search**: Generates a minified JSON search index from processed pages, mapping URLs to titles, tags, and content snippets for client-side search capabilities.
* **internal/taxonomy**: Processes tags and taxonomies from page frontmatter, grouping related content and generating tag-specific list pages.
* **internal/graph**: Constructs a directed graph of the site's inter-page links (nodes and edges) and serializes it to JSON for downstream structural visualization.
* **internal/ragexport**: Packages the project's source files into consolidated markdown bundles optimized for LLMs and Retrieval-Augmented Generation (RAG) consumption.
* **internal/config**: Loads, parses, and validates the application configuration (`Config`) from YAML files, providing sensible fallback defaults and enforcing local directory constraints.
* **internal/content**: Recursively walks markdown content directories to extract and normalize YAML frontmatter alongside the raw file content, producing the core `FileMeta` models.
* **internal/stub**: Auto-generates placeholder markdown stubs for missing pages that are linked to but do not exist, ensuring a complete graph.
* **internal/page**: Defines the centralized `Page` data structure that bridges the processed markdown metadata/configuration with the frontend HTML templates.
* **internal/sitedata**: Outputs aggregated frontend metadata (such as page titles and custom frontmatter properties) to a standardized `meta.json` in the output directory.
* **internal/markdown**: Centralizes the configuration and instantiation of the Goldmark engine, registering necessary plugins like GFM (GitHub Flavored Markdown) and custom AST transformers.
* **internal/git**: Provides Git CLI wrappers for operations like checking repository status, committing, branching, and extracting remote repository details.
* **internal/github**: Interacts with the GitHub API to list PRs, assert mergeability, monitor CI check-runs, and orchestrate the automated PR maintenance loop (Sync).
* **internal/watcher**: Implements local development server features, utilizing `fsnotify` for live-reloading upon file modifications and hosting a Server-Sent Events (SSE) server.

## Part 2: Micro-Improvements

Here are high-ROI, localized micro-improvements identified during the audit:

1. **Performance (Pre-allocation in Tag Normalization)**:
   In `internal/content/metadata.go`, the tag normalization iterates over strings and writes to a `strings.Builder`. We should call `sb.Grow(len(lower))` before the loop to prevent internal buffer reallocations for larger tag strings.

2. **Technical Debt (Map Lookups over Linear Slice Scans)**:
   In `internal/transform/link_transformer.go`, verifying if a file is already marked as missing iterates over a slice (`for _, p := range parents`). Converting the `MissingFiles` tracking from a `map[string][]string` to a `map[string]map[string]struct{}` would change this from an O(N) scan to an O(1) map lookup.

3. **Performance (Global Byte Slice Variables for Checks)**:
   In `internal/render/render.go`, `writeWithLiveReload` checks `bytes.Contains(rendered, []byte("new EventSource('/livereload')"))`. Allocating this byte slice literal on every page render adds unnecessary garbage collection pressure. Pre-defining it as a package-level variable (e.g., `var liveReloadCheck = []byte("new EventSource('/livereload')")`) would eliminate this overhead.

4. **Struct Memory Efficiency (Pass Configuration by Pointer)**:
   In `internal/page/page.go`, the `Page` struct embeds `config.Config` by value (as `Site config.Config`). Because `Config` has over a dozen fields (including slices and strings), copying it for every single generated page inflates memory usage. Embedding `*config.Config` instead would significantly pack memory tighter for large site builds.

5. **Error Context Wrapping**:
   In `internal/render/render.go` inside `writeWithLiveReload`, if `w.Write` fails when appending the script block (e.g., `_, err := w.Write(rendered[index:])`), the underlying raw error is returned. It should be wrapped with `fmt.Errorf("failed to write live reload script: %w", err)` to provide better diagnostic context in logs.
