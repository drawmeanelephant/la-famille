# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a mapping of the major components currently residing in the `internal/` directory and their responsibilities:

*   **`internal/generator`**: Orchestrates the build process. Its `Build` function executes multiple passes (gathering metadata, generating taxonomies, rendering markdown to HTML, writing JSON outputs, etc.) and utilizes a worker pool for concurrent markdown conversion.
*   **`internal/render`**: Handles loading and parsing HTML templates (e.g., layouts). It provides an `HTML` method to execute templates with a `page.Page` data structure and write the output securely.
*   **`internal/transform`**: Implements custom Goldmark AST transformers and parsers. It manages internal markdown link rewriting (calculating relative paths), generating stubs for broken links, and parsing custom syntax like Emoji Kitchen shortcuts.
*   **`internal/asset`**: Responsible for copying static assets from the configured asset directory to the final output directory.
*   **`internal/search`**: Extracts plain text snippets from markdown content (stripping tags) and defines the data structure for search index items.
*   **`internal/taxonomy`**: Reads frontmatter tags across all content, generates tag index pages, and groups related content together.
*   **`internal/graph`**: Defines data structures for site topography (Nodes and Edges).
*   **`internal/ragexport`**: Provides tools to export the site's content, architecture, and configuration into bundled markdown files tailored for consumption by LLMs (RAG archives).
*   **`internal/config`**: Defines the central `Config` struct (loading from YAML, setting defaults, validating paths to prevent traversal).
*   **`internal/content`**: Walks the content directory, parses YAML frontmatter using `adrg/frontmatter`, normalizes tags, validates dates, and prepares the `FileMeta` map.
*   **`internal/stub`**: Contains logic for generating stub pages when a markdown file links to a non-existent internal destination.
*   **`internal/page`**: Defines the data model (`Page` struct) passed to Go HTML layout templates.
*   **`internal/sitedata`**: Handles serializing metadata about the generated site (e.g., writing `sitemap.xml`).
*   **`internal/markdown`**: Configures and instantiates the `goldmark.Markdown` engine, injecting the custom transformers and inline parsers.
*   **`internal/git`**: Provides utility wrappers around git commands (e.g., getting commit hashes or status).
*   **`internal/github`**: Contains logic for fetching data from the GitHub API (e.g., fetching release information).
*   **`internal/watcher`**: Implements `fsnotify` file watching for live-reloading during local development (`serve --watch`). It handles debouncing filesystem events and managing SSE connections for connected clients.

## Part 2: Micro-Improvements

Here are 5 high-ROI micro-improvements that can be implemented to enhance performance, maintainability, and code quality:

### 1. Pre-allocate slices where length is known
*   **Location:** `internal/generator/generator.go` inside `Build()` (Pass 2)
*   **Issue:** `keys` is appended to in a loop over `fileMap`:
    ```go
    var keys []string
    for k := range fileMap {
        keys = append(keys, k)
    }
    ```
*   **Improvement:** Pre-allocate the slice to avoid multiple reallocations, improving memory efficiency during builds with many files.
    ```go
    keys := make([]string, 0, len(fileMap))
    for k := range fileMap {
        keys = append(keys, k)
    }
    ```

### 2. Struct field alignment for better memory packing
*   **Location:** `internal/content/content.go` (`FileMeta` struct) and `internal/page/page.go` (`Page` struct)
*   **Issue:** Go structs map directly to memory. Ordering fields from largest to smallest (or grouping by types) can reduce padding bytes. Currently, smaller types (like `*bool`) are interleaved with `string` and `[]byte` types.
*   **Improvement:** Reorder struct fields (e.g., placing `[]byte`, `[]string`, and `string` fields together, and smaller primitive types at the end) to optimize memory alignment.

### 3. Add explicit error wrapping
*   **Location:** `internal/content/metadata.go` inside `GatherMetadata()`
*   **Issue:** Errors returned from `filepath.WalkDir` or `os.ReadFile` are passed up naked (e.g., `return err`).
*   **Improvement:** Wrap errors with context using `fmt.Errorf("failed to read file %s: %w", path, err)` to make debugging easier.

### 4. Optimize tag validation loop (avoid string builder overhead)
*   **Location:** `internal/content/metadata.go` inside `GatherMetadata()`
*   **Issue:** The tag validation loop uses `strings.Builder` on every single tag, iterating rune by rune, even if the tag is already completely valid (which is the majority case).
    ```go
    for _, tag := range matter.Tags {
        lower := strings.ToLower(tag)
        var sb strings.Builder
        // ... rune iteration ...
    }
    ```
*   **Improvement:** Check if normalization is needed first using a fast regex (`regexp.MustCompile("^[a-z0-9-]+$")`) or a simple loop. Only use the builder fallback if invalid characters are detected.

### 5. Remove redundant mutex locks for independent map access
*   **Location:** `internal/generator/generator.go` (inside the worker pool loop)
*   **Issue:** The worker pool locks `mu` multiple times per job to update shared maps (`g.Nodes`, `metaData`, etc.). Since each worker handles a unique `id` (key), these map writes are isolated per key, but Go's map implementation isn't safe for concurrent writes even to distinct keys.
*   **Improvement:** Instead of locking/unlocking the central `mu` multiple times in a single worker iteration, aggregate the updates into a local result struct within the worker, and perform a single lock to flush them, OR use `sync.Map` for cleaner concurrent handling of `metaData` and `g.Nodes`.
