# Roadmap Implementation Plan

1. Created `todos/ROADMAP.md` tracking active roadmap tasks.
2. Extracted the build pipeline from `cmd/la-famille/main.go` into `internal/generator/generator.go` providing a `Build(cfg config.Config) error` method.
3. Updated `internal/content/metadata.go` to unmarshal into a generic map, normalizing frontmatter keys to lowercase prior to strictly typed parsing.
4. Added `AssetDir` (default: `"assets"`) and `RagDir` (default: `"rag-archive"`) to configuration.
5. Added a `Validate()` method to the configuration checking path boundaries and required fields.
6. Implemented a verbatim asset copy step in the generator that copies `cfg.AssetDir` content (ignoring `testdata`) into `public/assets`.
7. Formalized the RAG export path by migrating output from `internal/rag-archive` to `cfg.RagDir`, accepting `cfg` via `ragexport.RunExport(cfg)`, and updating `.gitignore`.
8. Created `internal/watcher` to monitor `content/` and `templates/` changes via `fsnotify`. Added `--watch` to the CLI and a `Serve Site with Watch` mode to the TUI.
9. Passed test suite (`go test ./...`) and vetted code (`go vet ./...`).
