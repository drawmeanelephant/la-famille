# La Famille - Architectural Component Map and Micro-Improvement Audit

## Part 1: Component Identification

### `internal/generator`
Orchestrates the static site generation process, including concurrent page building via worker pools and tying together rendering, data generation, and assets.

### `internal/render`
Manages HTML template compilation and rendering, converting page objects into final HTML files with double-checked locking for thread safety.

### `internal/transform`
Handles Abstract Syntax Tree (AST) transformations, primarily resolving and modifying internal markdown links to match the final generated site structure.

### `internal/asset`
Handles parsing of `.gitignore` files and copying static assets recursively from source to destination directories.

### `internal/search`
Generates a search index (`search.Item`) and handles sanitization of text from markdown content using regex and `strings.Builder`.

### `internal/taxonomy`
Manages tag taxonomies in `GenerateTags`.

### `internal/graph`
Constructs and manages a directed graph of inter-page links (`Graph` struct with Nodes and Edges) and writes this structure to JSON.

### `internal/ragexport`
Exports the project files into RAG-friendly (Retrieval-Augmented Generation) bundled markdown files.

### `internal/config`
Parses the site configuration file (using `yaml.v2`) and provides default settings.

### `internal/content`
Manages the reading and parsing of markdown content files.

### `internal/stub`
Generates placeholder HTML pages for broken or missing internal links found during the build process.

### `internal/page`
Defines the core `Page` struct containing site configuration, HTML content, metadata, and rendering fields.

### `internal/sitedata`
Writes out overarching site data like metadata dictionaries and XML sitemaps.

### `internal/markdown`
Configures and instantiates the Goldmark Markdown rendering engine with specific extensions (GFM, Typographer).

### `internal/git`
Provides native interactions with the git CLI (via `exec.Command`), handling actions like checking for uncommitted changes, committing, and pushing branches.

### `internal/github`
Implements an HTTP client (`Client` struct) for interacting with the GitHub API, handling PR listing and retrieving CI check statuses.

### `internal/watcher`
Manages file system watching (via fsnotify) for live-reloading during development.

## Part 2: Micro-Improvements

Based on a review of the codebase, here are 3 high-ROI micro-improvements focused on performance and maintainability:

### 1. Optimize Slice Usage in Iteration
- **`internal/graph/write.go`**: Sorting operations in `WriteGraphFiles` operate on slices from a map. We should investigate map iteration order determinism for graph logic or further pre-allocation.
- **`internal/sitedata/write.go`**: When extracting keys from the `metaData` map for deterministic sitemap generation, the slice `keys` is pre-allocated with `make([]string, 0, len(metaData))`. This is a good pattern that should be maintained and applied if new similar map-to-slice conversions are introduced.

### 2. Pre-allocate `strings.Builder` Growth
When concatenating strings (e.g., generating HTML script injections or building RAG bundles), utilizing `Builder.Grow(n)` before writing avoids multiple memory allocations.
- **`internal/render/render.go`**: The codebase already leverages `final.Grow(len(s) + 250)` when injecting the live-reload script tag into the HTML body. This pattern should be standardized.
- **`internal/search/search.go`**: `ExtractSnippet` uses `sb.Grow(len(s))` before stripping characters. This is optimal. We should ensure this is used in `internal/ragexport/export.go` if large buffers are built for RAG bundles.

### 3. Enhance Error Wrapping with `%w`
Improve error traceability by consistently using `fmt.Errorf("...: %w", err)` rather than `%v` or `.Error()`.
- **Throughout `internal/`**: For example, `internal/sitedata/write.go` uses `fmt.Errorf("failed to write meta.json: %w", err)`, which is excellent. However, a codebase-wide check should be performed to ensure all errors propagating from packages like `internal/content` or `internal/search/search.go` (e.g., `WriteMinifiedJSON` currently returns the raw `err`) wrap context appropriately.
