---
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

*Tip: Running the CLI without any commands will launch the interactive [Terminal UI (TUI)](tui.md).*

---

## Commands

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

*Example:* `go run ./cmd/la-famille build -c my_docs -o dist -t templates/custom.html`

### `serve`

Starts a local HTTP server to preview your generated site.

```bash
go run ./cmd/la-famille serve [flags]
```

*   **Description:** Launches a local web server (using Go's `http.FileServer`) pointing to the configured output directory (usually `public/`). This allows you to instantly preview your generated site in your web browser.
*   **Flags:**
    *   `--port`, `-p` (int): The port to run the server on. Overrides the value set in `config.yaml`. Defaults to `8080` if not set in config.

*Example:* `go run ./cmd/la-famille serve -p 3000`

### `rag`

Generates a Retrieval-Augmented Generation (RAG) archive.

```bash
go run ./cmd/la-famille rag
```

*   **Description:** Scans the generated output and metadata to construct an optimized dataset designed for Large Language Models (LLMs). This exports files like `rag-system.md`, `rag-config.md`, and `rag-content.md` into the `internal/rag-archive/` directory. See the [RAG Export Guide](rag.md) for more details.

### `pr`

Manages GitHub Pull Requests.

```bash
go run ./cmd/la-famille pr [command]
```

*   **Description:** A suite of tools for managing PRs, particularly useful for clearing out stale PRs created by automation agents via the `sync` subcommand.
*   **See Also:** [Pull Request Management Guide](pr.md) for full details and configuration requirements.
