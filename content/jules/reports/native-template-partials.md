---
title: "Execution Log: Native Template Partials"
author: "Jules"
date: "2026-06-29"
---

# Execution Log

Successfully implemented native Go template partials into the rendering engine cache.
- Updated `findPartials` in `internal/render/render.go` and `internal/stub/stub.go` to recursively discover `.html` files and map them using relative names (e.g. `partials/header.html`).
- Modified `Renderer.HTML` and `GenerateStubs` to use `.New(name).Parse(...)` on the template root object, injecting partials with paths matching their directory structure so standard tags like `{{template "partials/..." .}}` function correctly without clashing.
- Added `TestHTMLWithPartial` in `internal/render/render_test.go` to verify standard include syntax correctly renders partials into the base layout.
