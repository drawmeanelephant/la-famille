# Architectural Review: La Famille

## Part 1: Component Identification
- **internal/generator**: Orchestrates the static site build process by converting Markdown to HTML, coordinating parallel page processing, and accumulating site metadata.
- **internal/render**: Handles HTML rendering via Go templates, discovering layouts/partials, caching templates, and injecting WatchMode live-reload scripts.
- **internal/transform**: Parses and modifies the Markdown AST (e.g., resolving internal wikilinks via `LinkTransformer`, checking for missing stubs, and handling `EmojiKitchenParser` custom triggers).
- **internal/asset**: Manages copying static assets to the output directory while natively interpreting and respecting `.gitignore` patterns to prevent boundary breakouts.
- **internal/search**: Generates the search index payload by extracting and minifying clean text snippets from parsed Markdown content.
- **internal/taxonomy**: Manages tag taxonomies by generating index pages for each tag found across the site's content.
- **internal/graph**: Generates graph visualization JSON (`nodes` and `edges`) and handles backlink mappings for interconnected Markdown pages.
- **internal/ragexport**: Exports the repository and content files into combined, RAG-friendly markdown bundles for LLM consumption.
- **internal/config**: Loads, parses, and validates the primary configuration defining site settings and output paths.
- **internal/content**: Walks the content directory to read Markdown files, extracts YAML frontmatter (via `FileMeta`), and validates tags.
- **internal/stub**: Generates placeholder (stub) pages for missing internal wikilinks to ensure site navigability and backlink continuity.
- **internal/page**: Defines the core `Page` struct representing the data model passed into the HTML templates.
- **internal/sitedata**: Generates and writes site-wide data artifacts like `meta.json` and the `sitemap.xml`.
- **internal/markdown**: Configures the Goldmark Markdown rendering engine with custom AST transformers and extensions like GFM and typographer.
- **internal/git**: Provides Go wrappers for local git command execution (status, checkout, add, commit, push) via subprocesses.
- **internal/github**: Implements an HTTP client for interacting with the GitHub API to manage Pull Requests and check commit statuses.
- **internal/watcher**: Implements development file watching (`fsnotify`) to trigger incremental builds and handles Server-Sent Events (SSE) for live-reloading.

## Part 2: Micro-Improvements
1. **Pre-allocate slices for known lengths**
   - In `internal/generator/generator.go`, `var searchIndex []search.Item` can be pre-allocated using `searchIndex := make([]search.Item, 0, len(fileMap))` to eliminate reallocation overhead during the sequential build phase.
   - Similarly, in `internal/content/metadata.go`, `var normalizedTags []string` should be pre-allocated as `normalizedTags := make([]string, 0, len(matter.Tags))`.

2. **Pre-allocate `strings.Builder` with `Grow()`**
   - In `internal/sitedata/write.go`, `var sitemapBuilder strings.Builder` dynamically appends string content for every metadata key. Since the exact number of keys is known (`len(keys)`), pre-allocating the buffer with `sitemapBuilder.Grow(len(keys) * 100)` would effectively mitigate internal array reallocation overhead.

3. **Optimize loop structures and minimize allocations**
   - In `internal/asset/copy.go`, the `isIgnored` function repeatedly calls `strings.Split(slashPath, "/")`, allocating a slice of segments for every file processed. This can be optimized by progressively iterating with `strings.Index` or executing string prefix/suffix matching before evaluating the expensive split logic.
   - The function also iterates over `segments` sequentially multiple times. Combining the exact match check and `filepath.Match` check into a single linear loop pass will eliminate localized technical debt.

4. **Improve deferred error wrapping**
   - In `internal/asset/copy.go`'s `CopyFile`, the deferred `destination.Close()` explicitly captures its error as `cerr`, but assigns it to `err` without context. Upgrading this to `err = fmt.Errorf("failed to close destination file: %w", cerr)")` ensures critical file handle contexts aren't swallowed. Additionally, `destination.Sync()` is returned directly at the end of the function; wrapping its error would improve diagnostic tracing.
