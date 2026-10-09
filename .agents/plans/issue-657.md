# Issue #657 — TUI Serve ignores siteurl base path

## Task ID
`issue-657`

## Problem
`cmd/la-famille/tui.go` mounts `http.Dir(OutputDir)` at `/` and registers only
`/livereload`. When `siteurl` carries a subpath (e.g. `https://example.com/repo`),
rendered pages use `/repo/...` URLs, so the TUI preview 404s every page and the
`/repo/livereload` endpoint the injected script connects to. The non-TUI `serve`
command already mounts under `cfg.BasePath()` correctly (fixed in #528,
`cmd/la-famille/main.go:405-429`).

## Approach
- Extract the TUI's ad-hoc mux construction into a helper `serveMux(cfg, watchEnabled)`
  that mirrors the `main.go` serve logic exactly:
  - `base := cfg.BasePath()`; when non-empty and not `/`, strip the prefix with
    `http.StripPrefix`, mount the file server at `cleanBase+"/"`, and redirect
    `/` → `cleanBase+"/"` (other root paths 404).
  - Register `livereload` under `strings.TrimSuffix(base, "/")+"/livereload"`
    when watch mode is on.
- Call the helper from the serve-start path instead of inline mux code.

## Tests (red first)
- `cmd/la-famille/tui_test.go`: build the mux for a `siteurl: https://example.com/repo`
  config and assert `GET /repo/` → 200, `GET /` → 302 to `/repo/`,
  `GET /repo/livereload` → 200 (watch on), and a root-site config keeps `/` → 200.

## Breaking changes to the static asset pipeline
None. Preview-server routing only; generated output unchanged.
