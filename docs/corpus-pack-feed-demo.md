# Local-feed pull: compiled subscriber evidence

> Historical evidence. Ask was retired in #629. Current subscriber tests
> inspect verified archive content and metadata directly, using the renamed
> `assets/testdata/pack-subscriber` fixture. The old Ask steps below no longer
> apply; use `content/docs/corpus-pack-feeds.md` for current commands.

Scope: #583, milestone 9, #623 and #624. Baseline: fetched `origin/master`,
`1af408c`, on 2026-10-02. No overlapping local-feed implementation or issue
was found before creating tracking. Existing #612/#618/#621/#622 were reused,
not reimplemented. Empty milestone 8 was closed.

## Reproduce the subscriber demonstration

```bash
go test ./cmd/la-famille -run '^TestPackPullCompiledSubscriber$' -count=1 -v
```

The same-package integration test compiles `./cmd/la-famille` into a temporary
binary, then invokes **that binary** throughout. It copies
`assets/testdata/pack-ask` to a disposable publisher and uses existing
`build`, `rag`, `pack build`, and `pack diff` commands only there to prepare
feed artifacts. This is publisher fixture setup, not a subscriber build.

From a separate, initially empty subscriber directory, it runs:

```bash
la-famille pack pull /absolute/feed --output base.tar
```

The cold result is `mode full`, seven verified members, and byte-identical to
the advertised initial archive. The publisher changes the migration survey
interval from **seven days** to **three days**, prepares the canonical target
and exact-base delta, and updates `pack-feed.json`.

Before the subscriber updates, the test deletes the **entire publisher site**
(notes, templates, configuration, public output and loose RAG archive) and
the **advertised full target archive**. It then runs:

```bash
la-famille pack pull /absolute/feed --base base.tar --output updated.tar
la-famille pack verify base.tar
la-famille pack verify updated.tar
la-famille ask --pack /absolute/subscriber/updated.tar \
  --provider fake --no-browser --port <free-loopback-port>
```

The update succeeds in `mode delta` with the full target unavailable, proving
that its payload is not read. The result equals the saved target bytes exactly;
the base still equals its original bytes. The subscriber contains only
`base.tar` and `updated.tar`, before and after Ask: no checkout, configuration,
source notes, public directory, loose RAG archive, or subscriber site build.

## Recorded results

Recorded on Darwin/arm64 with Go 1.27.1. Archive hashes can differ with binary
provenance; this run checked its own exact archive bytes, not fixed golden hashes.

| Identity | SHA256 |
| --- | --- |
| Initial full/base | `092fbebafd29ecb7833b37c1e96b3c85bed4dd7106a509b77e7b55737e5e4920` |
| Updated full/delta result | `6861bc7c7a30d0a93536208dc0ec32655d00cd7948b2965b9e2815b0e2e99e41` |

Readable update output:

```text
Pulled pack updated.tar: mode delta, 7 members
Target archive SHA256: 6861bc7c7a30d0a93536208dc0ec32655d00cd7948b2965b9e2815b0e2e99e41
Target content root: 6491804c21a9eef8c15790d982e0c8a745bbdeba7ad10d4609de4d3f420b9165
Pack diff: 0 added, 0 removed, 3 changed members
  ~ rag-content.md
  ~ search.json
  ~ site-manifest.json
```

The optional Ledger report identified one changed page (`birds`), no added or
removed pages, no graph/taxonomy changes and no regressions.
`/api/status` reported `provider: fake`, `ready: true`, `loopback_only: true`,
two documents and two chunks.

POSTing `{"question":"What is the migration survey interval?"}` to `/api/ask`
returned `status: answered`, one `[1]` source card and no dropped citations:

- Title: **Bird Observation Notes**
- URL: `/field-guide/bird-observations/`
- Excerpt: **# Bird observations Meadow bird counts use repeated field
  observations to estimate migration. The migration survey interval is three days.**

The fake provider's generated “yes” is synthetic. This proof checks updated
retrieval and citation evidence, not factual model generation. The loopback
server was terminated after the request.

## Fixed acceptance coverage

- `TestPullColdAndDelta`: exact cold bytes and delta bytes, absent full target
  on update, unchanged base, unknown binary member retained, member changes.
- `TestPullExplicitFallbackAndExactBaseIdentity`: a verified base with the same
  payload but different header bytes cannot select the old delta; full fallback
  succeeds without reading an unselected corrupt delta.
- `TestPackCommands`: command wiring, required/invalid flags, full/delta reports,
  target identity, named member changes and explicit full-pack fallback.
- `TestPullSelectedDeltaFailuresNeverFallback`: corrupt, truncated, missing,
  wrong-base and inconsistent selected deltas fail even with a valid full
  target available; target-digest mismatch fails before publication.
- `TestPullColdRetainsAlternateLayoutButDeltaRequiresAdvertisedIdentity`:
  cold pull preserves legal noncanonical headers; delta reconstruction must
  match the advertised archive digest and fails for a different layout.
- `TestPullFullFailuresAndExistingDestinations` and
  `TestPullDeltaDoesNotOverwrite`: corrupt full/base, mismatched target digest,
  existing files/directories/hard links/symlinks and base-as-output are safe.
- `TestFeedStrictSchemaAndLimits`, `TestPullRejectsUnsafeAndDuplicateDeclaredPaths`,
  `TestPullRejectsSymlinkMediatedSources`, and
  `TestPullResourceLimitsAndMalformedFeedCleanup`: strict schema, byte/count
  boundaries, unsafe paths, duplicate paths/base identities, links in selected
  and unselected sources, malformed/oversized inputs, no completed output and
  no staging files on failure.
- `TestFeedSnapshotRetainsOpenedVerifiedBytes`: destroying original input paths
  after capture cannot replace consumed base/delta bytes.
- `TestPullDeltaLedgerUsesTargetMemberOverlay`: no-change and other-member-only
  deltas read an unchanged site manifest from the verified base; replaced and
  removed site manifests retain the appropriate comparison/warning.
- `TestFeedLeafRejectsLinksAndDoesNotWaitForFIFO`: Unix source opening does not
  wait for a FIFO writer, and the source policy rejects symlinks and FIFOs.
- Existing v1 build/verify/diff/apply and pack-backed Ask regressions pass.

Canonical v1 builder packs are the supported delta byte-round-trip source.
No v1 format change, new dependency, generator change, or in-place update was
introduced. See the [feed contract](../content/docs/corpus-pack-feeds.md).

## Validation

Passed:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go test -shuffle=on -count=2 -parallel=4 ./...`
- `./format_check.sh`, including module hygiene and lint (zero issues)
- Repository file-size, AGENTS.md and Ask log-scrubbing checks
- Linux/amd64 and Windows/amd64 pack-test cross-compilation (not runtime tests)
- `git diff --check`

The installed golangci-lint v1.64.5 was built with Go 1.24; validation used
a temporary golangci-lint v2.14.0 built with the current Go toolchain.
The first lint run rejected a test variable named `real`; it was renamed and
the complete quality check passed.

Three parallel simplify reviews covered the complete intended change. Their
Ledger snapshot-source and blocking leaf-open findings were fixed with
regression tests. Unix leaf opens are nonblocking; the existing confined
traversal, Lstat, regular-file and identity checks enforce the source policy.
Go's confined opener resolves symlinks itself, so an initial `O_NOFOLLOW`
probe was not treated as sufficient link enforcement. All final validation
above passed after the fixes.

No transport/watch, publishing automation, signatures, compression, page chunks,
deployment, release, benchmarks or adoption requirements were added.
The PR remains for review, not merged. #623/#624 and milestone 9 close after
merge; #583 remains open.
