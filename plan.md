## Task: Fix RAG export bloating and automate via GitHub Actions

1. Fixed RAG export directory matching logic in `internal/ragexport/export.go`. Root-level patterns like `*.go` and `README.md` now only match files in the root directory, rather than in all subdirectories, and `vendor/` and `node_modules/` are explicitly excluded from the directory walk.
2. Updated `.github/workflows/deploy.yml` to automatically run `go run ./cmd/la-famille rag` and copy the output `rag-archive/` directory into `public/rag-archive/` so it is hosted alongside the static site.
3. Tests and code compilation completed successfully.

### Potential Breaking Changes:
- RAG export archives generated moving forward will no longer erroneously include deep nested files that happen to match root-level patterns, which is the intended behavior.
- The `rag-archive/` directory will now be hosted as part of the public site, which should not affect standard page generation.
