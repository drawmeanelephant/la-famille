1. **Refactor `findPartials` to recursively find files and return paths relative to the templates directory:**
    - Replace the `findPartials` function with an implementation that returns a `map[string]string` mapping partial names (relative paths) to absolute paths.
2. **Update parsing logic to inject templates with their relative path names:**
    - Modified `Renderer.HTML` and `GenerateStubs` to use `.New(name).Parse(...)` on the template root object, injecting partials with paths matching their directory structure so standard tags like `{{template "partials/..." .}}` function correctly without clashing.
3. **Verify code edits:**
    - Run `read_file` on `internal/render/render.go` and `internal/stub/stub.go` to confirm the edits were applied successfully.
4. **Add a test for partial inclusion:**
    - Added `TestHTMLWithPartial` in `internal/render/render_test.go` to verify standard include syntax correctly renders partials into the base layout.
5. **Write execution log:**
    - Wrote a report to `content/jules/reports/native-template-partials.md`.
6. **Verify tests:**
    - Run `go test ./...` to ensure all existing and new tests pass.
