---
date: "2026-07-09"
title: "Pull Request Management"
author: "Jules"
---

# Pull Request Management

La Famille includes a built-in command to help manage automated pull requests. This is Raoul(s) clearing the litterbox: he still scoops, but now he checks the tag on the bag, verifies the correct tray, confirms the diamonds are accounted for, and asks for explicit permission before operating the industrial scoop.

## Automated PR Sync (`pr sync`)

```bash
go run ./cmd/la-famille pr sync
```

### Safe-by-default contract

| Behavior | Default |
|----------|---------|
| Dry-run | **Yes** — inspect and decide only |
| Mutations | Only with `--apply` |
| Required label | `litterbox-approved` (mandatory; no empty bypass) |
| Zero check runs | **Not** accepted (use `--allow-no-checks` to override) |
| Conflicts | Reported and skipped (use `--close-conflicts` to authorize closure) |
| Local working-tree publish | Disabled (use `--publish-local-changes`) |

Without `--apply`, the command prints a deterministic decision for every relevant open PR (`SKIP`, `WOULD_MERGE`, `WOULD_CLOSE`, …) and exits without calling merge, close, push, or create-PR endpoints.

### Requirements

```bash
export GITHUB_TOKEN="your_personal_access_token"
```

The token needs permission to read pull requests and checks. For `--apply` merges/closes, it also needs write access to pull requests (and contents if local publishing is enabled).

Owner and repository are inferred from the `origin` remote.

### Policy gates (merge)

A PR may be merged only when **all** of the following are true:

1. Author is on the bot allowlist (default: `google-labs-jules`, `google-labs-code`; case-insensitive).
2. Base branch exactly matches the resolved/configured target.
3. It is not a draft.
4. It has the required label (default `litterbox-approved`; case-insensitive).
5. Head ref matches a configured `--head-prefix` when any prefixes are set.
6. GitHub reports `mergeable: true` (null / still computing is skipped; never treated as conflict).
7. At least one check run exists unless `--allow-no-checks`.
8. Every check run is complete with conclusion `success`, `skipped`, or `neutral`.
9. The command was invoked with `--apply`.

Merges use **squash** and send the evaluated head SHA so the PR cannot be merged after its head moves. The merge response is checked for `merged: true`.

Policy skips are normal and do not fail the command. Operational failures (API errors, truncated pagination, merge/close failures) produce a non-zero exit after other PRs are still evaluated where safe.

### Conflict policy

- Default: `SKIP conflict detected; use --close-conflicts to authorize closure`
- `--close-conflicts` without `--apply`: `WOULD_CLOSE`
- Both flags, identity/trust gates satisfied: `CLOSED`

Closing a conflict still requires allowlisted author, matching base, non-draft, required label, and head-prefix rules. Check-run success is **not** required to close a genuine conflict.

### Local working-tree publishing

With `--publish-local-changes` (and `--apply` for real mutations), dirty working trees may be staged **in full**, committed, pushed to a deterministic `jules-auto-YYYYMMDDHHMMSS` branch, and opened as a PR.

- Dry-run reports `WOULD_CREATE_PR` without git or GitHub mutation.
- Locally created PRs do **not** receive `litterbox-approved` automatically. A human or separate trusted automation must apply the label before the litterbox will merge them.
- Enabling this stages **all** current working-tree changes. Use with care.

### Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--base` | empty | Target base branch. Empty → resolve repository `default_branch` via GitHub API |
| `--apply` | false | Perform mutations |
| `--required-label` | `litterbox-approved` | Label required to merge or close |
| `--close-conflicts` | false | Authorize closing conflicting PRs that pass identity gates |
| `--allow-no-checks` | false | Allow merge when zero check runs are reported |
| `--publish-local-changes` | false | Authorize local branch/commit/push/PR creation |
| `--bot-author` | Jules bot logins | Repeatable allowlist |
| `--head-prefix` | none | Optional repeatable head ref prefixes |

There is no separate `--dry-run` flag; omit `--apply` for dry-run.

### Nightly workflow (this repository)

`.github/workflows/cron-sync.yml` (“Clear the Litterbox”) is the **single merge authority** for automation PRs. It runs:

```bash
go run ./cmd/la-famille pr sync \
  --base master \
  --required-label litterbox-approved \
  --apply
```

It deliberately does **not** pass `--close-conflicts`, `--allow-no-checks`, or `--publish-local-changes`. This repository’s scheduled workflow explicitly targets `master`.

### Jules CI

`.github/workflows/jules-ci.yml` **verifies** Jules-related PRs (tests, race, lint). It does **not** merge. Verification may cover more PRs than the litterbox will merge; verification is not mutation.

### Example dry-run output

```text
🐙 Clear the Litterbox
Repository: drawmeanelephant/la-famille
Base: master
Mode: dry-run

PR #412 SKIP         missing required label "litterbox-approved"
PR #413 SKIP         checks pending
PR #414 WOULD_MERGE  all policy gates passed
PR #415 SKIP         conflict detected; use --close-conflicts to authorize closure

Summary: inspected=4 skipped=3 would_merge=1 merged=0 would_close=0 closed=0
```
