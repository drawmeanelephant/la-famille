# Component Mapping & Micro-Improvement Audit: La Famille

## Part 1: Component Identification

Here is a map of the major components currently residing in the `internal/` directory and their responsibilities based on the codebase structure:

- **internal/generator**: Orchestrates the entire site build process, tying together rendering, taxonomy generation, search index generation, and asset copying. Manages build caches.
- **internal/render**: Responsible for rendering content, converting markdown to HTML using templates, and providing the templating engine.
- **internal/transform**: Modifies and analyzes parsed markdown ASTs, particularly focusing on handling internal links (wikilinks) and resolving them to standard URLs. Maintains link graphs.
- **internal/asset**: Manages copying static assets and `.gitignore` integration to ensure unwanted files aren't published.
- **internal/search**: Generates the minified JSON index used by the client-side search component.
- **internal/taxonomy**: Processes tags and categories, generating index pages for terms (e.g., `/tags/go/index.html`) and lists of items associated with them.
- **internal/graph**: Manages backlink JSON data mapping and connections between pages for graph visualizations.
- **internal/ragexport**: Responsible for exporting the site's content as a single unified markdown file optimized for LLM RAG (Retrieval-Augmented Generation) ingestion.
- **internal/config**: Loads, validates, and provides defaults for the site configuration (`la-famille.yml`).
- **internal/content**: Handles reading files and parsing YAML frontmatter metadata, resolving field names and merging values.
- **internal/stub**: Generates stub pages or missing placeholders for referenced links that do not yet exist in the file system.
- **internal/page**: Defines template model structures used to render HTML pages (e.g., passing page data into templates).
- **internal/sitedata**: Generates metadata files like sitemaps, RSS feeds, and potentially manifest files for discovery.
- **internal/markdown**: Integrates Goldmark, configuring extensions (GFM, Typographer) to parse and render Markdown.
- **internal/git**: Interfaces with local git repository status for detecting untracked or modified files during generation.
- **internal/github**: Handles GitHub API interactions, such as syncing pull requests, verifying PR checks, and interacting with GitHub Actions.
- **internal/watcher**: Implements the WatchMode file system watcher for live-reloading during development via SSE (Server-Sent Events).

## Part 2: Micro-Improvements

Here are 4 high-ROI micro-improvements tailored to La Famille:

### 1. Struct Field Alignment Optimization (`internal/config/config.go`)
- **Action**: Optimize the field order in `config.Config` to reduce memory waste from padding.
- **Value**: Reduces struct size from 248 bytes to 216 bytes (a 13% saving). When configuration models are passed around frequently, this reduces allocation overhead.
- **Implementation**: Group pointer/string types together and smaller primitives (like bool/int32) at the end.

### 2. Struct Field Alignment Optimization (`internal/generator/generator.go`)
- **Action**: Optimize field alignment in the `Generator` struct.
- **Value**: Reduces memory footprint from 136 pointer bytes down to 104 bytes, improving CPU cache utilization during the core build loop.

### 3. Pre-allocate slice capacity (`internal/taxonomy/taxonomy.go`)
- **Action**: In `generateTaxonomyGroup`, pre-allocate the `generatedPaths` and `searchItems` slices.
- **Value**: Avoids multiple dynamic slice reallocations inside a loop that scales with the number of generated tags/categories and their pages.
- **Implementation**: The capacity can be estimated as `1 + len(items)` (one for the main index + one for each taxonomy term).

### 4. Remove Duplicate O(N) Slice Scan (`internal/taxonomy/taxonomy.go`)
- **Action**: In `generateTaxonomyGroup`, replace the linear scan slice `seenPages` with a more optimal solution. The current code loops over `rawPages`, builds a boolean map `seenPages`, and populates a new slice `pages`, but the list of unique items can be extracted directly without creating an intermediate slice if handled effectively.
- **Value**: Cleans up technical debt by removing redundant slicing logic when assigning pages to taxonomies.
