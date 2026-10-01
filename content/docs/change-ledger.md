---
title: "Change Ledger"
description: "Trustworthy build diffs, regression-only gates, and the TUI Changes pane."
date: "2026-09-30"
author: "La Famille"
---

# What did this build change?

Every successful build publishes three deterministic artifacts:

- `site-manifest.json`: the canonical source/graph projection, effective
  supported frontmatter, Markdown body hashes, published page/file hashes,
  and the actual sitemap entries.
- `diff.json`: a versioned ledger (`version`, `baseline`, `changes`).
- `diff.txt`: the same changes in a readable report.

The first build establishes a baseline, without treating existing broken
links or orphans as new regressions. Later builds compare against the last
successfully published manifest. Failed builds leave the published artifacts
untouched. Cache hits preserve the last ledger rather than erasing it.
Deleting the generated output establishes a new baseline; preserve a manifest
elsewhere if you need comparisons across clean builds or CI runs.

Manifests contain no build timestamps or machine-specific paths. A no-op
build produces byte-identical manifests. File hashes exclude the manifest and
ledger themselves, avoiding recursive changes. The private build cache keeps
the snapshot and ledger, but must never be published.

## Compare snapshots

```bash
la-famille diff public-before public
la-famille diff before.json after.json --json
la-famille diff ref:master ref:HEAD --gate --report-dir reports
```

Filesystem paths resolve from `--project-root`. Git inputs use a committed
manifest when available; otherwise the current generator builds each revision
in a disposable archive using that revision's configuration. It never checks
out files into your working tree, runs repository scripts, or contacts the
network. Source archives with symlinks or external input/configuration paths
are rejected; use built manifests for those sites.

The report separates **source changes** (`changed_pages`) from **rendered
dependency changes** (`rendered_pages`). For example, editing a title changes
that page's metadata and may also change its neighbors' backlink panels.
Template-only changes are visible as rendered changes. `file_changes` records
asset and derived-metadata drift, including files without Markdown references.

Renames match a unique unchanged title among unmatched pages. Ambiguous titles
or simultaneous identity/title changes are reported as removal/addition,
not guessed. Losing a previously linked public URL remains a regression even
when a rename is recognized.

Legacy v1 manifests remain readable, but cannot detect prose, extra
frontmatter, output bytes, or sitemap changes. Reports explicitly warn about
incomplete coverage, and strict gating requires complete v2 snapshots. Rebuild
both sides with the same generator to avoid comparison gaps.

## Regression-only gate

`--gate` saves/prints its report before exiting nonzero for:

| Machine kind | Regression |
| --- | --- |
| `broken_link` | An internal link becomes unresolved or a new unresolved link appears |
| `orphan` | A rendered page other than the root index newly has no inbound links |
| `removed_linked_page` | A public page that previously had inbound links disappears |
| `taxonomy` | A tag/category disappears from the entire site's taxonomy |
| `sitemap` | An actual sitemap entry is lost |

Prose, asset changes, added sitemap entries, and changes to a taxonomy term's
membership that do not erase the term are not failures. Pre-existing orphans
and broken links are unchanged debt, not new regressions. Moving an existing
broken link to another source line does not create a new regression. Broken
link detection uses the existing checker's rules; it does not add fragment
validation or external link checks.

Intentional removals still need human review. Compare against an approved new
baseline after reviewing the report, rather than silently weakening the gate.

## TUI

Choose **Changes** or press `l`. `j`/`k` select rows, `r` filters to regressions,
`d` toggles Diagnostics, and `?` toggles help. A prose-only three-page edit
shows green `no regressions, 3 pages changed`; one new broken link shows red
`1 newly-broken link`. The pane updates after manual and watcher builds and
loads the saved ledger when the TUI starts.

## Repository CI rollout and closure evidence

`.github/workflows/change-ledger.yml` runs strict synthetic controls on the
`anchor-links` and `artisanal-ceramics` fixtures, then compares the repository
site at the PR base/head. It uploads both reports and records the comparison
outcome in the job summary. The repository comparison starts **advisory**:
regressions are visible without blocking unrelated work. Synthetic controls
always fail the job if prose is rejected or a new broken link is accepted.

To reproduce the strict controls locally:

```bash
go test ./cmd/la-famille -run '^TestChangeLedgerGateFixtures$' -count=1 -v
```

Before closing #580, capture actual PR evidence, not just local assertions:

1. Open a prose-only test PR. Verify the comparison step passes.
2. Open a test PR adding `[Synthetic regression](ledger-does-not-exist.md)`
   to a linked page. Verify the comparison step fails and its artifact
   identifies exactly that new broken link. Advisory mode keeps the overall
   job non-blocking; record the step's failure.
3. Observe a full week of quiet advisory comparisons. Set the repository
   Actions variable `CHANGE_LEDGER_ENFORCE=true`, then rerun the broken-link
   PR and verify the **job** fails. Make the gate a required check if desired.
4. Record the run links and the advisory observation window in the PR
   description. Close the synthetic PRs without merging their test edits.

### Recorded real-PR controls

The implementation is [PR 599](https://github.com/drawmeanelephant/la-famille/pull/599).
On October 1, 2026 (UTC), strict workflow copies on draft synthetic PRs proved
the actual job behavior without changing repository-wide enforcement:

- [Prose-only PR 600](https://github.com/drawmeanelephant/la-famille/pull/600):
  [strict job passed](https://github.com/drawmeanelephant/la-famille/actions/runs/36800978395/job/110175016854).
  Its saved report contains zero regressions and no coverage warnings.
- [Broken-link PR 601](https://github.com/drawmeanelephant/la-famille/pull/601):
  [strict job failed](https://github.com/drawmeanelephant/la-famille/actions/runs/36800983594/job/110175035316)
  at the comparison step, not setup/build. Its saved report contains exactly
  one regression: `index`, `ledger-does-not-exist.md`, source line 36.

These are do-not-merge controls. They do **not** establish a week of quiet
advisory operation after rollout. That observation, enforcement decision, and
final issue closure remain separate steps.
