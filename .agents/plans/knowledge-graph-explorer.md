# Knowledge Graph Explorer (moonshot)

## Goal

Ship the static Knowledge Graph Explorer for La Famille as a first-class
feature. After `la-famille build`, users open `public/graph/index.html` and
see their site rendered as an interactive directed graph, sourced from the
existing graph artifacts with no runtime backend.

## Branch

`freebuff/new-thread-thmrz8k5193m70`

## Scope

- New internal package `internal/graphexplorer/` that emits
  `<output>/graph/index.html` from a `go:embed`'d HTML template.
- New self-contained client bundle `assets/graph/explorer.{js,css}`,
  copied by the existing asset pipeline.
- `internal/generator/generator.go` wired to invoke `graphexplorer.Write`
  after `graph.WriteGraphFiles` + `sitedata.Write`.
- `internal/config/config.go` extends the Config struct with
  `GraphExplorer bool` (default `true`, snake_case yaml key
  `graph_explorer` — matches neighboring `check_asset_health`).
- Documentation updates: README, content/docs/{cli,config,publishing,
  generator}.md.
- Unit and integration tests:
  - internal/graphexplorer/graphexplorer_test.go
  - internal/generator/generator_test.go (3 new tests appended).

## Decisions

- **No meta.json contract change.** Public URLs are derived client-side
  from the page id (`/<id>/`). Sites that use front-matter `slug:` will
  show unslugged links — documented limitation in publishing.md rather
  than an additive JSON field that would change the existing fixture
  expectations in cmd/la-famille.
- **Skipped go:embed for JS/CSS.** Only the HTML template is embedded.
  The client bundle ships as `assets/graph/explorer.{js,css}` and is
  copied by the established asset pipeline. This matches the original
  architecture step 1 ("asset bundle under assets/, copied by the
  established asset pipeline") instead of polluting the binary.
- **Config key: snake_case.** `graph_explorer` rather than the spec's
  camelCase `graphExplorer`. Documented as a deliberate deviation in
  content/docs/config.md and .agents/plans/.
- **Focus mode = hide, not dim.** Hides nodes that aren't in
  `selected ∪ {adj[selected]}`. Easier to read on small-to-medium graphs.
- **Threshold for "large site" mode = 500 nodes.** Surfaced in the page
  footer so future tuning does not require code changes.

## Verification

- `gofmt -w .` → exit 0
- `go vet ./...` → exit 0
- `go test -count=1 ./...` → exit 0
- `go test -race ./...` → exit 0
- Live browser check of the built `/graph/` page:
  - `loaded=true`, `rendered=true`, `title="Knowledge Graph — La Famille"`,
    no console errors, no page errors.

## Known limitations (documented)

1. Custom front-matter `slug:` overrides produce unslugged links in the
   detail panel.
2. Auto-navigation injection is intentionally omitted (would break
   custom-template compatibility). A stable manual nav snippet is
   documented in publishing.md and README.
3. Config key is snake_case `graph_explorer`. The spec's camelCase form
   is documented but not accepted.

## Risks / future work

- Tags/categories toggle is implemented only as a search-prefix form
  (`tag:foo`, `category:bar`). A dedicated "With taxonomy" filter pill is
  plausible future work.
- Asset-dir content hashes are not part of the build-cache fingerprint
  beyond mtime+size; if mtime-equivalent content changes are needed, a
  hash-based fingerprint would be a follow-up.
