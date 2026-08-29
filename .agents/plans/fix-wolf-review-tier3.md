# Task Plan: fix-wolf-review-tier3

## Issues addressed (umbrella #500 — WOLF review backlog)

Tier 3 batch off `origin/master` (includes merged #557/#558):

- **#547** reduce global mutable state in `cmd/la-famille/main.go`
- **#548** eliminate `config.yaml` / `defaultConfigYaml` duplication

## #547 — main.go global state

The ten flag globals (`globalLogFile`, `contentDir`, `outputDir`, `assetDir`,
`templateFile`, `siteURL`, `projectRoot`, `configPath`, `showVersion`,
`versionJSON`) plus `logFile` are now fields of a per-tree `cliState` struct
bound by cobra `StringVar`/`BoolVar` to `&st.field`, captured by the command
closures. `setupRootCmd(cfg)` is kept as a thin wrapper for the ~45 test call
sites; `setupRootCmdState(cfg)` returns `(*cobra.Command, *cliState)` so main
can `defer st.closeLogFile()`.

The TUI handoff globals (`tuiRuntimeConfig`, `tuiRuntimeConfigSet`) and the
package-level `tuiCmd` var are gone: `setupTUICmd(st, runtimeCfg, runtimeCfgSet)`
builds the command with the bootstrapped config captured in parameters.
`guardUnusableConfig` now takes the state (`st.showVersion`).

Left as package-level on purpose: `p *tea.Program` (cross-goroutine handle used
by watcher callbacks) and the immutable lipgloss style vars. `askFlagBundle` in
ask.go was not in the issue's enumerated list and is out of scope.

## #548 — config.yaml / defaultConfigYaml dedup

They had already drifted (siteurl comment wording; the `graph_explorer` docs
block existed only in the const). Chosen approach (issue's first suggestion):

- `config.yaml` at repo root is the single canonical default; added the
  `graph_explorer` comment block so init writes the same docs the repo ships.
- New generator `internal/config/gendefault` (wired via `//go:generate go run
  ./gendefault` in config.go) emits `default_config_gen.go` with the const,
  byte-identical to config.yaml.
- The hand-written const in config.go was deleted; `WriteDefaultWithLayout`'
  theme `strings.Replace` still matches (verified by existing themed-init tests).
- New `TestDefaultConfigYamlMatchesCanonical` drift test: fails if the const
  and config.yaml diverge, telling the operator to re-run `go generate`.

## Tests
- `internal/config/default_config_test.go` — drift guard (#548).
- Existing gate tests updated for the new signatures (`guardUnusableConfig`,
  `setupTUICmd`); no behavior change.

## Static asset pipeline impact
None.

## Verification
- `go test ./...`, `go vet ./...`, `gofmt -l`, `go mod tidy -diff` — clean
- `go generate ./internal/config` idempotent (regeneration produces no diff)

## Handoff
Uncommitted on `fix/wolf-review-tier3`. Commit referencing #547 #548 (one
`Closes` line per issue in the PR body), then confirm the four open #500
findings are the only review backlog left (#549 was folded into #557's scope;
check the checklist for any stragglers).