# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

- **`internal/generator`**: Build orchestration, coordinating content parsing, rendering, taxonomy generation, search index building, and RSS feed generation.
- **`internal/render`**: HTML template parsing, caching, and rendering using standard `html/template`. Contains logic for WatchMode live-reloading injection.
- **`internal/transform`**: Markdown link transformation (AST manipulation), URL generation, and emoji kitchen logic.
- **`internal/asset`**: Static asset copying with local `.gitignore` pattern matching to exclude specific files from the output.
- **`internal/search`**: Extraction of snippets and headings from Markdown, stripping HTML tags, and generating minified JSON for client-side search.
- **`internal/taxonomy`**: Extraction and grouping of taxonomy terms (tags, categories) from content metadata.
- **`internal/graph`**: Constructing directed graphs of content relationships (backlinks), writing adjacency lists for the graph explorer UI.
- **`internal/ragexport`**: Exporting site content into clean, aggregated Markdown formats suitable for RAG (Retrieval-Augmented Generation).
- **`internal/config`**: Configuration loading and validation, providing defaults for site-wide settings.
- **`internal/content`**: Parsing Markdown files, extracting YAML frontmatter, and handling file metadata.
- **`internal/stub`**: Generating missing or stub files (e.g., taxonomy term pages) dynamically.
- **`internal/page`**: Template models, preparing data structures passed to HTML templates during rendering.
- **`internal/sitedata`**: Writing site-wide metadata and discovery files like sitemaps.
- **`internal/markdown`**: Goldmark initialization and Markdown-to-HTML conversion configuration.
- **`internal/git`**: Local git repository status checks.
- **`internal/github`**: GitHub API interactions, syncing data and checking PR statuses.
- **`internal/watcher`**: File system watching for development mode and SSE (Server-Sent Events) based live-reloading.

## Part 2: Micro-Improvements

### 1. Optimizing search heading extraction pre-allocation (`internal/search/search.go`)
In `ExtractHeadings` (`internal/search/search.go`), the `headings` slice is appended to inside a loop that iterates over all lines in the file. Since the number of headings is typically a fraction of the total lines, pre-allocating this slice with a reasonable heuristic (e.g., `make([]string, 0, len(lines)/10)` or a small constant) reduces dynamic array resizing and GC pressure during the build phase.

### 2. Struct memory packing in search index structs (`internal/retrieval/loader.go` & `internal/search/search.go`)
Structures like `searchIndexEntry` and `search.Item` map to JSON output and currently have fields like `Title string`, `URL string`, `Tags []string`, etc. While Go aligns these sequentially, reviewing struct definitions across the internal packages (like `retrieval/loader.go`'s `metaEntry`) to ensure 8-byte primitives (like `int` or `float64`) are grouped tightly together rather than interleaved with boolean flags or smaller types can save padding overhead across thousands of indexed documents.

### 3. Refactoring taxonomy items grouping array scans to map lookups (`internal/taxonomy/taxonomy.go`)
When collecting unique taxonomy items (like tags or categories) during the generation phase, the current implementation iterates to ensure uniqueness before appending (`if cat != "" && !taxonomySeen[cat] { ... taxonomyTerms = append(...) }`). By relying entirely on map sets and converting them to slices once at the end, we can slightly reduce complexity and potential linear search debt when combining taxonomy paths.

### 4. Better context wrapping for path errors in build generation (`internal/generator/generator.go`)
Many path resolution errors in the output directory logic use `fmt.Errorf("output path collision: %s and %s...", ...)`. While some use `%w` for `err` wrapping, others return raw strings. Wrapping all nested filesystem errors (like `os.MkdirAll` or `os.WriteFile` errors in `generator.go`) explicitly with `%w` using contextual messages (e.g. `fmt.Errorf("failed to create output dir %q: %w", filepath.Dir(outPath), err)`) will drastically improve build debugging.

