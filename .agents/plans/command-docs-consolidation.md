# Consolidate command architecture documentation

## Scope

Convert the 13 generated command-documentation attempts in `SUPPORT/la-famille-cmd-docs/` into concise, source-grounded documentation that renders as part of the site.

## Work items

- Discard prompt transcripts, branding, placeholder citations, stale snapshot claims, and publish-agent boilerplate.
- Consolidate by workflow rather than documenting every test file as an isolated chapter.
- Add a command-architecture index plus CLI, TUI, contract-test, and release-fixture pages under `content/docs/cmd/`.
- Verify links, Markdown/frontmatter parsing, site build, `go test ./...`, and `go vet ./...`.

## Breaking-change assessment

Documentation-only. No generated site behavior or application code changes are intended.
