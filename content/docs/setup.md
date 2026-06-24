---
title: "Getting Started Guide"
author: "Jules"
---

# Getting Started with La Famille

Welcome to La Famille! This guide will walk you through the process of setting up the project on your local machine, initializing your first workspace, and running the local development server.

## 1. Prerequisites

La Famille is written in Go. Before you can build or run the project, you need to have Go installed on your system.

*   **Install Go:** Head over to the official [Go Installation Guide](https://go.dev/doc/install) and download the appropriate installer for your operating system.
*   **Verify Installation:** Once installed, open your terminal and run `go version` to ensure it is correctly installed and in your PATH.

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

## 4. Run the Static Site Generator

To process the markdown files in the `content/` directory and generate the static HTML site in the `public/` directory, use the build command:

```bash
go run ./cmd/la-famille build
```

This step will parse your markdown files, process frontmatter, resolve links, generate graph data, and compile everything using the HTML layouts found in the `templates/` directory.

## 5. Serve the Site Locally

You don't need a separate web server to view your generated site! La Famille comes with a built-in HTTP server to serve the `public/` directory.

```bash
go run ./cmd/la-famille serve
```

By default, the server will start on port `8080`. Open your web browser and navigate to `http://localhost:8080` to see your new static site.

*Note: If you need to stop the server, simply press `Ctrl+C` in your terminal.*

## 6. What's Next?

Now that you have the site running, here are a few things you can do next:

*   **Explore the TUI:** Try running `go run ./cmd/la-famille` without any arguments to see the interactive Terminal UI. See the [TUI Guide](tui.md) for more details.
*   **Learn the CLI:** Read the [CLI Reference](cli.md) to discover all available flags and options.
*   **Design with Templates:** Find out how to change the look of your site using different layouts in the [Templating Guide](templates.md).
