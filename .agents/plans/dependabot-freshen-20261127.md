# Dependabot resolution and repository freshen-up

## Task ID
`dependabot-freshen-20261127`

## Objective
Bring the repository back onto current `master`, resolve and merge eligible
Dependabot pull requests, perform a lightweight dependency and hygiene pass,
and identify the next useful work items.

## Scope
- Inspect open Dependabot PRs and merge those that are mergeable with passing
  required checks.
- Fast-forward the local `master` checkout after remote merges.
- Review dependency/configuration drift, especially Go modules, GitHub
  Actions, and Dependabot configuration.
- Run the required local validation: `go test ./...` and `go vet ./...`.
- Run the repository formatting/quality hook or equivalent checks when useful.
- Preserve the embedded `t3-b09a428c` worktree because it contains uncommitted
  changes; do not delete it without explicit operator approval.

## Potential static asset generation impact
No intentional static-output or asset-generation behavior changes are planned.
This pass should be limited to dependency/configuration hygiene unless validation
exposes a defect that must be fixed. Any dependency-induced output difference
will be treated as a regression and investigated before merging further work.

## Validation
- `go test ./...`
- `go vet ./...`
- `./format_check.sh` if the local toolchain supports it
- `actionlint .github/workflows/*.yml`
- Real content build plus `publish-check` (non-strict; `--strict` intentionally
  fails because `content/docs/index.md` links the missing-page demo)

## Work completed in this pass
- Merged Dependabot PR #684 (`golang.org/x/net` v0.62.0 -> v0.63.0); no open
  pull requests remain.
- Fast-forwarded local `master` to the post-merge remote head.
- Confirmed direct Go module dependencies are already current; no module file
  changes were needed.
- Aligned `jules-ci.yml` with the main lint workflow (`golangci-lint-action@v8`,
  `golangci-lint` v2.x) because the old v1 linter cannot read the repository's
  v2 lint configuration.
- Added a workflow contract test that compares the Jules and main lint
  toolchains instead of pinning duplicate literals.
- Made watcher and pack tests compatible with current errcheck releases by
  explicitly asserting watcher cancellation and by using a struct channel for
  the FIFO timing signal.

## Follow-up assessment
Inspect the roadmap, open issues, stale local branches/worktrees, and repository
maintenance configuration. Report a prioritized next-step list rather than
implementing unrelated feature work in this maintenance pass.

Findings:
- The only open issue is #665 (Raoul TUI state machine).
- The embedded `t3-b09a428c` worktree contains an uncommitted implementation of
  #665. Its patch applies cleanly to current `master`; in a disposable
  worktree, `go test ./cmd/la-famille -run Raoul -count=1` and
  `go test ./cmd/la-famille -count=1` passed.
- The embedded worktree should be preserved or deliberately finished/moved. It
  should not be deleted as cleanup because it contains uncommitted work.
- `content/meta/roadmap.md` is stale: several “current” release-readiness and
  publishing items shipped in v0.2.0.
- Many showcase pages still reference retired/non-bundled layouts and fall back
  to the default template; decide whether to restore those templates, mark the
  pages historical, or remove the stale demos.
