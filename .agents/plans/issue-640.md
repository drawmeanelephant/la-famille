# Issue #640: pr sync must honor --project-root

## Scope and ownership

- Owner: Devin on `t3/fix-config-cli`. HIGH RISK: `--project-root B` run from
  repo A currently reports and mutates repo A (merge/close PRs, commit+push).
- Root causes: `cmd/la-famille/pr.go` never threads the bootstrapped project
  root into `github.SyncConfig`; `internal/git/git.go` runs every git command
  without `cmd.Dir`; `internal/github/sync.go` `resolveClient` reads the CWD
  repo's origin.

## Proposed change

- `internal/git/git.go`: introduce `type Repo struct{ Dir string }` whose
  methods run each git command with `cmd.Dir = Dir` (empty = CWD, unchanged
  behavior). Convert the package-level helpers into `Repo` methods; keep
  `ParseOwnerRepo` a pure function.
- `internal/github/sync.go`: add `SyncConfig.WorkDir`; when `cfg.Git` is nil,
  default it to a dir-bound `realGit` (`git.Repo{Dir: cfg.WorkDir}`). All
  local-repo operations — remote resolution and publish-local-changes — then
  run under `--project-root`.
- `cmd/la-famille/pr.go`: construct the `pr` command tree from the
  bootstrapped `cfg` (same pattern as `setupNewCmd`) and pass
  `WorkDir: cfg.ProjectRoot` into `SyncConfig`. The bootstrapper already
  resolves the flag/config value to an absolute path.
- Test seam: `var runPRSync = github.RunSync` so a cmd-level test can capture
  the `SyncConfig` without a GitHub token or network.

## Tests

- `internal/git/git_test.go`: two fixture repos with different origins; assert
  `Repo{Dir: B}` commands report B's remote/branch while run from A.
- `internal/github/sync_test.go`: `resolveClient` via a `WorkDir`-bound
  runner returns repo B's owner/repo while the process CWD is repo A.
- `cmd/la-famille/pr_test.go`: executing `pr sync` with `cfg.ProjectRoot = B`
  from inside repo A produces `SyncConfig.WorkDir == B`.
- Fixture repos only — no real `pr sync` against the live repo.

## Breaking changes / pipeline impact

- `internal/git` package-level command helpers become `Repo` methods (sole
  caller is `internal/github`). `pr sync` in a repo invoked without
  `--project-root` is unchanged (`WorkDir` empty → CWD).
- No static-asset pipeline impact.

## Verification

- `go test ./internal/git/ ./internal/github/ ./cmd/`, `go test ./...`,
  `go vet ./...`.
