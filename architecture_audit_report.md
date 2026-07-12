# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a mapping of the major components currently residing in the `internal/` directories and a brief summary of their responsibilities based on the current code:

*   **`internal/generator`**: Orchestrates the core build process for the static site. It ties together config reading, metadata gathering, stub generation, link transforming, and rendering markdown into the output directory.
*   **`internal/render`**: Manages HTML template rendering. It discovers partials and layouts, implements caching for parsed templates, and applies the parsed templates to page structures to generate the final HTML output.
*   **`internal/transform`**: Handles AST transformations during markdown parsing. This includes link resolution/transformations (`LinkTransformer`) to handle relative/absolute URLs and custom extensions like the `EmojiKitchenParser`.
*   **`internal/asset`**: Responsible for copying static assets (like images, CSS, JS) from the asset directory to the output directory while honoring `.gitignore` rules natively.
*   **`internal/search`**: Processes markdown and HTML content to extract clean text snippets, stripping code blocks and formatting markers, and formats this into JSON for the client-side search index.
*   **`internal/taxonomy`**: Handles tag extraction, normalization, and generating index pages for each tag found across the markdown content.
*   **`internal/graph`**: Constructs and serializes the graph of bidirectional links (backlinks and forward links) between pages, enabling network views.
*   **`internal/ragexport`**: Packages the project’s content into clean, concatenated markdown files suitable for Retrieval-Augmented Generation (RAG) consumption by LLMs.
*   **`internal/config`**: Defines the `Config` struct, handles loading `la-famille.yml`, and validates configuration parameters (e.g., path boundaries, port numbers).
*   **`internal/content`**: Walks the content directory, parses YAML frontmatter using `goldmark`, and returns metadata (like Title, Tags, Date) along with the raw body text.
*   **`internal/stub`**: Automatically generates placeholder HTML pages for broken links (files that are linked to but don't exist yet) along with backlinks pointing to where they were referenced.
*   **`internal/page`**: Defines the core `Page` struct used to pass data (Content, Meta, Config, etc.) into the HTML templates during the render phase.
*   **`internal/sitedata`**: Handles the writing of structural site data to the output directory, such as the `sitemap.xml` and `site_meta.json`.
*   **`internal/markdown`**: Configures and initializes the Goldmark engine with extensions (e.g., GFM, Typographer, frontmatter) and custom link transformers.
*   **`internal/git`**: Provides native Go wrappers and shell-outs for standard git operations (like checkout, commit, push) and checking for uncommitted changes.
*   **`internal/github`**: Manages interaction with the GitHub API for syncing PRs, listing open PRs, checking PR statuses (CheckRuns), and orchestrating background sync workflows.
*   **`internal/watcher`**: Implements WatchMode. Uses `fsnotify` to watch for file changes, triggers partial rebuilds, and manages the Server-Sent Events (SSE) server for browser live-reloading.

## Part 2: Micro-Improvements

Here are 4 high-ROI micro-improvements that can be implemented for localized enhancements:

1.  **Pre-allocate `strings.Builder` capacity with `Grow()`**:
    *   **Location**: `internal/sitedata/write.go` (`sitemapBuilder`), `internal/taxonomy/taxonomy.go` (`htmlContent`), `internal/stub/stub.go` (`htmlContent`), and `internal/content/metadata.go` (`sb`).
    *   **Issue**: These packages build relatively large HTML or XML strings in memory iteratively without pre-allocating the underlying byte slice.
    *   **Fix**: Call `builder.Grow(estimatedSize)` right after declaring the `strings.Builder`. For example, in `sitemapBuilder.Grow(4096)` based on an estimated sitemap size. This prevents multiple memory re-allocations as the string grows.

2.  **Optimize Struct Alignment for Memory Packing**:
    *   **Location**: `internal/config/config.go` (`Config`), `internal/content/metadata.go` (`FileMeta`), and `internal/page/page.go` (`Page`).
    *   **Issue**: Several large structs have loosely ordered fields. For instance, in `Config`, boolean fields (`WatchMode`) or integer fields (`Port`) are placed next to slices (`SiteLinks`) or strings.
    *   **Fix**: Reorder the fields from largest to smallest (e.g., Slices/Maps first, Strings/Pointers next, Integers next, and Booleans last) to reduce struct padding and optimize memory layout, especially when processing hundreds of pages.

3.  **Use `sync.OnceValues` for Discovering Layouts/Partials**:
    *   **Location**: `internal/render/render.go` (in `DiscoverLayouts` and `DiscoverPartials` initialization).
    *   **Issue**: The `Renderer` currently stores `onces map[string]*sync.Once` and requires manual locking to initialize specific templates.
    *   **Fix**: If Go 1.21+ is available, use `sync.OnceValues` or standard `sync.Map` for concurrent cache population, which slightly simplifies the locking logic in `Render.HTML` and reduces the risk of map-read panics.

4.  **Refine Slice Pre-allocation in `generator.go`**:
    *   **Location**: `internal/generator/generator.go`
    *   **Issue**: `searchIndexItems := make([]search.Item, len(keys))` is well-allocated. However, loops that append to slices like `missingFiles` or dynamically built file lists might benefit from capacity hints where bounds are known.
    *   **Fix**: Ensure `make([]T, 0, len(known))` is consistently applied across all packages where the maximum bound is known beforehand, particularly in the `WriteGraphFiles` loop inside `internal/graph/write.go`.
