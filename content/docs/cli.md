---
date: "2026-07-09"
title: "CLI Reference"
author: "Jules"
---

# Command Line Interface Reference

La Famille is equipped with a robust command-line interface (CLI) powered by the `cobra` library. The CLI provides commands for initializing projects, generating the site, serving content locally, and exporting AI-ready datasets.

## Global Execution

To execute the CLI, run the compiled binary or use `go run` targeting the package directory:

```bash
go run ./cmd/la-famille [command] [flags]
```

*Tip: Running the CLI with the `tui` subcommand will launch the interactive [Terminal UI (TUI)](tui.md).*

---

## Commands

### `tui`

Launches the semi-graphical user interface.

```bash
go run ./cmd/la-famille tui
```

*   **Description:** Starts the interactive Bubbletea Terminal UI. This provides a menu-driven interface to build, serve, export RAG data, and view project stats. See the [Terminal UI Guide](tui.md) for more details.

### `init`

Initializes a new La Famille workspace.

```bash
go run ./cmd/la-famille init
```

*   **Description:** Creates a default `config.yaml` file in the current directory if one does not already exist. This is the first step when setting up a new site.

### `build`

Generates the static site from your Markdown files.

```bash
go run ./cmd/la-famille build [flags]
```

*   **Description:** Parses the Markdown files in the content directory, processes frontmatter, handles link resolution, sanitizes HTML, and writes the final output (HTML files, `graph.json`, `backlinks.json`, `meta.json`) to the output directory.
*   **Flags:**
    *   `--content`, `-c` (string): The path to the directory containing your Markdown source files. Defaults to `content`.
    *   `--output`, `-o` (string): The path to the directory where the generated HTML should be placed. Defaults to `public`.
    *   `--template`, `-t` (string): The path to the default HTML layout template to use. Defaults to `templates/layout.html`.
    *   `--site-url` / `--siteurl` (`-s`) (string): The public base URL of the site. Used for canonical links, `og:url`, and absolute URLs in the sitemap, feed, and Knowledge Graph page. Defaults to unset (root-relative URLs only).

*Example:* `go run ./cmd/la-famille build -c my_docs -o dist -t templates/custom.html`

After the build, a static Knowledge Graph Explorer page is also written to `<output>/graph/index.html` (default enabled). The explorer page is self-contained — opening it directly in a browser, or serving the `public/` directory with any static file server, works without any runtime backend. To opt out, set `graph_explorer: false` in `config.yaml`.

### `serve`

Starts a local HTTP server to preview your generated site.

```bash
go run ./cmd/la-famille serve [flags]
```

*   **Description:** Launches a local web server (using Go's `http.FileServer`) pointing to the configured output directory (usually `public/`). This allows you to instantly preview your generated site in your web browser.
*   **Flags:**
    *   `--port`, `-p` (int): The port to run the server on. Overrides the value set in `config.yaml`. Defaults to `8080` if not set in config.
    *   `--watch`, `-w` (bool): Watch for file changes and auto-rebuild.

*Example:* `go run ./cmd/la-famille serve -p 3000 -w`

### `rag`

Generates a Retrieval-Augmented Generation (RAG) archive.

```bash
go run ./cmd/la-famille rag
```

*   **Description:** Scans the generated output and metadata to construct an optimized dataset designed for Large Language Models (LLMs). This exports files like `rag-system.md`, `rag-config.md`, and `rag-content.md` into the `rag-archive/` directory. See the [RAG Export Guide](rag.md) for more details.

### `pr`

Manages GitHub Pull Requests.

```bash
go run ./cmd/la-famille pr [command]
```

*   **Description:** A suite of tools for managing PRs, particularly useful for clearing out stale PRs created by automation agents via the `sync` subcommand.
*   **See Also:** [Pull Request Management Guide](pr.md) for full details and configuration requirements.

### `new`

Scaffolds a new Markdown content file with YAML frontmatter.

```bash
go run ./cmd/la-famille new <slug-or-filename> [flags]
```

*   **Description:** Creates a new Markdown file in the configured content directory (`content/` by default). Generates valid frontmatter with default or custom metadata, creating parent directories if needed.
*   **Flags:**
    *   `--title`, `-t` (string): Title of the page. Defaults to title-cased filename if omitted.
    *   `--tags` (strings): Comma-separated list or multiple instances of tags.
    *   `--layout` (string): Layout template name for the page.
    *   `--date` (string): Publication date in `YYYY-MM-DD` format. Defaults to today's date.
    *   `--force`, `-f` (bool): Force overwrite an existing file.
    *   `--content`, `-c` (string): Override target content directory.

*Example:* `go run ./cmd/la-famille new blog/my-first-post --title "My First Post" --tags "tech,go"`

