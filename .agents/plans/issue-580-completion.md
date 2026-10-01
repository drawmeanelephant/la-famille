# Issue #580: complete the Change Ledger

## Findings and scope

Phase 1 and a read-only Phase 2 command are already present. Remaining gaps:
prose and additional frontmatter edits are invisible; asset bytes and sitemap
entries are not captured; builds do not publish a ledger; no regression gate,
Changes pane, or repository CI gate exists.

1. Extend the deterministic manifest with body/frontmatter fingerprints,
   published page and asset hashes, and actual sitemap entries. Read legacy
   manifests explicitly, reporting unavailable coverage rather than silently
   claiming a complete diff.
2. Complete semantic comparison and deterministic `diff.json`/text artifacts,
   preserving the previous successful build as the baseline. Add regression
   classes for newly broken links/orphans, removed linked pages, vanished
   taxonomy terms, and lost sitemap entries. Keep explicit `--gate` strict.
3. Support source Git revisions without requiring ignored generated files to
   be committed, by building isolated snapshots without running repository
   scripts or changing the user's checkout.
4. Add a TUI Changes pane with page navigation, regression filtering, and
   working Diagnostics/help. Cover normal builds and watcher updates.
5. Add real-fixture goldens, differential checks, synthetic prose/broken-link
   gate tests, and an advisory repository CI comparison with saved reports.
   Strict repository enforcement requires a documented week of quiet CI.
6. Run format checks, all tests, race tests, shuffled tests, vet, and a real
   repository build/publish check. Record evidence and remaining remote-only
   acceptance criteria.

## Static-output and compatibility impact

Add `public/diff.json` and `public/diff.txt`, and reserve those paths against
content/asset collisions. Bump the manifest/build-cache schemas; legacy v1
manifests remain readable with an incomplete-coverage warning. No existing
page URLs or HTML rendering rules change. Published files must remain
deterministic and transactional. First builds establish a baseline without
classifying existing debt as new regressions. Explicit `diff --gate` exits
nonzero for regressions; the repository CI job starts advisory as required by
the issue. No external writes, pushes, PRs, or issue closure without approval.

## Remote evidence

Actual synthetic PR runs and the one-week advisory observation cannot be
claimed from local checks. Document exact reproduction and enforcement steps;
report this acceptance gap honestly before closing the issue.

## Local verification, 2026-09-30

- Full `go test ./...`, `go vet ./...`, race suite, and shuffled two-run
  stability suite pass. Coverage is 82.7% against the 75% repository floor.
- `format_check.sh` passes with golangci-lint v2.14.0 (zero findings), and
  module hygiene, file sizes, AGENTS.md, and Ask log-scrubbing checks pass.
  The preinstalled v1.64.5 linter cannot handle the repository's Go version.
  Building CI-pinned v2.13.1 from source failed due to an upstream module
  package-resolution error; v2.14.0 was installed into a disposable local
  tools directory instead. No repository lint configuration was weakened.
- A real repository-content build emits 198 pages and passes `publish-check`;
  comparison against itself is empty and passes the strict gate. Existing
  unsupported-layout fallback warnings remain, unrelated to the ledger.
- Reviewed both fixture mutation goldens and the extended manifest golden.
  Differential checker tests pass after broken-link/orphan mutations.
- Interactive TUI checks verify three prose changes, a newly broken link,
  navigation, filtering, Diagnostics return, and a 70-column pane. Screenshots
  are saved locally under ignored `SUPPORT/issue-580/`; all test sessions stop.
- Issue discussion contains no comments. No remote writes were performed
  during local implementation and validation.

Remaining acceptance evidence: publish the implementation PR, run real
prose-only and broken-link synthetic PRs, observe one advisory week, then
enable strict repository enforcement and capture the failing job run.

## Publication

The user approved publishing the implementation and synthetic test PRs.
Synthetic PRs will target `master`, include the complete implementation, and
temporarily force strict comparison in their own workflow copies. This proves
the actual CI job's pass/fail behavior without prematurely enabling strict
enforcement for the repository. They are draft, do-not-merge test branches.
#580 remains open until the rollout observation requirement is satisfied.

## Real CI evidence

- Implementation PR: https://github.com/drawmeanelephant/la-famille/pull/599.
  Go suite, lint, and advisory comparison all passed on the published tree.
- Prose-only draft control: https://github.com/drawmeanelephant/la-famille/pull/600.
  Strict job passed: https://github.com/drawmeanelephant/la-famille/actions/runs/36800978395/job/110175016854.
- Broken-link draft control: https://github.com/drawmeanelephant/la-famille/pull/601.
  Strict job failed at comparison: https://github.com/drawmeanelephant/la-famille/actions/runs/36800983594/job/110175035316.
- Downloaded both actual `change-ledger` artifacts and asserted zero prose
  regressions, no coverage warnings, and exactly one broken-link regression
  (`index`, `ledger-does-not-exist.md`, line 36), with no unrelated failures.
- Synthetic controls are draft, do not merge, and remain available for review.
  No repository-wide enforcement variable, branch protection, or issue state
  was changed. A quiet week after production rollout is still required.
