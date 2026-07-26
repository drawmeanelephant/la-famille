---
title: "TUI Architecture"
author: "La Famille maintainers"
date: "2026-07-26"
---

# TUI Architecture

`cmd/la-famille/tui.go` implements the Bubble Tea application behind `la-famille tui`. It is an operator console over the same generator, watcher, and RAG packages used by the CLI; it does not contain a second site-generation pipeline.

## State model

The TUI model tracks the active screen, menu cursor, build progress, latest build statistics, diagnostics, server, watcher cancellation, and terminal dimensions. Messages move the model between menu, working, stats, diagnostics, help, mascot, and serving states.

Commands are selected from the menu:

- Build Site
- Serve Site
- Toggle Watch Mode
- Stats
- Diagnostics
- RAG Export
- Just Raoul
- Help

`q` and `Esc` return from transient screens; `d` opens diagnostics; `c` clears diagnostics; `j`/`k` and arrow keys navigate the menu or diagnostic list.

## Serve and watch lifecycle

Selecting Serve Site always performs an initial build. A failed build records a diagnostic and prevents the HTTP server from starting. When watch mode is enabled, the TUI starts the file watcher after the successful build and sends build results back into the Bubble Tea program.

Server and watcher cancellation is centralized in `stopServing`. It is used for normal exit, Ctrl-C, and server failure so the TUI does not leave a watcher or HTTP listener behind. The server serves the configured output directory on loopback and exposes live reload only when watch mode is active.

## Diagnostics and progress

Build and server failures become diagnostics with recovery guidance. Successful builds update the stats dashboard, including cache state, page counts, health metrics, and RAG archive estimates. Warnings and errors are kept separate from the screen transition so the operator can inspect them after a failed or warning-producing operation.

## Mascot animation

The `Just Raoul` screen and serving screen share the `raoulFrames` animation. It is a six-frame ASCII animation with stable terminal-oriented proportions, changing eyes, body position, and tentacle choreography. The animation is intentionally local to the TUI and has no effect on generated sites.

## Testing the TUI

`cmd/la-famille/tui_test.go` tests model transitions and rendering without requiring a real interactive terminal. Important lifecycle coverage includes:

- menu selection and watch-mode toggling;
- initial-build failure during Serve Site;
- server shutdown and restart;
- diagnostic drawer navigation and clearing;
- build progress, warnings, and recovery guidance;
- mascot menu selection and animation-frame wraparound.

Use `go test ./cmd/la-famille` for focused feedback. The server lifecycle tests bind ephemeral loopback ports, so they need normal localhost access in restricted environments.
