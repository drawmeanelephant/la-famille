---
title: "Terminal UI Guide"
author: "Jules"
---

# Interactive Terminal UI (TUI)

La Famille features an interactive, full-screen Terminal UI built with the `Bubbletea` framework. The TUI provides a visual, menu-driven alternative to the standard CLI commands, making it easier to manage your workspace and view project statistics.

## Launching the TUI

To launch the TUI, use the `tui` subcommand:

```bash
go run ./cmd/la-famille tui
```

The application will take over your terminal screen (using `tea.WithAltScreen()`) and present you with the main menu.

*You can exit the TUI at any time by pressing `q` or `Esc` on the main menu, or `ctrl+c` globally.*

## Main Menu Options

The main menu allows you to navigate through the core features of the application using the `up` and `down` arrow keys and pressing `Enter` to select.

### 1. Build Site
Executes the standard site generation pipeline. This is visually equivalent to running `go run ./cmd/la-famille build`. It processes your markdown files, creates the HTML output in the `public/` folder, and generates the necessary metadata graphs.

### 2. RAG Export
Triggers the Retrieval-Augmented Generation (RAG) export logic. This extracts the content and structure of your site into specialized LLM-friendly formats located in the `rag-archive/` directory.

### 3. Serve Site
Starts the built-in local development server.
*   This will run an HTTP server pointing to your `public/` directory in the background.
*   While the server is running, the TUI displays a screen featuring an ASCII animation of Jules, the project mascot!
*   Press `q` or `Esc` to stop the server and return to the main menu.

### 4. Serve Site with Watch
Starts the built-in local development server and watches for file changes.
*   This will run an HTTP server pointing to your `public/` directory and automatically rebuild the site when content or templates change.
*   While the server is running, the TUI displays a screen featuring an ASCII animation of Jules, the project mascot.
*   Press `q` or `Esc` to stop the server and return to the main menu.

### 5. Stats
Displays a statistics dashboard with insights about your generated site, including:
*   Last build time (in milliseconds)
*   Total pages generated
*   Error count
*   RAG token estimations (approximated from exported RAG markdown bundle sizes)

This screen updates live automatically when using watch mode.

### 6. Just Raoul
Displays an animation of the project mascot.
*   This option simply shows a screen featuring an ASCII animation of Jules, the project mascot.
*   Press `q` or `Esc` to return to the main menu.

## Mascot Integration
Keep an eye out for Jules! The TUI integrates ASCII graphics of the project's mascot to make long-running tasks (like serving the site locally) more enjoyable.
