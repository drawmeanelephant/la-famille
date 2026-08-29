# Task Plan: fix-wolf-review-tier2

## Issues addressed (umbrella #500 — WOLF review backlog)

Tier 2 batch off `origin/master` (which includes merged #557):

- **#551** go.mod hygiene + yaml version
- **#550** test coverage for `internal/page/page.go`
- **#552** extract unit tests from `exec.Command` integration tests

## #551 — go.mod / yaml

**Migration decision (important):** a full yaml.v2→v3 port is NOT safe. Probe
results:

- yaml.v3 resolves date-only scalars (`2026-08-29`) into `time.Time` when
  decoding into `map[string]interface{}`; yaml.v2 leaves them as strings.
  Re-marshaling then rewrites `date: 2026-08-29` as `2026-08-29T00:00:00Z`,
  corrupting published dates — so the frontmatter *parse step* must stay on v2.
- `adrg/frontmatter` v0.2.0 (latest; no v3 release) is the only thing pulling in
  yaml.v2; it cannot be upgraded.

Done:
- Migrated all **direct** yaml usage to v3: `internal/content/frontmatter.go`
  (incl. `StringList.UnmarshalYAML` rewritten to the v3 `*yaml.Node` API),
  `internal/config/config.go`, `internal/checker/checker.go`,
  `cmd/la-famille/new.go`.
- `go mod tidy`; yaml.v2 demoted to `// indirect` (transitive via frontmatter),
  yaml.v3 added as a direct dependency.
- Behavioral probes confirmed identical outcomes for every pinned test:
  `tags: yes` → nil tags, float tag `1.5` → normalized `15`, error-line wording,
  and `*bool` allocation on decode failure.
- Go directive: the toolchain rewrites `go 1.25` → `go 1.25.0` (its canonical
  stable form; the issue's "pre-release" claim is a misread — Go 1.25.0 is the
  stable Aug-2025 release).

## #550 — internal/page coverage

New `internal/page/page_test.go` (was `[no test files]`):
- zero-value Page renders every field empty (`TestPageZeroValueRendersEmpty`)
- all 14 render values incl. `Site` config subfields (`TestPageRendersAllFields`)
- Content renders raw HTML; Title/SiteName are escaped (`TestPageEscapesTextButNotContent`)
- `Site` is a value copy, not an alias (`TestPageSiteIsACopy`)

## #552 — exec.Command extraction

The codebase had already decomposed much of this (#540-#542: `loadProjectConfig`,
`resolveProjectPath`, cache logic all have direct unit tests). Remaining wins:

- `sharedGateBinary` (`sync.OnceValue`) builds the test binary **once** per run;
  `buildGateBinary` and the three per-test `go build` sites in `main_test.go`
  now reuse it. `cmd/la-famille` test time ~9.3s → ~4.2s.
- New `TestWriteInitialConfig` unit tests pin the `init` file contract
  (create / refuse+repair-path / `--force` backup / themed layout / unknown
  theme) without a binary. True end-to-end scenarios (exit codes, `serve`,
  config gate) intentionally kept as exec tests per the issue.

## Static asset pipeline impact
None. No `assets/`, `templates/`, or generated-output changes.

## Verification
- `go test ./...` — all pass
- `go vet ./...`, `gofmt -l` — clean
- `go mod tidy -diff` — clean

## Handoff
Uncommitted on `fix/wolf-review-tier2`. Roll into one commit referencing
#550 #551 #552, open a PR, and close the issues (one `Closes #N` per line to
avoid the #557 multi-issue auto-close flake).