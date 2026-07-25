# Codex PR Litterbox Hardening Plan

## Objective

Convert `la-famille pr sync` (“Clear the Litterbox”) from implicit write automation into an explicit, safe-by-default policy engine: dry-run by default, label-gated merges, no automatic conflict closure, no silent zero-check success, no local publishing unless authorized, and a single merge authority (nightly litterbox only).

## Current State (inspected)

| Area | Behavior today |
|------|----------------|
| CLI | Only `--base` (default `master`); always mutates |
| Sync | Closes conflicts automatically; merges when checks “pass” |
| Checks | First page only; **zero checks count as passing** |
| Merge | No SHA, no squash body, no `merged` verification |
| Local changes | Always publishes branch/PR when dirty working tree |
| Jules CI | Independent `gh pr merge --auto --squash` |
| Docs | Contradictory default (`main` vs `master`) |

## Architecture

Keep package boundaries; add a thin decision seam:

```
cmd/la-famille/pr.go          CLI flags + stdout summary
internal/github/policy.go     pure EvaluatePR (no HTTP)
internal/github/sync.go       orchestration + SyncResult
internal/github/github.go     hardened REST client
internal/git/git.go           CurrentBranch/Checkout + existing ops
```

```
Fetch state  →  EvaluatePR(policy)  →  Apply mutations (if --apply)
     │                  │                        │
  Client API      unit-testable            Merge/Close/CreatePR
```

## Implementation Steps

### 1. GitHub client (`internal/github/github.go`)

- Headers: `Accept: application/vnd.github+json`, `X-GitHub-Api-Version`, `User-Agent: la-famille`
- Keep 10s timeout; add optional `BaseURL` for httptest
- Bounded error body reads; contextual errors (method/path/status); never log token
- Extend `PullRequest`: Draft, Labels, Base, Head SHA/ref
- `ListOpenPRs(authors, base)`: `per_page=100`, multi-page, max-page guard, base query, case-insensitive author filter, sort by number
- `GetCheckSummary(ref)` → `CheckState` (`none|pending|failed|passing`) with full pagination + truncation error
- `MergePR(number, sha)`: squash + expected SHA; decode `{merged,message,sha}`; error if `merged:false`
- `ClosePR`, `CreatePR` preserved
- `GetDefaultBranch()` from repo metadata
- `APIClient` interface for orchestration fakes

### 2. Policy layer (`internal/github/policy.go`)

Pure `EvaluatePR(PRState, PolicyConfig) PRDecision`:

**Merge gates (all required):** bot author, base match, not draft, required label (case-insensitive), head-prefix if configured, mergeable true (not null), checks passing (or none only with `AllowNoChecks`), `Apply` for `merge` vs `would_merge`.

**Conflict:** null mergeable → skip (computing); conflict without `--close-conflicts` → skip; with flag → `would_close`/`close` after identity gates (checks not required).

Policy skips ≠ operational errors.

### 3. Orchestration (`internal/github/sync.go`)

- `SyncConfig` with all flags + injectable `Git`, `Now`, `Sleep`, `Client`/`Owner`/`Repo`
- `RunSync` → `(SyncResult, error)`
- Resolve base from API when empty; failure is operational (no mutations)
- Evaluate PRs in ascending number order; continue on per-PR API errors; `errors.Join` at end
- Local publish quarantined behind `--publish-local-changes` + `--apply`; dry-run reports `WOULD_CREATE_PR` without git/GitHub mutation
- Restore original branch after local publish when possible
- Deterministic branch names from injected clock
- Do not auto-apply `litterbox-approved` to locally created PRs

### 4. Git package

- Add `CurrentBranch()`, `Checkout(existing)` for restore
- Package functions remain; sync injects a thin `GitRunner` wrapper

### 5. CLI (`cmd/la-famille/pr.go`)

Flags:

| Flag | Default |
|------|---------|
| `--base` | `""` (resolve default_branch) |
| `--apply` | false |
| `--required-label` | `litterbox-approved` |
| `--close-conflicts` | false |
| `--allow-no-checks` | false |
| `--publish-local-changes` | false |
| `--bot-author` | `google-labs-jules`, `google-labs-code` |
| `--head-prefix` | none |

Reject empty `--required-label`. Print stable summary to `cmd.OutOrStdout()`.

### 6. Workflows

**cron-sync.yml**
- Concurrency group `pr-litterbox`, `cancel-in-progress: false`
- Permissions: contents/write, pull-requests/write, checks/read
- `pr sync --base master --required-label litterbox-approved --apply`
- Explicit comments: no close-conflicts, no publish-local-changes, no allow-no-checks

**jules-ci.yml**
- Rename job to `verify-jules-pr`
- Remove `gh pr merge` step
- No PR write permissions; keep verify-only

### 7. Documentation

Update `content/docs/pr.md` and PR section of `content/docs/cli.md` with full policy contract and litterbox personality.

### 8. Tests (no live network/git)

- Pure policy: 16 cases (eligible dry/apply, author, base, draft, label, label case, prefix match/miss, mergeable null, conflict variants, checks pending/failed/none/allow-none)
- Client: pagination, query params, sorting, checks, merge body/response, close, default branch, error sanitization
- Orchestration: dry-run zero mutations, apply merge one PR, no close without flag, aggregate errors, local publish gates, sleeper injection
- CLI: flag defaults, token, empty label, dry-run default, stdout summary, non-zero on ops error
- Workflow static: cron flags, jules no merge

### 9. Verification

```bash
gofmt -w cmd/la-famille internal/github internal/git
go test -count=1 ./internal/github ./internal/git ./cmd/la-famille
go test -count=1 ./...
go test -race ./...
go vet ./...
golangci-lint run
```

## Breaking / intentional behavioral changes

- Dry-run default (nightly must pass `--apply`)
- Required label mandatory
- Zero checks no longer pass by default
- Conflicts not closed by default
- Local changes not published by default
- Jules CI no longer merges

## Non-goals

No GitHub SDK, no config.yaml for policy, no site-gen/RAG/TUI changes, no auto-label of local PRs, no approval-review semantics.

## Risk notes

- Existing automation that relied on unlabeled auto-merge must start applying `litterbox-approved`
- Nightly workflow must keep `--apply` or it becomes a no-op reporter
- Branch protection still ultimate authority; litterbox only requests merges
