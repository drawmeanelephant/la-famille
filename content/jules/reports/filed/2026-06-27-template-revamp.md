# Template Loading and Execution Revamp Report

## Issue Addressed
The previous rendering logic in `internal/render/render.go` used a flawed pattern for parsing and executing Go `html/template`s:
1. Templates were parsed repeatedly using `template.ParseFiles` on every page render, leading to unnecessary disk I/O and potential state leaking.
2. The `Execute` method was called on the template set without explicitly naming the root template. Since `ParseFiles` associates the resulting template set with the base filename of the first file parsed, calling `Execute` could render the wrong template depending on the order of files parsed.

## Solution Implemented
The `internal/render` package has been refactored to use a lazy-loading `Renderer` struct containing a template cache.

1. **Lazy Caching:** A `map[string]*template.Template` guarded by a `sync.Mutex` caches parsed templates based on their file paths. `template.ParseFiles` is now only invoked once per unique layout file per `Renderer` lifecycle.
2. **Execution Isolation:** To prevent per-page state bleed or corruption, the cached template is cloned via `template.Must(tmpl.Clone())` for each page execution.
3. **Addressing the Name Trap:** The execution call has been updated from `clonedTmpl.Execute(...)` to `clonedTmpl.ExecuteTemplate(outFile, filepath.Base(templatePath), p)`. By explicitly providing the base filename of the intended root template, we guarantee that the correct layout is rendered regardless of the order in which partials were parsed.
4. **Integration:** `internal/generator/generator.go` was updated to instantiate a new `Renderer` via `render.New()` at the beginning of each `Build` call. This keeps the cache scoped to a single build run, naturally side-stepping invalidation issues during watch mode.

## Testing
The test coverage in `internal/render/render_test.go` has been significantly expanded:
- **`TestHTML`**: The existing regression test was preserved and updated to use the new `Renderer`.
- **`TestHTMLLayoutSelection`**: A new table-driven test was added covering three distinct scenarios:
  - *No Layout Specified*: Verifies that pages fall back to the default `config.yaml` template.
  - *Layout Specified*: Verifies that pages use the explicitly defined layout file (e.g., `custom.html`).
  - *Back-to-back Renders*: Renders two different pages sequentially using the same `Renderer` instance to verify that layouts are rendered correctly without cross-contamination.

All tests, including `go vet`, pass successfully.
