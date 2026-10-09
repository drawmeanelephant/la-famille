# Plan: #636 — stub pages bypass applyBasePath under subpath siteurl

## Bug
`stub.generateSingleStub` renders via a hand-rolled
`template.New(...).Parse(...)` + `ExecuteTemplate` instead of
`render.Renderer.HTML`, so stub pages never get `applyBasePath`, the
`la-famille-base-path` meta tag, or watch-mode livereload injection. Under
`siteurl: https://example.com/blog` stubs emit `href="/assets/..."` that 404
on the real deploy.

## Fix
`internal/stub/stub.go`: take the shared `*render.Renderer` and render through
`Renderer.HTML`, which performs template+partials parsing, base-path rebasing,
meta injection, and livereload — identical to normal pages. Remove the local
os.Create/template plumbing; the renderer creates the file. The ownership
claim still gates the write before rendering happens.

`internal/generator/generator.go`: pass `bc.renderer` into
`stub.GenerateStubs`.

## Tests
- `internal/stub`: subpath siteurl produces stub HTML with rebased
  `href="/blog/assets/..."` and the base-path meta; WatchMode adds the
  livereload script.

## Breaking changes
- `stub.GenerateStubs` signature gains the renderer (internal package;
  callers and tests updated).
