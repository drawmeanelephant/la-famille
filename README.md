# La Famille 🐙

[![GitHub Repository](https://img.shields.io/badge/GitHub-Repository-blue?logo=github)](https://github.com/drawmeanelephant/la-famille/)

La Famille is a fast, feature-rich static site generator written in Go. It goes beyond simple markdown-to-HTML conversion by offering powerful developer tools, an interactive Terminal UI (TUI), and AI-ready RAG (Retrieval-Augmented Generation) exports.

This project is built and maintained primarily by **Jules** (AI assistant) alongside an eight-legged friend, Raoul(s) the Octopus. We take a "Jules-forward" approach to development. If you are opening a Pull Request, please make sure to tag Jules in the comments to keep the AI looped in.

## Features ✨

*   **Lightning-Fast Static Generation:** Converts Markdown content into clean, semantic HTML using the `goldmark` library.
*   **Interactive TUI:** A sleek Bubbletea-powered terminal interface for managing builds, serving the site locally, and viewing project stats.
*   **Robust CLI:** A powerful command-line interface built with `cobra` for tasks like initialization, building, serving, and RAG generation.
*   **RAG Export:** Native tools to extract your site's content and metadata into clean archives optimized for LLM context windows (`rag-system.md`, `rag-content.md`, etc.).
*   **Flexible Templating:** Support for multiple HTML layouts (e.g., standard, cyberpunk, minimal) easily overridden via YAML frontmatter.
*   **Built-in Local Server:** Instantly preview your site with `go run ./cmd/la-famille serve`.
*   **Smart Graphing:** Automatically generates `graph.json`, `backlinks.json`, and handles non-existent internal links by generating helpful stub pages.

## Quickstart 🚀

### Prerequisites
*   [Go](https://go.dev/doc/install) installed on your machine.

### Build & Run
To run the static site generator using the CLI:
```bash
go run ./cmd/la-famille build
```

To launch the interactive TUI:
```bash
go run ./cmd/la-famille tui
```

### TUI Navigation & Controls
The TUI uses standard, frictionless keybindings for easy navigation:
*   **Navigation:** Use `up`/`down` arrows or Unix-centric `j`/`k` primitives to move through the menus.
*   **Selection & Exit:** Press `Enter` or `Space` to execute a command. Press `q` or `Esc` to safely drop back to the main menu screen buffer.
*   **Active Server Views:** When you select "Serve Site" (or "Serve Site with Watch"), the TUI locks into an alternate screen buffer, displaying the dancing mascot animation (Raoul!). To gracefully tear down the network handle and exit back to the main menu, press `q` or `Esc`.

To serve the generated site locally (defaults to port 8080):
```bash
go run ./cmd/la-famille serve
```

## Documentation 📚

The commands above will get you started, but La Famille has a lot more to offer. For deep-dive guides on how to use all the features, please explore our documentation:

*   **[Setup & Getting Started](content/docs/setup.md)**
*   **[CLI Reference](content/docs/cli.md)**
*   **[Using the TUI](content/docs/tui.md)**
*   **[Templating Guide](content/docs/templates.md)**
*   **[RAG Export Guide](content/docs/rag.md)**
*   **[How the Generator Works](content/docs/generator.md)**

---
*Generated with ❤️ by Jules*
