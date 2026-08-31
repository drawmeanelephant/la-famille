# Octoburger: the global default flagship theme

Task ID: `octoburger-pages-default`
Branch: master

## Scope

Octoburger is La Famille's flagship soul theme (already reflected in
`internal/runtimeassets/assets.go`, `.agents/plans/497-octoburger-core-theme.md`,
`content/showcase/layout-octoburger.md`). This task promotes it from a
selectable flagship to the **global default**: every build of this project
(local and GitHub Pages) and every freshly `init`ed site renders octoburger,
and the theme is documented as an integral part of the project.

## Changes

- `config.yaml` + regenerated `internal/config/default_config_gen.go`: the
  canonical `template` is now `templates/layout-octoburger.html`, so every
  build (no flag needed) renders octoburger. The comment notes the value must
  stay in sync with `internal/config.DefaultLayoutPath`.
- `internal/config/config.go`:
  - New `DefaultLayoutPath = "templates/layout-octoburger.html"` const, used
    as the `DefaultConfig().Template` zero-default.
  - `WriteDefault` now writes with `DefaultLayoutPath` (fresh plain `init`
    sites default to octoburger).
  - `WriteDefaultWithLayout` replaces the `defaultLayoutPath` token
    (previously hardcoded `templates/layout.html`) so `init --theme <name>`
    still swaps layouts correctly now that the canonical default differs.
- `.github/workflows/deploy.yml`: `Build Static Site` passes
  `-t templates/layout-octoburger.html`, keeping Pages explicitly octoburger
  even if `config.yaml` is later edited.
- `action.yml`: the `template` input default is now octoburger, so the GitHub
  Action no longer overrides a site's octoburger default back to
  `layout.html`.
- Tests: `cmd/la-famille/main_test.go` and `config_gate_test.go` updated where
  they hardcoded the old `layout.html` default (plain-init expectation, and the
  gate/`serve` setups that scaffold the default template); they now point at
  `config.DefaultLayoutPath`.
- Docs: `README.md` (octoburger section now states it is the global default;
  Action example), `content/docs/config.md`, `content/docs/cli.md`,
  `content/docs/templates.md`, `content/docs/rag.md`, `RELEASE-QUICKSTART.md`
  all describe octoburger as the default template.
- Plan recorded in `.agents/plans/octoburger-pages-default.md` (this file).

## Breaking changes to the asset pipeline

None. Template selection is file-based; octoburger is already embedded in
`templates/embed.go` and installed by `init`. One intentional behaviour
change: fresh `init` (without `--theme`) now writes an octoburger default,
where it previously wrote `layout.html`.

## Verification

- `go test ./...` — 34 packages, 0 failures.
- `go vet ./...` — clean.
- `go generate ./internal/config` — regenerated `default_config_gen.go` in
  sync with `config.yaml` (no drift test failure).