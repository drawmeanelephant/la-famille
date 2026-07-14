# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

The `internal/` directory contains the core application logic, modularized by responsibility. Here is the architectural map of the major components:

- **`internal/generator`**: The orchestration engine for the static site build process. It coordinates metadata gathering, multi-threaded Markdown conversion, graph generation, search indexing, and rendering.
- **`internal/render`**: Manages HTML output generation using Go templates. It includes a double-checked caching mechanism for layouts and handles WatchMode live-reload script injection.
- **`internal/transform`**: Handles Markdown AST transformations via Goldmark. Crucially, its `LinkTransformer` resolves internal links, tracks backlink edges for the graph, and detects missing files to be stubbed. Also includes specialized parsers like `EmojiKitchenParser`.
- **`internal/asset`**: Responsible for syncing static assets from the asset directory to the public output directory. It natively parses `.gitignore` rules to exclude non-public files without relying on external Git binaries.
- **`internal/search`**: Builds the search index data structure (`search.json`). Implements an `ExtractSnippet` utility to strip markdown and HTML noise, producing clean, minified snippet strings.
- **`internal/taxonomy`**: Handles tag extraction, validation, and generates localized HTML pages grouping content by their assigned tags.
- **`internal/graph`**: Writes deterministic JSON artifacts mapping the site's link graph (nodes and directional edges) and backlinks for client-side consumption.
- **`internal/ragexport`**: Provides functionality to bundle the project files into consolidated, RAG-friendly markdown exports for downstream AI and LLM consumption.
- **`internal/config`**: Defines the `Config` data model, parses YAML configuration files, and applies safe fallback defaults for the site.
- **`internal/content`**: Walks the source directory to parse frontmatter metadata from Markdown files (`GatherMetadata`), validates formatting, and tracks rendering rules.
- **`internal/stub`**: Generates missing link placeholder pages. Resolves broken internal links automatically to maintain graph integrity and prevent 404 dead-ends.
- **`internal/page`**: Defines the central `Page` struct model passed to HTML templates, encapsulating site config, frontmatter metadata, and the rendered content payload.
- **`internal/sitedata`**: Writes aggregate JSON metadata representing the site's structure to the output directory, useful for sitemaps or client-side indexing.
- **`internal/markdown`**: Configures and instantiates the `goldmark` Markdown engine, registering extensions like GFM, Typography, and custom parsers/transformers.
- **`internal/git`**: Wraps local Git repository operations, likely to extract commit metadata, modification dates, or author information for content.
- **`internal/github`**: Handles external GitHub API interactions, such as repository syncing operations and pull request checks.
- **`internal/watcher`**: Implements context-aware file system watching via `fsnotify`. Monitors content, templates, and assets to trigger debounced live rebuilds and Server-Sent Events (SSE) reloads.

## Part 2: Micro-Improvements

Here are 4 high-ROI, localized micro-improvements to enhance performance, maintainability, and code hygiene without major architectural shifts:

1. **Better Error Wrapping in File System Operations**
   - *Context:* Functions in `internal/watcher/watcher.go` (like `filepath.WalkDir` callbacks) and `internal/asset/copy.go` (like `os.MkdirAll`) often return raw errors (e.g., `return err`).
   - *Improvement:* Wrap these errors with context using `fmt.Errorf("failed to walk directory %q: %w", path, err)`. This will significantly improve debugging by pinpointing exactly which directory or file caused the underlying I/O failure.

2. **Reduce Linear Scans in `transform.LinkTransformer`**
   - *Context:* In `internal/transform/link_transformer.go`, deduplicating missing files relies on a linear slice scan (`for _, p := range parents`).
   - *Improvement:* Change the tracking structure for `MissingFiles` from `map[string][]string` to `map[string]map[string]struct{}` (or a similar set abstraction). This converts an $O(N)$ linear scan into an $O(1)$ map lookup, cleaning up localized technical debt and saving CPU cycles during large graph traversals.

3. **Pre-allocation Optimization in `generator.Build`**
   - *Context:* In `internal/generator/generator.go`, the final `searchIndex` slice is populated by iteratively appending items from `searchIndexItems` that have a valid URL.
   - *Improvement:* Pre-allocate `searchIndex` with a known capacity before the loop: `searchIndex := make([]search.Item, 0, len(searchIndexItems))`. This prevents dynamic slice reallocation overhead during the append operations.

4. **Struct Field Alignment in `search.Item`**
   - *Context:* In `internal/search/search.go`, the `Item` struct places a 24-byte slice (`Tags []string`) between 16-byte strings (`URL` and `Snippet`).
   - *Improvement:* Move the `Tags` slice to the first field of the struct (or group it with other slices/pointers). This minimizes padding bytes generated by the compiler to align fields on 64-bit architectures, improving memory packing for large search indexes.
