# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a mapping of the major components located in the `internal/` directory and their primary responsibilities based on the codebase structure:

*   **`internal/generator`**: Orchestrates the core static site build process (`Build`). It manages concurrent processing of markdown files using worker pools, handles metadata gathering, search indexing, stub generation, tag generation, and writes outputs using the render package.
*   **`internal/render`**: Manages HTML template discovery and rendering. It handles caching of parsed templates (`cacheEntry`), thread-safe layout compilation, partial injections, and applying the Goldmark processor to convert Markdown to HTML.
*   **`internal/transform`**: Responsible for AST-level transformations during the Markdown parsing phase. Includes link transformation (`LinkTransformer` mapping `.md` links to `.html` output paths) and custom AST node injections (e.g., `EmojiKitchenParser`).
*   **`internal/asset`**: Handles the copying of static web assets from the source asset directory to the output directory (`CopyAssets`). Features native `.gitignore` parsing to skip excluded assets without relying on `git` subprocesses.
*   **`internal/search`**: Responsible for creating the JSON search index required for client-side search functionality. It generates search data structs (`Item`) and writes out the index JSON.
*   **`internal/taxonomy`**: Handles categorization features, specifically gathering tags from Markdown frontmatter across all files and generating a tag map, which is subsequently rendered into tag-specific taxonomy pages.
*   **`internal/graph`**: Manages the construction and exporting of knowledge-graph data. It structures pages as nodes and inter-page links as edges (`Graph`), computing backlinks and outputting structured JSON representations.
*   **`internal/ragexport`**: Implements specialized project exporting for Retrieval-Augmented Generation (RAG) consumption. It creates localized markdown bundles of the codebase for AI agents or LLM contexts.
*   **`internal/config`**: Responsible for parsing, validating, and managing the core site configuration (`Config`, `SiteLink`). It sets up defaults, loads config from YAML files, and validates critical paths and parameters to prevent path traversal.
*   **`internal/content`**: Handles file system interactions for the content directory. Primarily focuses on recursively walking directories to extract YAML frontmatter and internal file metadata while skipping non-content files.
*   **`internal/stub`**: Responsible for detecting missing linked files (broken links) and automatically generating placeholder "stub" pages to prevent 404s and prompt future content creation, maintaining a fully connected graph.
*   **`internal/page`**: Defines the central data structures representing a fully parsed page (`Page` struct), combining its metadata, extracted links, and final HTML content for the templating layer to consume.
*   **`internal/sitedata`**: Handles the generation of site-wide metadata output files, specifically writing the `meta.json` file containing global site information and the XML `sitemap.xml` required for search engine indexing.
*   **`internal/markdown`**: Configures and initializes the primary Goldmark Markdown engine, attaching necessary extensions and registering custom parsers/transformers.
*   **`internal/git`**: Provides native Go wrappers around Git subprocess executions. Handles local repository status checks, adding, committing, pushing, and branch manipulation for the automated workflow.
*   **`internal/github`**: Manages interactions with the GitHub API (`Client`) and implements the automated Pull Request sync routine. It handles PR creation, listing open PRs, checking CI status, and automatic merging.
*   **`internal/watcher`**: Implements the local development live-reload server. It uses `fsnotify` to track file system changes and Server-Sent Events (SSE) (`LiveReloadHandler`) to automatically refresh the browser when markdown/templates change.

## Part 2: High-ROI Micro-Improvements

Here are 3 localized enhancements to improve performance, correctness, and codebase hygiene without requiring major architectural shifts:

### 1. Pre-allocate `sync.WaitGroup` worker buffers
In `internal/generator/generator.go`, the `errs` slice inside the job update loop (`type jobUpdate struct`) could benefit from exact capacity pre-allocation to prevent slice growth overhead during the parallel build phase, which handles hundreds of pages.

### 2. Improve HTTP Response Error Handling in GitHub Client
In `internal/github/github.go:64`, the error format `fmt.Errorf("API error: status=%d %s", resp.StatusCode, string(b))` dumps the entire raw HTTP response body directly into the error string. This should be truncated or properly wrapped as a custom error type to prevent massive log bloat if the GitHub API returns a large HTML 500 error page.

### 3. Consolidate Context Cancellation in Watcher
In `internal/watcher/livereload.go`, SSE clients are managed manually via `clients = make(map[chan struct{}]bool)` and `clientsMu sync.Mutex`. Using `context.Context` and its native `Done()` channel for SSE client disconnection handling (e.g. leveraging `r.Context().Done()`) would significantly clean up the manual client channel management and reduce the risk of goroutine leaks if a client abruptly disconnects without triggering the explicit deletion block.
