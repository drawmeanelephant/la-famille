# #583: Corpus Packs, Local Diff and Apply

## Ownership and base

- Branch: `droid/implement-corpus-pack-diff`.
- Start: current master `ffdc421`, including merged #612.
- Create one milestone and two subissues of #583; one reviewable PR may close
  both after review and merge. Keep #583 open.
- Do not edit Change Ledger rollout policy.
- Use standard-library facilities only; no new dependencies.

## Bounded implementation

1. Create tracking for pack comparison/deterministic delta creation and safe
   application/round-trip proof, preserving the fixed acceptance below.
2. Reuse the v1 bounded USTAR/path/manifest verifier and canonical header
   writer. Retain verified input bytes in private opaque archive snapshots,
   with offsets into open descriptors, never extracted payloads or reopened
   original inputs. Hash the entire base archive for exact-base binding.
3. Define delta schema v1: versioned metadata, exact base archive SHA-256,
   target manifest SHA-256, sorted added/changed/removed paths, complete target
   manifest bytes, and only added/changed payloads. Numbered payload headers
   avoid collisions with verified unknown member names or path-length limits.
4. Add `pack diff <before.tar> <after.tar> --output <delta.tar>` and `--json`,
   with member changes and existing Change Ledger semantic comparisons when
   supported site manifests are available.
5. Add `pack apply <base.tar> <delta.tar> --output <result.tar>`. Validate the
   delta's inventory against the exact base and target, carry forward verified
   unchanged unknown members, and recheck every copied payload. Verify a
   temporary reconstructed pack before exclusively publishing a new file.
6. Add same-package unit/security regressions and a real builder fixture
   proving byte identity to the target. Document the format and record a
   compiled-binary demonstration, then run all required quality checks.
7. Review the complete diff, commit, and open one PR with evidence. Leave the
   subissues and milestone open until review and merge.

## Static-output impact / byte contract

No generator output changes or v1 pack schema changes. Existing build/verify
behavior remains covered. New local output files never replace an existing
destination. Canonical v1 builder packs round-trip byte-identically; arbitrary
alternate tar-header layouts are not preserved. `rag-content.md` is one member:
editing one page can replace the complete corpus payload. No per-page storage
or page-sized download promise.

No Ask integration, pull/watch, subscriptions, network features, signatures,
compression, deployment, releases, or per-page repackaging.

## Fixed acceptance

1. Tests cover no-change, addition, modification, and removal.
2. Unchanged payload members are absent from the delta.
3. Applying a delta between canonical v1 builder packs produces bytes
   identical to the target full pack, asserted in a fixture/golden test.
4. Repeated delta creation is byte-identical.
5. Wrong-base, corrupted, truncated, unsafe-path, and oversized inputs fail
   without modifying the base or leaving a completed destination.
6. Existing build/verify behavior and v1 compatibility remain covered.
7. Record a compiled-binary build → diff → apply → verify demonstration.
8. Run `go test ./...`, `go vet ./...`, and repository quality checks.

## Validation

- Focused pack and CLI tests.
- `./format_check.sh`, `go test ./...`, `go vet ./...`.
- `go test -race ./...`.
- `go test -shuffle=on -count=2 -parallel=4 ./...`.
- Repository coverage, file-size, agent instructions, log-scrubbing checks.
- Compiled binary: fixture site build/RAG, base pack, edited target pack,
  repeated diff, apply, verify, `cmp`/SHA-256 proof.

## Status

Milestone 7 created with subissues #613 and #614 attached to #583. The fixed
criteria and member-only scope were copied into the milestone and both issues.

Implementation and same-package tests complete. Existing v1 tests pass.
The real artisanal-ceramics builder fixture/golden proves target byte identity
and repeated delta identity: 1 added, 1 removed, 8 changed members; 9 delta
payloads and 14 unchanged target payloads omitted. Ledger reports the edited
page, changed taxonomy term and added graph edge without changing its policy.

Compiled-binary demonstration recorded in `docs/corpus-pack-deltas-demo.md`.
Both byte comparisons passed; wrong-base and corrupt-delta attempts exited 1,
left no destination, and retained the exact base hash.

Full tests, vet, race, shuffled suite, format/module hygiene/lint and repository
coverage/file-size/agent/log-scrub/debt guards passed. Coverage: 83.8% overall,
89.4% pack. Bounded delta and full-pack fuzz runs passed. A temporary
golangci-lint 2.14.0 built with the active Go 1.27.1 toolchain supplied the v2
quality check; project modules/configuration were unchanged.

Complete implementation diff reviewed, including the v1 verifier refactor,
snapshot consumption, delta inventory/base binding and exclusive publication.
No generator, Ledger policy, module, release or deployment changes. Ready for
one reviewable PR linking #613/#614. Leave #583 open and do not close the
subissues or milestone before review and merge.
