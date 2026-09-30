# Issue #581 Phase 1: Golden retrieval evaluation

## Scope

Add a checked-in JSON evaluation set for the three named fixture sites and two
small, linked fixture sites. Each site records questions, acceptable source
pages, recall@K threshold, and the current BM25-lite recall@5 baseline. Add
`ask --eval <dataset.json>` to export fixture content into temporary RAG
archives, evaluate the existing ranker without changing scoring, report
per-question results, and verify the no-answer fallback for an unanswerable
question.

## Validation

- Unit-test dataset validation, recall@K aggregation, and no-answer handling.
- Run `ask --eval` against the checked-in golden dataset and record the
  baseline values.
- Run `go test ./...`, `go vet ./...`, and `format_check.sh`.

## Static-output pipeline impact

None. The evaluation command writes only temporary RAG archives and report
output; it does not change generated site output.
