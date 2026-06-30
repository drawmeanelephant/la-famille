1. Modify `internal/ragexport/export.go` to add `".github/workflows/*.yml"` to the patterns array in the first `writeBundle` call (System Bundle).
2. Modify `internal/ragexport/export.go` to remove `d.Name() == ".github"` from the `d.IsDir()` guard block in `writeBundle()`.
3. Potential breaking changes to the static asset generation pipeline: None. This only affects the RAG export functionality.
