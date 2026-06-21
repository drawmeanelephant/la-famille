---
title: "Terminal UI Guide"
author: "Jules"
---

# Interactive Terminal UI (TUI)

La Famille features an interactive, full-screen Terminal UI built with the `Bubbletea` framework. The TUI provides a visual, menu-driven alternative to the standard CLI commands, making it easier to manage your workspace and view project statistics.

## Launching the TUI

To launch the TUI, simply run the base command without any subcommands:

```bash
go run ./cmd/la-famille
```

The application will take over your terminal screen (using `tea.WithAltScreen()`) and present you with the main menu.

*You can exit the TUI at any time by pressing `q` or `Esc` on the main menu, or `ctrl+c` globally.*

## Main Menu Options

The main menu allows you to navigate through the core features of the application using the `up` and `down` arrow keys and pressing `Enter` to select.

### 1. Build Site
Executes the standard site generation pipeline. This is visually equivalent to running `go run ./cmd/la-famille build`. It processes your markdown files, creates the HTML output in the `public/` folder, and generates the necessary metadata graphs.

### 2. Serve Site
Starts the built-in local development server.
*   This will run an HTTP server pointing to your `public/` directory in the background.
*   While the server is running, the TUI displays a screen featuring an ASCII animation of Jules, the project mascot!
*   Press `q` or `Esc` to stop the server and return to the main menu.

### 3. Generate RAG Archive
Triggers the Retrieval-Augmented Generation (RAG) export logic. This extracts the content and structure of your site into specialized LLM-friendly formats located in the `internal/rag-archive/` directory.

### 4. Stats
Opens a statistics dashboard displaying insights about your generated site. The stats screen provides information on the total number of files processed, the build times, and the size of your RAG exports relative to standard LLM context windows.

## Mascot Integration
Keep an eye out for Jules! The TUI integrates ASCII graphics of the project's mascot to make long-running tasks (like serving the site locally) more enjoyable.
