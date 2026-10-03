# Issue 626: Corpus Packs finishing slice

## Scope and ownership

Owner: Droid on `droid/review-and-fix-issue-626`.
Finish HTTPS feed acquisition, reproducible publishing, durable polling/watch,
and operator-facing demonstration instructions. Reuse the shipped v1
build/verify/diff/apply, pack-backed Ask, and local feed pull contracts.
Git-ref sources, signatures, compression, format changes, retrieval work, and
new services are out of scope.

## Dependencies and baseline

The initial checkout is older than the issue's shipped baseline. Fetch current
`origin/master` and fast-forward this clean task branch before implementation,
preserving this unique plan. Inspect `internal/pack`, CLI pack commands, feed
and Ask documentation, and existing deployment workflows.

## Implementation plan

1. Inspect the shipped contracts and existing network security helpers.
2. Add explicit opt-in HTTPS acquisition with bounded requests, cancellation,
   same-origin confined members, safe redirects, and public-address enforcement.
3. Add deterministic feed publication using canonical packs, exact-base deltas,
   automatic manifest hashes, and bounded retained versions.
4. Add watch using the pull verification path, immutable version artifacts,
   an atomic durable current pointer, deterministic polling, and Ledger reports.
5. Integrate feed generation with existing static deployment, document behavior,
   and add same-package and workflow contract tests.
6. Validate and review the complete intended diff. Prepare local compiled-binary
   evidence where possible; record concrete blockers for hosted/model proof.

## Static-output and hosting impact

The website deployment will additionally publish a current `pack-feed.json`
and bounded immutable pack/delta artifacts under a dedicated feed directory.
The manifest must only reference complete artifacts. Packs retain the existing
content-only allowlist. Local pull remains no-overwrite; watch uses its own
subscriber-state directory and does not change that contract.
Production deployment requires operator approval. No credentials or corpus
text are included in diagnostic logs.

## Validation

Same-package tests cover HTTPS security, delta selection, publication history,
polling, no-change behavior, recovery, cancellation, and state preservation.
Run narrow tests first, then `./format_check.sh`, `go test ./...`,
`go test -race ./...`, `go vet ./...`,
`go test -shuffle=on -count=2 -parallel=4 ./...`, applicable workflow/static
output checks, and `git diff --check`.

## Close-out guardrails

Do not close #626 or #583, or claim the hosted proof, before implementation is
merged and evidence accepted. Updating external issue status, publishing a PR,
and production deployment require explicit operator authorization. The final
operator handoff must explicitly record Git-ref deferral and the archive-member
transfer boundary.

## Status

- Planning: completed; fast-forwarded from `8f80022` to shipped `ffe8579`.
- Implementation: HTTPS acquisition, automatic publication, durable watch,
  transfer tracing, same-package tests, CLI integration and Pages workflow done.
- Review: complete intended change reviewed for reuse, provenance/correctness
  and efficiency. Fixed nonpublic IPv6 acceptance, exact subscriber JSON fields,
  captured-orphan identity, duplicate changed-poll archive reads, and duplicate
  publication round-trip reads. New snapshot helpers compose confined opening
  and existing verification; the target-aware round-trip helper composes
  `applySnapshots`. Local v1 behavior and explicit fallback are retained because
  they shipped on the comparison base.
- Validation: full tests, vet, race, shuffled suite, quality hook, workflow
  checks and static build/publish-check passed. Coverage is 84.5% (75% required).
  Initial quality attempt failed because installed lint was built with Go 1.24;
  a temporary v2.14.0 linter built with current Go passed with zero issues.
- Local compiled evidence: automatic publish, watch cold start, named changed
  page, corrupt-delta refusal, byte-identical old state, recovery and clean
  cancellation passed from a subscriber containing only durable pack state.
- Hosted/model evidence: blocked on approved fixture deployment, separately
  provisioned subscriber and real Ollama. No executable or responding local
  Ollama API was available. Runbook and honest status are in
  `docs/corpus-pack-https-watch-demo.md`.
- External actions: operator authorized committing/pushing this branch and
  opening one PR against `master`, with no auto-closure. Operator chose to leave
  hosted/model proof pending with the runbook. No deployment, issue update or
  closure is authorized or performed.
