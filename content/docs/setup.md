---
date: "2026-07-09"
title: "Getting Started Guide"
author: "Jules"
---

# Getting Started with La Famille

Welcome to La Famille! This guide will walk you through the process of setting up the project on your local machine, initializing your first workspace, and running the local development server.

## 0. Choose source or binary

Use a released archive for CI and publishing. Verify `SHA256SUMS`, run
`./la-famille --version`, and use `--project-root /path/to/site` from any
working directory. Use a source checkout when you are developing La Famille
itself or changing its templates and parser.

## 1. Prerequisites

La Famille is written in Go. Before you can build or run the project, you need to have Go installed on your system.

*   **Install Go:** Head over to the official [Go Installation Guide](https://go.dev/doc/install) and download Go 1.24 or newer (the project requires `go 1.24.0` / toolchain `go1.24.3`).
*   **Verify Installation & PATH:** Open your terminal and run `go version` to ensure it is correctly installed. Additionally, ensure your Go binary directory (`GOPATH/bin`) is included in your shell's `PATH`:
    ```bash
    export PATH="$PATH:$(go env GOPATH)/bin"
    ```

## 2. Clone the Repository

Clone the La Famille repository from GitHub to your local machine:

```bash
git clone https://github.com/drawmeanelephant/la-famille.git
cd la-famille
```

## 3. Unified Getting-Started Workflow Path

Follow this concise sequential path to initialize, author, validate, build, preview, and export your site:

### Step 1: Initialize Project (`init`)
Initialize default configuration, layout templates, and required assets:
```bash
go run ./cmd/la-famille init
```
*(For a released binary outside source: `la-famille --project-root /path/to/site init`)*

### Step 2: Scaffold Content (`new`)
Create new markdown posts or pages with pre-formatted YAML frontmatter:
```bash
go run ./cmd/la-famille new posts/first-post --title "First Post" --tags "welcome,guide" --categories "updates"
```

### Step 3: Validate Content & Asset Health (`check`)
Check frontmatter syntax, internal links, slug collisions, and asset references:
```bash
go run ./cmd/la-famille check --asset-health
```
Diagnostic findings include the active build version and commit metadata.

### Step 4: Build Static Site (`build`)
Compile markdown files and assets into static HTML in `public/`:
```bash
go run ./cmd/la-famille build
```

### Step 5: Serve Locally with Watch Mode (`serve --watch`)
Launch the local HTTP server and automatically rebuild on content changes:
```bash
go run ./cmd/la-famille serve --watch
```
Navigate to `http://localhost:8080` in your web browser to preview your site live.

### Step 6: Export RAG Context Bundles (`rag`)
Generate LLM-ready context archives (`rag-system.md`, `rag-content.md`):
```bash
go run ./cmd/la-famille rag
```

## 4. GitHub Pages Deployment

GitHub Pages uses the Actions Pages artifact/deploy flow and uploads the whole `public/` tree; it does not imply or require a `gh-pages` branch. Set the `LA_FAMILLE_VERSION` repository variable to make the workflow download and checksum-verify a released binary. With the variable empty, it uses the clearly marked source-build fallback for development.

## 5. What's Next?

* **Explore the TUI:** Run `go run ./cmd/la-famille tui` to launch the interactive Terminal UI. See the [TUI Guide](tui.md).
* **Learn the CLI:** Read the [CLI Reference](cli.md) to discover all available flags and subcommands.
* **Design with Templates:** Customize HTML layouts in the [Templating Guide](templates.md).
* **Ask This Site:** Learn about the on-device AI assistant in the [Ask Guide](ask.md).
