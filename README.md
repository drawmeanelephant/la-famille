# La Famille 🐙

[![GitHub Repository](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/drawmeanelephant/la-famille/)

La Famille is a fast, feature-rich static site generator written in Go. It goes beyond simple markdown-to-HTML conversion by offering powerful developer tools, an interactive Terminal UI (TUI), and AI-ready RAG (Retrieval-Augmented Generation) exports.

This project is built and maintained primarily by **Jules** (AI assistant) alongside an eight-legged friend, Raoul(s) the Octopus. We take a "Jules-forward" approach to development. If you are opening a Pull Request, please make sure to tag Jules in the comments to keep the AI looped in.

## Features ✨

*   **Lightning-Fast Static Generation:** Converts Markdown content into clean, semantic HTML using the `goldmark` library.
*   **Interactive TUI:** A sleek Bubbletea-powered terminal interface for managing builds, serving the site locally, and viewing project stats.
*   **Robust CLI:** A powerful command-line interface built with `cobra` for tasks like initialization, building, serving, and RAG generation.
*   **RAG Export:** Native tools to extract your site's content and metadata into clean archives optimized for LLM context windows (`rag-system.md`, `rag-content.md`, etc.).
*   **Ask This Site (experimental):** A local, citation-grounded Q&A assistant that runs entirely on your machine. Binds only to loopback, never sends content off-device, and supports the Ollama daemon out of the box.
*   **Flexible Templating:** Support for multiple HTML layouts (e.g., standard, cyberpunk, minimal) easily overridden via YAML frontmatter.
*   **Built-in Local Server:** Instantly preview your site with `go run ./cmd/la-famille serve`.
*   **Smart Graphing:** Automatically generates `graph.json`, `backlinks.json`, and handles non-existent internal links by generating helpful stub pages.
*   **Interactive Knowledge Graph Explorer:** Every build emits a self-contained `/graph/index.html` page that visualizes the site as a directed graph — search by title, page ID, tag, category, or author, filter by render/raw/stub/orphan, jump into "focus mode" for a selected page plus its neighbors, and deep-link selections via `?node=`. No runtime server; just open the file. Disable with `graph_explorer: false` in `config.yaml`.

## Quickstart 🚀

### Choose a runtime

| Workflow | Use when | Required inputs |
| --- | --- | --- |
| Released binary | CI, GitHub Pages, or an operator who does not have the source checkout | A downloaded archive and its `SHA256SUMS` entry |
| Source checkout | Developing La Famille or changing templates/parser code | Go 1.24+ and this repository |

Released archives are self-contained: `--version` works offline from an empty
directory, and the default layout plus required graph/search assets are
embedded in the binary. Verify an archive before use, then select a site from
any working directory:

```bash
sha256sum --check SHA256SUMS --ignore-missing
./la-famille --version --json
./la-famille --project-root /path/to/site init
./la-famille --project-root /path/to/site build
```

Relative paths are resolved from `--project-root`. Precedence is explicit CLI
flags, then `config.yaml`, then the selected project root/current directory.
`public/` is the complete static publish artifact; the build cache is kept
beside the project and is never intended for hosting.

### Prerequisites
*   **Go Toolchain:** [Go 1.24 or newer](https://go.dev/doc/install) (the project `go.mod` specifies `go 1.24.0` with `toolchain go1.24.3`). Verify your installed version with `go version`.
*   **Go Installation & Binary Path:** Ensure `go` is in your `PATH`. When installing binaries via Go (e.g. `go install`), binaries are placed in `$(go env GOPATH)/bin` (typically `~/go/bin`). Ensure `$(go env GOPATH)/bin` is added to your shell's `PATH`:
    ```bash
    export PATH="$PATH:$(go env GOPATH)/bin"
    ```
*   **Source Code:** Clone this repository to your local machine:
    ```bash
    git clone https://github.com/drawmeanelephant/la-famille.git
    cd la-famille
    ```
    *Note for users who downloaded a release archive or source tarball instead of cloning:* Extract the source archive, navigate into the extracted root directory, and run the commands below. `go run ./cmd/la-famille` executes directly against the local module tree.

### Build & Run
To run the static site generator using the CLI:
```bash
go run ./cmd/la-famille build
```

The equivalent released-binary command is:

```bash
la-famille --project-root /path/to/site build
```

To launch the interactive TUI:
```bash
go run ./cmd/la-famille tui
```

To export RAG bundles:
```bash
go run ./cmd/la-famille rag
```

For an artifact-only Pages build, write the archive directly below the output
tree without changing the checkout:

```bash
la-famille --project-root /path/to/site rag --output /path/to/site/public/rag-archive
```

> After `go run ./cmd/la-famille build`, open `public/graph/index.html` to explore the site's relationships interactively. The page is fully static — it loads `graph.json`, `meta.json`, and `backlinks.json` from the same output directory via relative fetches.

> **Note:** `rag-archive/` is generated by `go run ./cmd/la-famille rag`. It is intentionally ignored and must not be edited or committed. The Pages workflow publishes a copy under `public/rag-archive/`; it uploads the complete `public/` directory as an Actions artifact and does not use a `gh-pages` branch.

### TUI Navigation & Controls
The TUI uses standard, frictionless keybindings for easy navigation:
*   **Navigation:** Use `up`/`down` arrows or Unix-centric `j`/`k` primitives to move through the menus.
*   **Selection & Exit:** Press `Enter` or `Space` to execute a command. Press `q` or `Esc` to safely drop back to the main menu screen buffer.
*   **Active Server Views:** When you select "Serve Site" (or "Serve Site with Watch"), the TUI locks into an alternate screen buffer, displaying the dancing mascot animation (Raoul!). To gracefully tear down the network handle and exit back to the main menu, press `q` or `Esc`.

To serve the generated site locally (defaults to port 8080):
```bash
go run ./cmd/la-famille serve
```

To launch the local-first **Ask This Site** assistant against your corpus:
```bash
go run ./cmd/la-famille rag                # refresh the corpus first
go run ./cmd/la-famille ask --model llama3.2  # then serve the assistant on 127.0.0.1:8090
```

> **Note:** `ask` is opt-in and experimental. It binds only to your loopback address, never sends your content off the machine, and never logs prompts or answers by default. See [content/docs/ask.md](content/docs/ask.md) for the full privacy and architecture notes.

## Documentation 📚

The commands above will get you started, but La Famille has a lot more to offer. For deep-dive guides on how to use all the features, please explore our documentation:

*   **[Setup & Getting Started](content/docs/setup.md)**
*   **[CLI Reference](content/docs/cli.md)**
*   **[Using the TUI](content/docs/tui.md)**
*   **[Templating Guide](content/docs/templates.md)**
*   **[RAG Export Guide](content/docs/rag.md)**
*   **[Ask This Site Guide](content/docs/ask.md)**
*   **[How the Generator Works](content/docs/generator.md)**

---
*Generated with ❤️ by Jules*


### CI/Testing
La Famille uses a comprehensive automated testing pipeline. All code merges are gated by passing `go test` and static analysis provided by `golangci-lint` to ensure security and code quality.

## GitHub Action 🤖

You can easily build your La Famille site in CI using our GitHub Action:

```yaml
steps:
  - uses: actions/checkout@v4
  - name: Build with La Famille
    uses: drawmeanelephant/la-famille@main
```

### Configurable Inputs

All inputs are optional and fall back to sensible defaults:

```yaml
steps:
  - uses: actions/checkout@v4
  - name: Build with La Famille
    uses: drawmeanelephant/la-famille@main
    with:
      project-root: '.'
      content-dir: 'content'
      output-dir: 'public'
      asset-dir: 'assets'
      template: 'templates/layout.html'
      site-url: 'https://example.github.io/my-site'
      # Pin a release in production; omit for the source-build fallback.
      release-version: 'v1.2.3'
```
