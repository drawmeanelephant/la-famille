# Architectural Review: La Famille

## Part 1: Component Identification

The `internal/` directory contains the core logic of the La Famille application, modularized into specific responsibilities:

- **`internal/generator`**: Orchestrates the overall static site build process. It coordinates metadata gathering, triggers markdown conversion, handles concurrent rendering, and generates outputs like stubs and search indexes.
- **`internal/render`**: Manages HTML templating. It parses layouts and partials, caches templates, and executes them to generate the final HTML. It also handles injecting the LiveReload script during WatchMode.
- **`internal/transform`**: Provides Markdown AST transformations (via Goldmark). It resolves and rewrites relative and root links (`link_transformer.go`), determines output URLs, and implements custom parsing like the Emoji Kitchen.
- **`internal/asset`**: Handles the copying of static assets (e.g., CSS, images) from the asset directory to the output directory while respecting `.gitignore` patterns and preventing path traversals.
- **`internal/search`**: Responsible for generating a minified JSON search index used by the frontend to provide search functionality over the markdown content.
- **`internal/taxonomy`**: Processes frontmatter tags across all content files to generate taxonomy pages or tag indexes.
- **`internal/graph`**: Defines the site's graph data structure (nodes and edges) and writes graph data and backlinks to JSON files, enabling network visualizations or bidirectional linking.
- **`internal/ragexport`**: Exports markdown bundles and content optimized for Retrieval-Augmented Generation (RAG) applications.
- **`internal/config`**: Handles loading, validating, and generating default configuration (`Config`) from YAML files to customize the site's behavior.
- **`internal/content`**: Gathers metadata by walking the content directory, parsing YAML frontmatter, validating dates/tags, and mapping each markdown file to a `FileMeta` struct.
- **`internal/stub`**: Generates placeholder HTML pages (stubs) for markdown files that are linked to but do not currently exist in the content directory.
- **`internal/page`**: Defines the `Page` data model (struct) that is passed into HTML templates during rendering.
- **`internal/sitedata`**: Writes aggregate site metadata (and potentially sitemaps) to JSON files in the output directory.
- **`internal/markdown`**: Configures and initializes the Goldmark markdown engine with necessary extensions and custom AST transformers.
- **`internal/git`**: Provides native Go wrappers around Git operations (commit, push, checkout) and parses repository URLs for automation.
- **`internal/github`**: Implements a GitHub API client to sync data, list/manage pull requests, and check CI/CD statuses.
- **`internal/watcher`**: Implements a file system watcher (`fsnotify`) to detect changes and a Server-Sent Events (SSE) handler to trigger live-reloads in the browser.

## Part 2: Micro-Improvements

Here are 5 high-ROI micro-improvements focusing on localized enhancements and technical debt:

1. **Pre-allocate Slices Where Length is Known (Performance):**
   In `internal/content/metadata.go`, the `normalizedTags` slice is dynamically appended to within a loop over `matter.Tags`. Pre-allocating the slice using `normalizedTags := make([]string, 0, len(matter.Tags))` will prevent unnecessary memory reallocations during parsing.

2. **Optimize `strings.Builder` Initialization (Performance):**
   In `internal/stub/stub.go`, the `htmlContent` (`strings.Builder`) is used extensively to construct HTML stubs. Adding an explicit `htmlContent.Grow(1024)` before writing will pre-allocate the internal byte slice, reducing reallocation overhead during heavy string concatenations.

3. **Convert Linear Slice Scans to Maps (Technical Debt):**
   In `internal/transform/link_transformer.go`, the `MissingFiles` map tracks parents of missing links using a slice (`map[string][]string`). When appending new parents, it performs a linear search to check for duplicates (`found := false; for _, p := range parents ...`). Changing this to a map-based set (e.g., `map[string]map[string]struct{}`) would change the check to an $O(1)$ lookup and simplify the logic.

4. **Better Error Wrapping for Path Traversal Validation (Logging/Errors):**
   In `internal/generator/generator.go`, when `pathutil.IsSafePath` fails, the system currently skips rendering and logs a generic warning (`slog.Warn("Potential path traversal...")`). Returning a wrapped error (e.g., `fmt.Errorf("path traversal detected for %s: %w", outPath, err)`) and properly surfacing it to the `errs` slice would make CI/CD failures explicitly clear rather than silently skipping files.

5. **Struct Field Alignment for Memory Packing (Memory):**
   In `internal/graph/graph.go`, the `Node` struct currently interleaves smaller fields (booleans) before a larger slice field, which introduces padding bytes. By moving the `Render bool` and `Missing bool` fields to the end of the struct, you can optimize memory packing, which is beneficial when the graph contains thousands of nodes.
