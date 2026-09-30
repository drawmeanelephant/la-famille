## Summary

Follow-up to merged #587; contributes to #581 **without closing the epic**.

Phase 1's original nested version-1 JSON, `Run(ctx, Options)` API, frozen golden
dataset and original tests are retained. This adds a more honest measuring stick
before embeddings or graph expansion rather than duplicating the merged PR.

- Add unique-page precision, question/dataset gates, per-class reporting,
  missing evidence and graph-route coverage; enforce the recorded recall@5 floor.
- Add two linked twelve-page content fixtures with topical decoys, disjoint
  paraphrase probes, a missing graph bridge and strict near-miss abstention.
- Build/export content-only copies in disposable projects using the real
  generator, graph and citation metadata. Confine source reads; reject content
  symlinks and prevent custom content names colliding with generated files.
- Reject unknown/trailing/oversized JSON, invalid metrics and missing/null
  answerable recall floors; validate evidence labels and real graph hops.
- Add `--eval-k`, stable diagnostics and clear aggregate-gate CLI errors.

## Recorded unchanged BM25-lite baselines

| Dataset | Answerable | Recall@5 | Precision@5 | Question gates |
| --- | ---: | ---: | ---: | ---: |
| Frozen #587 regression | 8 | 1.0000 | 0.4375 | 9/9 |
| New hard probes | 6 | 0.5833 | 0.3333 | 7/8 |

**The hard eval intentionally exits nonzero.** Warranty duration is absent but
retrieval returns topical sensor pages. That fails strict retrieval abstention;
no model fallback is invoked to mask it. Regression tests assert the known gap.
Zero-floor paraphrase measurements expose misses, not successful capability.
The graph probe finds both answer pages but misses the bridge (coverage 2/3).

Precision is relevant unique pages / retrieved unique pages within the top K
chunks, **not hits / K**. Answerable-only macro averages exclude abstention.

## Compatibility and scope

- Frozen #587 golden JSON and original tests are unchanged; project-relative
  fixture semantics and normal Ask host/content-directory behavior are retained.
- Original unanswerable questions may omit recall thresholds; answerable floors
  must be explicit. New zero-floor probes require `measurement_only: true`.
- No production ranking, prompt/citation verification, generator configuration,
  dependency or static assets change. Eval alone exercises real isolated builds.
- Probes are agent-authored synthetic keywords, not held-out natural questions.
  Page recall and graph coverage do not prove answer accuracy or provenance.
- Content-only builds do not reproduce fixture themes/config/deployment paths.
- Embeddings, hybrid fusion, bounded graph expansion and path UI remain deferred.

## Validation

- [x] `go test ./...`
- [x] `go vet ./...`
- [x] `go mod tidy -diff`
- [x] `go test -race ./internal/askeval ./internal/ask ./internal/retrieval`
- [x] `go test -race ./cmd/la-famille -run TestAskEval`
- [x] Changed Go files have no `gofmt -d` output
- [x] Original golden dataset and original tests pass unchanged
- [x] Hook formatting, vet, module hygiene and technical-debt checks passed
- [ ] Local `golangci-lint`: blocked because the installed binary was built
  with Go 1.24 and cannot load this Go 1.26 project. The user approved bypassing
  the commit hook after the other checks passed; compatible CI lint is still required.

Added coverage includes precision/recall floors, permitted baseline lifts,
aggregate-only CLI failures, K overrides, strict JSON and metric validation,
symlink confinement, source collisions, cancellation, graph/citation identity,
vocabulary leakage and invented/stale evidence labels.

## Try it

```bash
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions.json
# Expected nonzero exit for the documented near-miss abstention gap:
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions-hard.json
```

Please review the synthetic evidence labels and precision denominator before
using measured improvements as release-quality claims.
