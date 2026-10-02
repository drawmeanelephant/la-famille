# Issue #583: Corpus Packs v1, Build and Verify

## Ownership and scope

- Branch: `droid/implement-corpus-packs-v1`.
- Base: current `origin/master`, `08f9a8e` (fast-forwarded before changes).
- Implement only `la-famille pack build` and `la-famille pack verify`.
- Create one milestone and two subissues of #583; one PR implements both.
- Leave #583 open for later phases. Close the subissues and milestone only
  after review and merge, with the fixed criteria met.

## Prior inspection and dependencies

Inspected `internal/ragexport`, `internal/ragfmt`, `internal/jsonutil`,
`internal/sitedata` manifest/snapshot code, taxonomy/graph output, CLI bootstrap
and version helpers. Reuse the existing configuration, build identity and
site-manifest conventions. Use only standard-library archive, JSON and hashing
facilities; no new dependencies. Change Ledger implementation is on master.
Its advisory/enforcement rollout is not a dependency.

## Design and steps

1. Record the fixed criteria verbatim in both subissues and milestone.
2. Add `internal/pack` with a deterministic uncompressed USTAR container,
   strict versioned JSON manifest, site identity, available producer build
   provenance, sorted member sizes/SHA256 hashes, and a defined content root.
3. Build only already-generated public graph/search/metadata/taxonomy artifacts
   plus `rag-content.md`. Document the exact allowlist. Never automatically
   include system/config RAG bundles, source trees, caches, workflows or secrets.
4. Verify by bounded streaming without extraction or execution. Reject malformed
   archives/manifests, unsupported schemas, unsafe/duplicate paths,
   missing/unlisted members, and size/hash mismatches. Accept unknown listed
   members with valid integrity, not unknown schema versions.
5. Add same-package regression tests, including the artisanal-ceramics fixture,
   byte-identical builds and named expected/actual tamper diagnostics.
6. Document syntax, bytes, resource bounds, trust boundary and a compiled-binary
   demonstration. Run all required checks and open the linked PR.

## Static-output impact / breaking changes

No generator output contract changes. Packs are separate local output files,
and packaging does not build a site, regenerate RAG, or alter public output.
Existing destination files must not be overwritten. Taxonomy HTML is inert data
during verification. Content-only describes file selection, not secret scanning
or authenticity; hashes cannot authenticate an archive's publisher.

## Fixed acceptance criteria

1. Build a pack from the artisanal-ceramics fixture and verify it successfully.
2. Building twice from identical inputs produces byte-identical packs,
   asserted by an automated test.
3. Altering a member makes verification fail and identify the affected
   member and expected versus actual hash.
4. Tests cover malformed archives/manifests, unsafe and duplicate paths,
   missing members, and verified unknown members.
5. Record one end-to-end demonstration using the compiled binary.
6. Run go test ./..., go vet ./..., and the repository quality checks.
   Include the regression tests in the same packages as the implementation.

## Validation

- Narrow pack and CLI regression tests during development.
- `./format_check.sh`, `go test ./...`, `go vet ./...`.
- `go test -race ./...` (filesystem changes).
- `go test -shuffle=on -count=2 -parallel=4 ./...` (new tests).
- Repository coverage, file-size, agent instructions and log-scrubbing checks.
- Compiled-binary fixture build, RAG export, two pack builds, comparison, verify.

## Status

Milestone 6 and subissues #609/#610 created, with the fixed criteria and
content-only boundary copied into each; both subissues attached to #583.
Implementation and same-package regressions complete. Format and CLI docs
added, with a compiled-binary fixture demonstration in
`docs/corpus-packs-v1-demo.md` (23 members, identical archives, successful
verification, named expected/actual tamper hashes and exit 1).

Full tests, vet, race, shuffled suite, module hygiene and repository quality
scripts passed during development. The original lint executable was
toolchain-incompatible; the CI-pinned source install failed upstream module
resolution. Temporary golangci-lint 2.14.0 built with the active Go toolchain
passes with zero issues. No project dependency changes. Final validation,
including a fresh compiled-binary demonstration, passed. Repository coverage
83.5%, pack package 88.9%; bounded fuzz run passed after 4,181 executions.
Implementation is ready for review. No scope expansion or genuine blocker.
Subissues and milestone stay open pending review and merge; #583 remains the
tracking parent for future milestones.
