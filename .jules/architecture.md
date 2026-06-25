# Architecture Notes

## Refactoring Seams
* 2026-06-20: Extracted `GatherMetadata` (which walks directories and parses markdown frontmatter) out of `cmd/la-famille/main.go` into a new package `internal/content`. This improves the modularity of the codebase by moving file-system reading and parsing logic out of the CLI's main entry point, preparing it for potentially being used by other parts of the system (like the taxonomy or search features) independently of the main site generation loop.
* 2026-06-25: Extracted HTML rendering logic out of `cmd/la-famille/main.go` and `internal/generator/generator.go` into a new package `internal/render`. This isolates layout template parsing and HTML execution from the primary site generation loop, preparing the generation step for easier refactoring and sharing template responsibilities.
