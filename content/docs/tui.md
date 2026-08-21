---
date: "2026-07-09"
title: "Terminal UI Guide"
author: "Jules"
---

# Interactive Terminal UI (TUI)

La Famille features an interactive, full-screen Terminal UI built with the `Bubbletea` framework. The TUI provides a visual, menu-driven alternative to the standard CLI commands, making it easier to manage your workspace and view project statistics.

<div class="bg-info/10 border-l-4 border-info p-4 my-6">
  <strong>Tip:</strong> The TUI is fully interactive and uses keyboard navigation. It acts as a wrapper around the core CLI commands, giving you a more visual experience.
</div>

## Launching the TUI

To launch the TUI, use the `tui` subcommand:

```bash
go run ./cmd/la-famille tui
```

The application will take over your terminal screen (using `tea.WithAltScreen()`) and present you with the main menu.

*You can exit the TUI at any time by pressing `q` or `Esc` on the main menu, or `ctrl+c` globally.*

## Main Menu Options

The main menu allows you to navigate through the core features of the application using the `up` and `down` arrow keys and pressing `Enter` to select.

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">1. Build Site</h3>
    <p class="text-sm">Executes the standard site generation pipeline. This is visually equivalent to running <code>go run ./cmd/la-famille build</code>. It processes your markdown files, creates the HTML output in the <code>public/</code> folder, and generates the necessary metadata graphs.</p>
  </div>
</div>

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">2. RAG Export</h3>
    <p class="text-sm">Triggers the Retrieval-Augmented Generation (RAG) export logic. This extracts the content and structure of your site into specialized LLM-friendly formats located in the <code>rag-archive/</code> directory.</p>
  </div>
</div>

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">3. Serve Site</h3>
    <p class="text-sm">Starts the built-in local development server.</p>
    <ul class="text-sm list-disc pl-4 mt-2">
      <li>This will run an HTTP server pointing to your <code>public/</code> directory in the background.</li>
      <li>While the server is running, the TUI displays a screen featuring an ASCII animation of Jules, the project mascot!</li>
      <li>Press <code>q</code> or <code>Esc</code> to stop the server and return to the main menu.</li>
    </ul>
  </div>
</div>

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">4. Serve Site with Watch</h3>
    <p class="text-sm">Starts the built-in local development server and watches for file changes.</p>
    <ul class="text-sm list-disc pl-4 mt-2">
      <li>This will run an HTTP server pointing to your <code>public/</code> directory and automatically rebuild the site when content or templates change.</li>
      <li>While the server is running, the TUI displays a screen featuring an ASCII animation of Jules, the project mascot.</li>
      <li>Press <code>q</code> or <code>Esc</code> to stop the server and return to the main menu.</li>
    </ul>
  </div>
</div>

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">5. Stats</h3>
    <p class="text-sm">Displays a statistics dashboard with insights about your generated site, including:</p>
    <ul class="text-sm list-disc pl-4 mt-2">
      <li>Last build time (in milliseconds)</li>
      <li>Total pages generated</li>
      <li>Error count</li>
      <li>RAG token estimations (approximated from exported RAG markdown bundle sizes)</li>
    </ul>
    <p class="text-sm mt-2 italic">This screen updates live automatically when using watch mode.</p>
  </div>
</div>

<div class="card bg-base-200 shadow-sm border border-base-300">
  <div class="card-body p-4">
    <h3 class="card-title text-lg mb-2">6. Just Raoul</h3>
    <p class="text-sm">Displays an animation of the project mascot.</p>
    <ul class="text-sm list-disc pl-4 mt-2">
      <li>This option simply shows a screen featuring an ASCII animation of Jules, the project mascot.</li>
      <li>Press <code>q</code> or <code>Esc</code> to return to the main menu.</li>
    </ul>
  </div>
</div>

</div>

## Keybindings & Discoverability

Every screen shows its valid keys in the footer (e.g., menu: `↑/k, ↓/j: Navigate • Enter/Space: Select • m: Menu • d: Diagnostics • w: Watch • ?: Help • q: Quit`). The selected menu item uses a `focus-visible` highlight (underline + background) following `templates/layout.html` accessibility patterns.

| Key | Action | Where |
| --- | --- | --- |
| `↑` / `k` | Move selection up | Menu, Diagnostics |
| `↓` / `j` | Move selection down | Menu, Diagnostics |
| `Enter` / `Space` | Select command | Menu (when open) |
| `m` | Toggle octoburger menu open/closed | Menu |
| `d` | Toggle Diagnostics drawer (warnings → `Next:` hints) | Any screen |
| `w` | Toggle Watch Mode (auto-rebuild) | Menu / Stats / Diagnostics / Help |
| `?` / `h` | Open/close Help & Keybindings legend (one-line footer expands to full help) | Any screen |
| `c` | Clear diagnostics | Diagnostics drawer |
| `q` / `Esc` | Go back to main menu / close menu / quit (Esc closes menu, `q` on menu quits) | Any screen |
| `Ctrl+C` | Force quit application | Any screen |

Press `?` or `h` from any screen to expand the footer help line into the full `Help` screen; press `?`, `h`, `Esc`, or `q` to return to the previous screen. Press `w` to toggle Watch Mode without opening the menu — the dashboard and serve screens reflect the state (`Watch Mode: ENABLED/DISABLED`).

## Diagnostics → Next Action

The Diagnostics drawer (`d`) is not a dead end — each finding links to the next CLI action:

*   **Broken internal link** → `la-famille check`
*   **Missing asset / large raster / suspicious image / case mismatch** → `la-famille check --asset-health`
*   **Frontmatter parse or validation warning** → `fix frontmatter in <path>`
*   **Case-only output collision** → `see docs/issues-420-422.md` (same file on case-insensitive filesystem; rename one path)

Warnings from the build (frontmatter fallback to raw markdown, case-only collision notices) are propagated into the drawer as `warning` entries with `Next:` hints and appear in `View` as `Next: <hint>` beneath each diagnostic plus `Source:` and `Action:` guidance.

## Mascot Integration
Keep an eye out for Jules! The TUI integrates ASCII graphics of the project's mascot to make long-running tasks (like serving the site locally) more enjoyable.
