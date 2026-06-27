# Task Plan: Existential Goth Minimalist Layout
1. Implement support for Go template partials in the site generator engine (`internal/render/render.go` and `internal/stub/stub.go`). This will involve reading `templates/partials/*.html` using `filepath.Glob` and passing them along with the main layout template to `template.ParseFiles`.
   - Potential breaking changes: Minimal, as `template.ParseFiles` can accept multiple file paths. However, we need to ensure paths to `templates/partials` are resolved correctly even when tests run in subdirectories.
2. Create `templates/partials/footer-void.html` by extracting the bleak footer from the generated Stitch design.
3. Create `templates/layout-the-void.html` by extracting the main HTML structure from the generated Stitch design, incorporating Go template variables for content (`{{.Title}}`, `{{.Content}}`, `{{.Date}}`, `{{.Author}}`, etc.), adding DaisyUI theme attributes, Tailwind typography `prose` classes, and accessibility requirements.
4. Verify changes pass tests (`go test ./...` and `go vet ./...`).
