# #583: Corpus Packs, Pull from a Local Feed

## Status and ownership

- Owner: `droid/implement-pack-pull-from`.
- Baseline: fetched `origin/master`, `1af408c`, on 2026-10-02.
- Checked recent issues/PRs, epic subissues, and searches for `pack pull` and
  `local feed`: no overlapping implementation or tracking found.
- Dependencies already delivered: #612, #618, #621, #622. Do not repeat them.
- Tracking: milestone 9, #623 (feed/cold pull), #624 (update/subscriber).
- Closed milestone 8 after confirming zero open items and merged #621.

## Scope and implementation

1. Define a bounded, strict, versioned `pack-feed.json` contract for a current
   full archive path/SHA256 and delta paths/exact base archive identities.
2. Add `pack pull <feed-directory> --output <new-pack.tar>` and optional
   `--base <existing-pack.tar>`. Resolve only safe relative regular feed files,
   reject symlink-mediated paths, and consume captured verified bytes.
3. Cold pull copies the exact verified archive. Updates choose an exact-base
   delta without reading the full target payload. No match explicitly reports a
   full fallback; a broken selected delta fails. All modes check the advertised
   target archive digest before exclusive publication.
4. Reuse snapshot verification, delta application, resource limits, member
   comparison, and no-overwrite publication. Preserve unknown verified members
   and all existing v1 command behavior. Document canonical v1 builder packs as
   the delta round-trip source.
5. Add unit, safety, CLI, and compiled-binary subscriber tests. Demonstrate
   cold pull → update → verify → pack-backed Ask with fake provider, no source
   checkout or subscriber site build, and an unavailable full target on update.

## Fixed acceptance

- Cold copy and matching delta reconstruction are byte-identical to target.
- Delta mode works with the full target payload unavailable.
- No matching delta gives an explicit full-pack fallback.
- Corruption, target digest mismatch, unsafe paths, malformed feeds, and
  existing destinations leave the base unchanged and no completed new output.
- Unknown verified members and v1 behavior remain supported.
- Subscriber demonstration uses a compiled binary and fake provider.
- Run `go test ./...`, `go vet ./...`, `./format_check.sh`, race tests, and
  shuffled/repeated tests per `go-quality`.

## Boundaries and static-output impact

No generator, content, template, or static-output pipeline changes intended.
No breaking v1 pack/delta changes. No new dependencies intended.
HTTPS, Git-ref sources, watch/timers, publishing automation, signatures,
compression, page chunks, in-place updates, deployments, releases, benchmarks,
and adoption evidence are out of scope. New ideas require separate issues,
not additional closure criteria.

## Delivery

Close empty milestone 8. Create the two requested subissues:
1. Local feed contract and verified cold pull.
2. Delta selection, safe update, and subscriber demonstration.

One reviewed PR can close both. Close the new milestone only after merge.
Leave #583 open for later transport/watch work.

## Validation and evidence

Implemented the strict 64 KiB/128-delta feed contract, local CLI, verified cold
copy, exact-base delta selection, explicit full fallback, mandatory target
archive digest verification, and exclusive new-output publication.

Safety tests cover malformed/oversized feeds and archives, unsafe/duplicate
paths, selected/unselected symlinks, wrong or corrupt selected deltas,
noncanonical target archive identity mismatches, and output/base preservation.
Unknown verified members survive updates. The v1 commands/formats and module
dependencies remain unchanged.

The compiled-binary subscriber test passed with the entire publisher site and
full target removed before update. Subscriber files are only the base and updated
packs. Fake-provider Ask cites the updated **three days** interval, not the old
**seven days** interval. Evidence: `docs/corpus-pack-feed-demo.md`.

Passed full tests, race tests, vet, shuffled/repeated tests, `format_check.sh`
(temporary Go-compatible golangci-lint v2.14.0, zero issues), file-size,
AGENTS.md, log-scrubbing, and whitespace checks. An initial lint naming issue
was fixed before the passing quality run.

Three parallel simplify reviews examined the complete intended staged diff
and baseline provenance. Kept the feed-specific schema/traversal orchestration:
it composes existing strict JSON/path primitives but requires metadata-only
unselected-source checks and pinned directory handles. Kept v1/noncanonical
cold-copy compatibility and the explicit no-match fallback, all supported by
the baseline contract or supplied requirements. The bounded `feedDeltaPaths`
projection needs no additional abstraction.

Fixed the reviewed unchanged-site-manifest Ledger source bug by selecting
the base/delta member overlay, including removed-member behavior. Fixed the
blocking Unix source-open race with nonblocking confined leaf opens.
`openFeedLeaf` is a small platform-specific primitive; Windows/other platforms
keep confined opening and the common source policy. No dependency was added.
The `O_NOFOLLOW` probe showed that Go's confined opener resolves symlinks itself;
the production source policy retains Lstat/regular-file/identity checks.
Regression tests and the complete tests/race/vet/shuffle/quality suite passed
after these fixes. Linux/amd64 and Windows/amd64 pack tests cross-compiled;
runtime validation ran on Darwin/arm64.

Implementation and review are complete. The operator approved committing,
pushing this branch, and opening one PR targeting `master`.
#623/#624 and milestone 9 remain open until the reviewed PR merges; #583 remains
open for later transport/watch work.
