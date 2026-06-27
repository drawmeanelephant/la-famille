# Plan for Audit Template

1. **Extract footer partial:**
   Extract the `<footer>` section from the provided layout to `templates/partials/footer-audit.html`.
2. **Create main layout:**
   Replace the hardcoded footer with `{{template "footer-audit.html" .}}` in the layout, parameterize the content using Go variables, and save to `templates/layout-the-audit.html`.
3. **Update parser logic:**
   Update `internal/render/render.go` and `internal/stub/stub.go` to discover and parse files in `templates/partials/*.html` via `filepath.Glob`.
4. **Test the changes:**
   Run `go test ./...` and `go vet ./...` to verify functionality.

*Note: There are no breaking changes to the static asset generation pipeline.*
