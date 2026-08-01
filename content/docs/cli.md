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

The runtime-independent identity commands are available from any directory:

```bash
la-famille --version
la-famille --version --json
```

`--json` is valid with `--version` and returns stable `version`, `commit`,
`build_date`, `target`, and `go_version` fields. Release builds set the first
three fields with `-X main.buildVersion=...`, `-X main.buildCommit=...`, and
`-X main.buildDate=...`.

Global path flags are resolved before the config is loaded:

* `--project-root` selects the site root. Relative content, template, asset,
  output, and RAG paths are resolved from it.
* `--config` selects a config file. Without `--project-root`, its directory is
  the project root. An explicit `--project-root` takes precedence.

The precedence order is explicit flags, then `config.yaml`, then the current
directory/defaults.

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

*   **Description:** Creates a default `config.yaml` file in the current directory. This is the first step when setting up a new site.
*   **Existing configuration:** If `config.yaml` is already there, `init` refuses rather than replacing it, so a second run cannot discard your `siteurl`, `output_dir` or theme.
*   **`--force` / `-f`:** Replaces an existing `config.yaml`, keeping the current one as `config.yaml.bak`. This is also how a `config.yaml` too broken to parse gets regenerated — the backup means the original is still there to read afterwards.

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
    *   `--asset-dir` (string): The directory containing static assets. Defaults to `assets`.
    *   `--site-url` / `--siteurl` (`-s`) (string): The public base URL of the site. Used for canonical links, `og:url`, and absolute URLs in the sitemap, feed, and Knowledge Graph page. Defaults to unset (root-relative URLs only).

*Example:* `go run ./cmd/la-famille build -c my_docs -o dist -t templates/custom.html`

After the build, a static Knowledge Graph Explorer page is also written to `<output>/graph/index.html` (default enabled). The explorer page is self-contained — opening it directly in a browser, or serving the `public/` directory with any static file server, works without any runtime backend. To opt out, set `graph_explorer: false` in `config.yaml`.

`init` installs the embedded default layout, required partial, and runtime
assets only when the corresponding site files are absent. Existing site files
remain the explicit override. Run `publish-check --output public` before
uploading an artifact to get a deterministic file manifest and validate local
HTML references.

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
*   **Flags:** `--output` selects the archive directory, `--project-root` selects the source project, and `--content`, `--asset-dir`, and `--template` select the source inputs. These flags allow CI to write directly to `public/rag-archive` without changing the checkout.

### `publish-check`

Validates a generated static artifact and prints every relative file path:

```bash
la-famille --project-root /path/to/site publish-check --output public
la-famille --project-root /path/to/site publish-check --output public --json
```

The check rejects an accidentally published `.la-famille-cache.json`, verifies
local `href`/`src` references, and requires the graph explorer's payload and
companion CSS/JS when `graph/index.html` is present.

### `pr`

Manages GitHub Pull Requests (Clear the Litterbox).

```bash
go run ./cmd/la-famille pr [command]
```

*   **Description:** Tools for managing automation PRs. The `sync` subcommand inspects open bot-authored pull requests and applies an explicit merge policy.
*   **`pr sync`:** Dry-run by default. Requires `GITHUB_TOKEN`. Mutations require `--apply`. Merges require the `litterbox-approved` label (configurable via `--required-label`). Zero checks are not accepted by default. Conflicts are not closed by default (`--close-conflicts`). Local working-tree publishing is off by default (`--publish-local-changes`). When `--base` is omitted, the repository `default_branch` is resolved via the GitHub API. This repository’s nightly workflow targets `master` explicitly. Jules CI verifies but does not merge.
*   **Flags (`pr sync`):**
    *   `--base` (string): Target base branch (empty → API default branch).
    *   `--apply` (bool): Perform mutations (default false / dry-run).
    *   `--required-label` (string): Label required to merge or close (default `litterbox-approved`).
    *   `--close-conflicts` (bool): Authorize closing conflicting PRs that pass identity gates.
    *   `--allow-no-checks` (bool): Allow merge when no check runs are reported.
    *   `--publish-local-changes` (bool): Authorize staging all local changes and opening a PR.
    *   `--bot-author` (strings): Bot author allowlist (defaults include Jules bots).
    *   `--head-prefix` (strings): Optional head ref prefix restrictions.
*   **See Also:** [Pull Request Management Guide](pr.md) for the full policy contract.

### `new`

Scaffolds a new Markdown content file with YAML frontmatter.

```bash
go run ./cmd/la-famille new <slug-or-filename> [flags]
```

*   **Description:** Creates a new Markdown file in the configured content directory (`content/` by default). Generates valid frontmatter with default or custom metadata, creating parent directories if needed.
*   **Flags:**
    *   `--title`, `-t` (string): Title of the page. Defaults to title-cased filename if omitted.
    *   `--tags` (strings): Comma-separated list or multiple instances of tags.
    *   `--categories` (strings): Comma-separated list or multiple instances of categories.
    *   `--layout` (string): Layout template name for the page.
    *   `--date` (string): Publication date in `YYYY-MM-DD` format. Defaults to today's date.
    *   `--force`, `-f` (bool): Force overwrite an existing file.
    *   `--content`, `-c` (string): Override target content directory.

*Example:* `go run ./cmd/la-famille new blog/my-first-post --title "My First Post" --tags "tech,go" --categories "news"`
