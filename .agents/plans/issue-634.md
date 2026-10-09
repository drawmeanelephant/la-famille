# Plan: #634 — WatchMode excluded from fingerprint but changes rendered output

## Bug
`cacheFingerprint` hashes `json.Marshal(cfg)`, and `WatchMode` is tagged
`json:"-"`, so watch-mode and production builds share a fingerprint. But
`render.Renderer.HTML` injects the livereload `<script>` only when
`cfg.WatchMode` is set — output bytes differ while the fingerprint does not:

- `build` then `serve --watch` → cache hit → served pages have no livereload.
- `serve --watch` then `build` → cache hit → `new EventSource(...)` ships in
  the production artifact.

## Fix
`internal/generator/cache.go`: write the WatchMode bit into the fingerprint
explicitly (the field stays `yaml:"-"`/`json:"-"` in Config; only the hash
input gains it). Update the stale comment that claimed the exclusion was safe.

Alternative considered: injecting livereload at serve time instead of into
cached pages. Rejected — the serve file server is a plain `http.FileServer`
and rewriting responses there is a larger change; fingerprinting the mode is
the smaller correct fix the issue's acceptance criteria allow.

## Tests
- `internal/generator`: build with WatchMode=false → flip to true → cache
  miss + livereload present; flip back → cache miss + no `EventSource` in
  `public/`; same-mode rebuild stays a hit.

## Breaking changes
- First build after upgrade misses once (new fingerprint input). Self-healing.
- `serve --watch` after a production `build` now pays one rebuild, which is
  exactly the corrected behavior.
