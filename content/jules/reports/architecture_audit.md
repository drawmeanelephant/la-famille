# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a map of the major components within the `internal/` directories and their core responsibilities based on the codebase structure:

- **generator**: Orchestrates the static site build process, manages caching (`.la-famille-cache.json`), handles concurrency during generation, and computes overall content health metrics.
- **render**: Responsible for rendering markdown AST into HTML templates, managing the output generation, and injecting live-reload scripts when in WatchMode.
- **transform**: Processes markdown abstract syntax trees (AST). Specifically handles URL rewriting, relative link resolution, and emoji processing (Emoji Kitchen) during page generation.
- **asset**: Manages the copying of static assets. It includes a native `.gitignore` parser to safely filter out ignored files without shelling out to `git`, preventing accidental publication of internal files.
- **search**: Generates the minified JSON search index used by the frontend for full-text and tag-based searches.
- **taxonomy**: Manages the extraction, aggregation, and organization of tags and categories across all markdown files.
- **graph**: Constructs and manages the knowledge graph (nodes, edges, backlinks) representing the relationships between different markdown pages.
- **ragexport**: Handles the export of the entire knowledge base into a single concatenated Markdown file optimized for Retrieval-Augmented Generation (RAG) consumption by LLMs.
- **config**: Parses, validates, and provides default values for the site's configuration (usually from `la-famille.yml`).
- **content**: Responsible for reading Markdown files from the content directory, parsing YAML frontmatter, and separating metadata from the raw content.
- **stub**: Handles the generation of stub pages (placeholder markdown files) for unresolved internal links to prevent broken links in the knowledge graph.
- **page**: Contains the view models and data structures passed into the HTML layout templates during rendering.
- **sitedata**: Generates global site metadata files, primarily the `sitemap.xml`.
- **markdown**: Configures and initializes the Goldmark markdown parser with custom extensions (GFM, Typographer) and HTML renderers.
- **git**: Provides local Git repository operations, such as checking working tree status, committing changes, and handling branching.
- **github**: Manages interactions with the GitHub API, including syncing changes, checking pull request policies, and validating automated workflows.
- **watcher**: Implements the WatchMode file system watcher (using `fsnotify`) for live-reloading, avoiding infinite build loops by ignoring output directories.

## Part 2: Micro-Improvements

Here are localized high-ROI micro-improvements that can be made to the architecture to reduce technical debt and improve performance:

### 1. Struct Field Alignment for Better Memory Packing
Several structs across the codebase are sub-optimally packed, leading to wasted memory via padding bytes. Running `fieldalignment` reveals optimizations:
- In `internal/generator/cache.go`, reordering `buildCache` fields reduces pointer bytes from 176 to 144.
- In `internal/generator/generator.go`, `BuildResult` and the internal `jobUpdate` struct can be tightened (e.g., dropping 32 pointer bytes).
- Other instances exist in `internal/ask/server.go`, `internal/retrieval/loader.go`, and `internal/github/policy_test.go`. Consistently aligning struct fields (putting larger types first) will passively improve memory density without logic changes.

### 2. Upgrading Linear Slice Scans to Maps (Technical Debt)
In `internal/transform/link_transformer.go` (around line 186), when an unresolved link is encountered, the target is added to the `MissingFiles` map. However, the code performs a linear scan (`for _, p := range parents`) to check if the current file is already recorded as a parent. Converting `MissingFiles` from `map[string][]string` to `map[string]map[string]struct{}` (or a set abstraction) would upgrade this from $O(N)$ to $O(1)$ and make the semantics explicitly clear that duplicates are not allowed.

### 3. Pre-allocating Slices Where Length is Known (Performance)
In `internal/generator/health.go` (`ComputeContentHealth`), slices like `health.MissingDescriptions` and `health.MissingDates` are appended to dynamically. Since the maximum possible size is `renderedCount` (which can be calculated first by iterating the map to count valid pages, or dynamically tracked), pre-allocating these slices with a capacity via `make([]string, 0, expected)` would reduce reallocation overhead during the main loop over `fileMap`.

### 4. Better Error Context via Wrapping (Observability)
In `internal/asset/copy.go` (`CopyFile`), errors are properly wrapped (e.g., `fmt.Errorf("failed to open source: %w", err)`). However, in many other places like `internal/generator/cache.go` (`loadBuildCache`, `hashFile`), raw errors are returned directly (e.g., `return cache, err`). Wrapping these generic I/O and JSON parsing errors with the specific context (e.g., `fmt.Errorf("reading cache file %q: %w", path, err)`) would significantly aid debugging during build failures.
