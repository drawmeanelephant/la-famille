# Issue #603: conservative lexical evidence coverage

## Scope

Fix topical near-miss retrieval abstention without changing frozen datasets,
thresholds, graph-route labels, or embedding algorithms. Preserve the unrelated
untracked #580 documents; do not stage or publish them in this PR.

## Reproduction

At `bb9c495`, the original lexical CLI passes 9/9 (recall 1.0000,
precision 0.4375). The hard CLI fails 7/8, solely on the absent sensor
warranty question, with recall 0.5833 and precision 0.3333.

## Plan

1. Add independent lexical/graph near-miss and answerable controls first,
   including a positively supported warranty question and non-warranty topics.
2. Test a small deterministic query-coverage guard before BM25 scoring:
   ignore conversational stop words; abstain when half or more of meaningful
   query terms have no lexical evidence. Preserve scores/order for accepted
   queries. Do not add dataset IDs, topic-specific terms, or model calls.
3. Verify frozen recall/precision and graph routes; change the tests that
   explicitly expected the known failure into strict success assertions.
4. Verify canonical no-answer without calling a completion provider, including
   graph off/on. Document the heuristic's limits, not general factual safety.
5. Run formatting/quality, all tests, vet, shuffled tests, and frozen CLI probes.
6. Review staged changes, commit, push, and create a PR if integration permits.

## Output and compatibility impact

No generator, template, static asset, or wire-schema changes are planned.
Accepted lexical queries retain ranking. Low-coverage queries now abstain,
which can reject sparse paraphrases. Embeddings remain a separate opt-in arm;
do not claim this lexical policy proves live-model correctness or dense
retrieval abstention.

## Status

Implementation and validation complete; publication pending.

- Added a strict-majority guard over unique meaningful query terms in the
  existing lexical index. Accepted queries keep their original scores/order;
  graph retrieval uses the same guard. No topic/fixture exceptions.
- Shared conversational/structural exclusions include pronouns and headings.
  The first policy iteration rejected the original heading query; treating
  structural heading vocabulary consistently fixed that regression without
  changing frozen labels. Tests also retain metadata-query controls.
- Independent controls include telescope insurance/refund near-misses, repeated
  vocabulary, minor gaps, and positively supported warranty/insurance facts.
  Server tests require zero context/citations/paths and canonical no-answer
  before completion, including off/on/off graph toggles.
- Updated old failure-expectation tests to require strict abstention. The hard
  graph comparison's unlabeled-bridge precision failure remains visible.
- Frozen original CLI: 9/9, recall 1.0000, precision 0.4375, exit 0.
  Frozen hard lexical CLI: 8/8, recall 0.5833, precision 0.3333, exit 0.
  Separate graph CLI: 6/6, recall 1.0000, precision 0.6067, comparison passes;
  graph off metrics remain 0.8667/0.4867 and single-page non-dilution passes.
  Evidence logs: `/tmp/la-famille-603-eval.b0z0Si/`.
- `./format_check.sh`, `go test ./...`, `go vet ./...`,
  `go test -shuffle=on -count=2 -parallel=4 ./...`, and
  `go test -race ./...` passed. Optional golangci-lint was skipped by the
  repository script because the binary is not installed.
- Frozen JSON datasets are unchanged. Documentation distinguishes lexical
  coverage from semantic sufficiency, dense retrieval, and live-model correctness.
- Prior #580 task plan, `findings.md`, and `spec.md` stay untracked and excluded
  from the #603 commit/PR.
