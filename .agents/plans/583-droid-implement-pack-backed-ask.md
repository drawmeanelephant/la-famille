# #583: Corpus Packs: Pack-backed Ask

## Baseline and scope

- Branch: `droid/implement-pack-backed-ask`.
- Fetched `origin/master` and fast-forwarded to `4b95932` (#612 and #618).
- First check: Ask has no `--pack` flag or verified pack corpus input. No
  subsequent merged PR implements it.
- Build/verify and diff/apply are complete. Do not duplicate their tracking,
  plans, demonstrations, or implementations.
- Static asset generation pipeline breaking changes: none expected.

## Tracking

Create one milestone, **Corpus Packs: Pack-backed Ask**, with two subissues
of #583: verified corpus loading; Ask wiring and citation evidence. One
reviewed PR may close both. Close the milestone only after merge. Keep
#583 open for later subscription work.

- Milestone 8: https://github.com/drawmeanelephant/la-famille/milestone/8
- #619: verified corpus loading.
- #620: Ask wiring and citation evidence.

## Implementation

1. Reuse #618's verified opaque snapshot and consume its verified member
   bytes, never reopen the input path or extract/execute payloads.
2. Adapt retrieval corpus loading to reuse RAG parsing, generated metadata,
   enrichment, ranking, providers, and citations without local fallbacks.
3. Add `ask --pack /absolute/path/corpus.tar` independently of config/site,
   public output, and loose RAG directories. Reject conflicting directory
   and rebuild flags explicitly.
4. Keep directory-backed Ask behavior unchanged and preserve source-card
   titles and URLs, including deployment subpaths.

## Fixed acceptance and validation

- Fixture pack yields the expected corpus.
- Two-note fake-provider integration proves source-card titles and URLs.
- Ask reads an updated fact from an applied pack after deleting the
  original source and generated directories. This is Ask integration
  coverage, not revalidation of diff/apply.
- Corrupt, unsupported, missing-content, and malformed payloads fail clearly.
- Snapshot consumption and cleanup have regression tests.
- Existing directory-backed Ask tests stay unchanged and pass.
- Record a compiled-binary demonstration with a fake provider.
- Run `./format_check.sh`, `go test ./...`, `go test -race ./...`,
  `go vet ./...`, and `go test -shuffle=on -count=2 -parallel=4 ./...`.
- Review the full diff and open one PR with closure and evidence links.

## Exclusions

No pull/watch, remote downloads, signatures, retrieval improvements,
benchmarks, TUI expansion, deployments, or release work. New ideas do not
expand acceptance.

## Delivered and validated

- `pack.LoadCorpus` uses #618's existing `loadPack` snapshot and consumes
  verified member section readers, with cleanup before returning.
- Retrieval shares directory-backed parsing/chunking/enrichment/graph code.
  Pack mode is strict, resolves one consistent content root from the generated
  manifest, and preserves authoritative generated titles and URLs.
- CLI/bootstrap and HTTP server support pack-only input without reading config
  or falling back to local site files. Directory-backed Ask remains unchanged.
- Acceptance tests cover the fixture corpus, two-note citation integration,
  applied-pack updated facts without original directories, malformed payloads,
  snapshot replacement/cleanup, and conflicting flags.
- Compiled fake-provider evidence: `docs/pack-backed-ask-demo.md`, repeated
  against the final implementation after the review fix.
- Two-cluster code review covered all changed implementation, tests, fixtures,
  and docs. It found overlapping source suffixes collapsing identities.
  Reproduced with a failing test, fixed by consistent-root resolution, and
  verified the regression and full checks. No other findings were returned.

Final checks passed:

```text
./format_check.sh
go test ./...
go test -race ./...
go vet ./...
go test -shuffle=on -count=2 -parallel=4 ./...
git diff --check
```

The initially installed `golangci-lint` 1.64.5 was incompatible with the
repository's Go version. Built v2.14.0 into a temporary directory and reran
the unchanged quality script successfully, with zero lint issues. No quality
checks were skipped and no installed user tools or module dependencies changed.

Milestone 8 and #619/#620 stay open until the PR merges. #583 stays open.
