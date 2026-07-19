# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is the functional mapping of the core `internal/` packages based on the current architecture:

*   **`internal/generator`**: The core build orchestrator. Coordinates parallel processing (via worker goroutines) to transform markdown files into HTML and assemble the site.
*   **`internal/render`**: HTML templating engine that handles template caching, partials discovery, and injecting live-reload scripts for the local server.
*   **`internal/transform`**: Houses Goldmark AST transformers (like `LinkTransformer`) to resolve relative links, compute output paths, parse custom blocks (like Emoji Kitchen), and populate the backlink graph.
*   **`internal/asset`**: Manages copying static assets to the output directory and natively parses local `.gitignore` files to exclude unneeded assets safely.
*   **`internal/search`**: Generates the minified JSON search index and handles clean text snippet extraction from markdown content using regex and rune iteration.
*   **`internal/taxonomy`**: Processes file tags and generates aggregate HTML tag index pages.
*   **`internal/graph`**: Manages the data model for the backlink and file dependency graph.
*   **`internal/ragexport`**: Generates bundled Markdown exports (RAG archives) for LLM ingestion.
*   **`internal/config`**: Handles loading, writing defaults, and structural boundary validation of the core `Config` struct.
*   **`internal/content`**: Walks the content directories and parses YAML frontmatter and markdown body content into `FileMeta` structs.
*   **`internal/stub`**: Generates placeholder "stub" HTML pages for broken or missing internal links discovered during AST traversal.
*   **`internal/page`**: Defines the standard `Page` data model used as the context for executing HTML templates.
*   **`internal/sitedata`**: Writes out overarching site metadata (`meta.json`) and the XML sitemap (`sitemap.xml`).
*   **`internal/markdown`**: Configures the Goldmark markdown engine with extensions (GFM, Typographer) and registers custom parsers.
*   **`internal/git`**: Provides native wrapper functions for local Git CLI operations (status, commit, push) used by automated routines.
*   **`internal/github`**: Interacts with the GitHub API to list, validate, merge, and create PRs (used by the background sync routine).
*   **`internal/watcher`**: Implements the file system watcher and SSE (Server-Sent Events) handler for the live-reload development server.

---

## Part 2: High-ROI Micro-Improvements

Here are 4 targeted, localized enhancements that address technical debt and performance without overhauling the architecture:

### 1. Concurrency Bottleneck in Template Cache (`internal/render/render.go`)
*   **Issue**: The `Renderer.HTML` method uses an exclusive lock (`r.mu.Lock()`) for *every single map read* when looking up cached templates and `sync.Once` instances. Because this occurs on every page render, the parallel workers in `internal/generator` serialize around this lock.
*   **Fix**: Implement a true read-write double-checked lock. Acquire `r.mu.RLock()` to check for `once` and `entry` existence. If missing, `RUnlock()`, acquire the exclusive `Lock()`, verify existence again, and then instantiate. This will dramatically improve concurrent build performance.

### 2. Linear Slice Scan in Link Deduplication (`internal/transform/link_transformer.go`)
*   **Issue**: When recording missing files for stubs, the transformer iterates linearly over `t.MissingFiles[targetRelPath]` (`for _, p := range parents`) to check if `t.CurrentFile` is already present before appending.
*   **Fix**: Change the `MissingFiles` tracking structure to use a map for deduplication (e.g., `map[string]map[string]struct{}`). This converts an $O(N)$ linear slice scan into an $O(1)$ lookup, cleaning up localized technical debt and speeding up graph traversal.

### 3. Memory Allocation in Search Snippets (`internal/search/search.go`)
*   **Issue**: `ExtractSnippet` attempts to normalize whitespace by calling `strings.Fields(sb.String())` and then `strings.Join(..., " ")`. This causes multiple unnecessary array allocations and increases garbage collection overhead.
*   **Fix**: Since the `strings.Builder` is already being populated character-by-character, simply track the previous character's state to prevent writing sequential spaces natively. This eliminates the `strings.Fields` and `strings.Join` calls entirely.

### 4. Improve Error Context in Static Asset Sync (`internal/asset/copy.go`)
*   **Issue**: If an individual file fails to copy in `CopyAssets`, the `filepath.WalkDir` returns the raw error.
*   **Fix**: Enhance error wrapping inside the walk callback (e.g., `fmt.Errorf("failed to process asset %s: %w", path, err)`) so that debugging which specific static file caused a permission or read failure is trivial.
