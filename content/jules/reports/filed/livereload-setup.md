---
title: Execution Log - LiveReload Setup
author: "Jules"
date: "2026-06-29"
---

# Execution Log: Zero-Dependency LiveReload

**Status:** Completed

## Actions Taken
1. Added `WatchMode` flag to `config.Config` (in `internal/config/config.go`) to inform downstream logic whether to inject livereload scripts.
2. Created a minimal Server-Sent Events (SSE) server in `internal/watcher/livereload.go`, exporting `LiveReloadHandler` and `BroadcastReload()`.
3. Updated `internal/watcher/watcher.go` to call `BroadcastReload()` after every successful generator build.
4. Updated `internal/render/render.go` to intercept `</body>` and inject the vanilla JavaScript SSE listener when `WatchMode` is true.
5. Updated `cmd/la-famille/main.go` and `cmd/la-famille/tui.go` so that the web servers in watch mode use `http.ServeMux` to serve both the files and the `/livereload` endpoint.

## Notes
- Relying exclusively on standard library `net/http` means zero external dependencies.
- The JavaScript payload is resilient; it fails silently if it drops the connection.
