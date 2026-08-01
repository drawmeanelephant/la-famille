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

## 3. Initialize the Project

La Famille includes a helpful initialization command that sets up default configuration files for you. Run the following command from the root of the project:

```bash
go run ./cmd/la-famille init
```

This will create a `config.yaml` file in the root directory if one doesn't already exist. You can read more about what settings are available in the [Configuration Guide](config.md).

For a released binary and a site outside the current directory:

```bash
la-famille --project-root /path/to/site init
```

`init` installs the embedded default layout, required partials, and runtime
assets only when site-owned files are absent. Existing overrides are preserved.

## 4. Run the Static Site Generator

To process the markdown files in the `content/` directory and generate the static HTML site in the `public/` directory, use the build command:

```bash
go run ./cmd/la-famille build
```

The output directory is the complete publish artifact. Validate it before
uploading:

```bash
la-famille --project-root /path/to/site publish-check --output public
```

This step will parse your markdown files, process frontmatter, resolve links, generate graph data, and compile everything using the HTML layouts found in the `templates/` directory.

## 5. Serve the Site Locally

You don't need a separate web server to view your generated site! La Famille comes with a built-in HTTP server to serve the `public/` directory.

```bash
go run ./cmd/la-famille serve
```

By default, the server will start on port `8080`. Open your web browser and navigate to `http://localhost:8080` to see your new static site.

*Note: If you need to stop the server, simply press `Ctrl+C` in your terminal.*

GitHub Pages uses the Actions Pages artifact/deploy flow and uploads the whole
`public/` tree; it does not imply or require a `gh-pages` branch. Set the
`LA_FAMILLE_VERSION` repository variable to make the workflow download and
checksum-verify a released binary. With the variable empty, it uses the
clearly marked source-build fallback for development.

## 6. What's Next?

Now that you have the site running, here are a few things you can do next:

*   **Explore the TUI:** Try running `go run ./cmd/la-famille tui` to see the interactive Terminal UI. See the [TUI Guide](tui.md) for more details.
*   **Learn the CLI:** Read the [CLI Reference](cli.md) to discover all available flags and options.
*   **Design with Templates:** Find out how to change the look of your site using different layouts in the [Templating Guide](templates.md).
