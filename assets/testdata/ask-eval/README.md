# Ask retrieval evaluation — #581 follow-up to #587

The original version-1 nested dataset and public `Run(ctx, Options)` API from
#587 are retained. The original golden JSON and original tests are unchanged.
This follow-up adds precision gates, per-class summaries, graph-route coverage,
and larger synthetic fixtures before implementing semantic or graph retrieval.

## Run

From the project root, with no daemon or environment variables:

```bash
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions.json
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions-hard.json
go run ./cmd/la-famille ask --eval assets/testdata/ask-eval/golden-questions.json --eval-k 8
go test ./internal/askeval -run TestGoldenFollowUpBaselines -v
```

Eval defaults to `fake`, starts no listener and opens no browser. Normal Ask
still defaults to Ollama. Fixture paths remain **project-root-relative**. The
real generator and RAG exporter run in disposable content-only projects; caches,
archives, graph edges and citation metadata are kept outside source fixtures.

## Lexical baseline (2026-09-30)

| Dataset | Answerable | Recall@5 | Precision@5 | Question gates |
| --- | ---: | ---: | ---: | ---: |
| Original #587 regression | 8 | 1.0000 | 0.4375 | 9/9 |
| Hard | 6 | 0.5833333333 | 0.3333333333 | 7/8 |

The regression precision differs from the unmerged local prototype because its
labels are the frozen #587 labels, not expanded relevance labels. Production
BM25-lite is unchanged. Neither benchmark is evidence of answer accuracy.

The hard command **intentionally exits nonzero**: `sensor-warranty-absent`
retrieves topical pages even though no fixture page states a warranty duration.
Strict abstention requires zero retrieval plus the canonical server fallback.
Nonempty near-miss retrieval fails immediately; no model call is attempted that
could hide the failure behind fake or semantic abstention. Tests assert this
known failure rather than making an unsupported capability look green.

| Hard class | Recall@5 | Precision@5 |
| --- | ---: | ---: |
| Multi-page (3) | 0.8333333333 | 0.5000 |
| Paraphrase (2) | 0.0000 | 0.0000 |
| Multi-hop (1) | 1.0000 | 0.5000 |

Each hard site has twelve published linked pages and topical decoys. The
`drinkable-backup` probe retrieves nothing. `unpowered-chill` retrieves an
appliance decoy, but neither required evidence page. For `sensor-to-assay`, both
answer pages are found, but `registry` is missing from the real graph route
`sensor → registry → assay` (path coverage 2/3). Path coverage is separate from
evidence recall and does not prove a generated answer explains the route.

## Metric and gate semantics

- K limits **chunks before page deduplication**, matching production ranking.
  Recall is relevant retrieved pages / all required pages. Precision is relevant
  retrieved pages / retrieved unique pages, **not** hits / K. Empty retrieval
  yields zero. Averages include answerable questions only.
- `acceptable_pages` names **all required evidence**, not alternative answers.
  IDs are source-root-relative without `.md`, not slugs or generated URLs.
- `minimum_precision_at_k` adds a dataset floor; a question may override it.
  Both question and aggregate precision gates are enforced.
- `baseline_recall_at_5` is a regression floor, not a ceiling. Improvement passes.
  `--eval-k` accepts 1–100; zero retains the dataset's K. Recorded recall@5
  comparison is skipped for other depths.
- Zero recall floors require `measurement_only: true` and an explicit non-null
  `minimum_recall_at_k: 0`. They preserve known misses as probes, not successful
  retrieval claims. Missing pages and class summaries remain visible.
- Original unanswerable questions may still omit the recall field, preserving
  #587 compatibility. Every answerable JSON question must explicitly supply it.
- Evidence/path labels are checked against generated corpus pages and graph
  edges. Paths allow either link direction; invented hops are rejected.
- Strict no-answer controls test retrieval abstention plus fallback wiring, not
  a real model's ability to refuse after receiving partially related context.
  `--provider ollama` does not change ranking or make an answer-quality claim.

## Label review

These are agent-authored **synthetic keyword probes**, not held-out natural
reader questions. Intended information needs are inspectable:

| Probe | Intended need and evidence |
| --- | --- |
| `capture-and-share` / `drinkable-backup` | How is the reserve collected and shared? `cistern` describes capture/storage; `rationing` gives daily allowances. |
| `passive-cooling` / `unpowered-chill` | How are rooms kept cool without powered equipment? `thermal-mass` describes heat buffering; `night-purge` describes ventilation. |
| `sensor-to-assay` | Which observation and laboratory finding describe the same anomaly? `sensor` records R17; `assay` gives the chloride result; `registry` supplies the custody route. |
| `lot-custody-and-result` | Who delivered R17 and what did the test find? `registry` names the courier; `assay` states the result. |
| `sensor-warranty-absent` | What is the warranty duration? No page answers; sensor pages are decoys. |
| `absent-answer-hard` | Token-disjoint control used only to check fallback wiring. |

## Safety and limitations

Eval copies only content into canonical `content/`, so custom source directories
named `public`, `templates`, or `rag-archive` cannot collide with generated files.
Project-root and content-root handles confine reads with `os.OpenRoot`. Stable
symlinks inside content are rejected, including internal aliases; confinement
also prevents external reads if a link is introduced during copying. A fixture
root link cannot escape the project. JSON rejects unknown fields, trailing
values, datasets over 4 MiB, invalid metrics, duplicates and stale labels.

Fixture-specific configuration, assets, layouts and deployment base paths are
not reproduced. Real graph/metadata generation is exercised, not rendered
visual design. Page recall does not ensure the supporting fact reaches the
model: production currently excerpts only 160 runes per chunk. Audit prompt
evidence before making answer-quality claims. Preserve the original benchmark
and add versioned labels when the contract evolves; never rewrite it to flatter
a new retriever.

Embeddings, hybrid fusion, bounded graph expansion, answer provenance and path
UI remain separate phases of #581. No production ranker, prompt, citation
verifier, dependency or static generation configuration is changed here.
