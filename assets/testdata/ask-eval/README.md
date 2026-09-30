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

## What this gate can and cannot detect

A regression gate that cannot fail is not a gate, so the suite proves its own
teeth rather than assuming them. `internal/askeval/mutation_test.go` injects
deliberately broken rankers through `Options.Ranker` and asserts they fail:

- **Stub ranker** (returns the first chunk regardless of query) is rejected by
  both datasets, and falls below the recorded recall floor. This is a hard gate.
- **Reversed ranker** (production ranking inverted, a realistic comparator bug)
  is asserted to change measured retrieval, so a green run cannot hide an
  ordering regression.

**Known limitation, measured rather than assumed.** The frozen #587 dataset is
saturated: its fixtures have 2-4 pages, so at K=5 every candidate page is
retrieved regardless of order. Reversing the ranking changes the *order* of
retrieved pages but leaves recall at 1.0, so that dataset detects a stub
ranker but not reversed ordering. `TestSaturatedRegressionSetCannotDetectOrdering`
records this as a canary: if a future retriever regresses ordering on small
corpora, that test is the signal to grow the dataset rather than trust a green
run. The twelve-page hard dataset carries more of the ordering signal.

`Options.Ranker` is the seam the next #581 phases need to compare lexical,
hybrid and graph arms through identical gates. A nil `Ranker` uses the
production BM25-lite scorer, which remains the only behaviour any release claim
may rest on; `TestRankerSeamDefaultsToProduction` guards that the nil and
explicit paths cannot diverge.

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

## Phase 3: paired lexical graph evaluation

```bash
go run ./cmd/la-famille ask \
  --eval assets/testdata/ask-eval/golden-questions-graph.json \
  --eval-compare-graph --no-embeddings
go run ./cmd/la-famille ask \
  --eval assets/testdata/ask-eval/golden-questions.json \
  --eval-compare-graph --no-embeddings
go test ./internal/askeval -run TestGraph -v
```

The new graph file adds questions, not rewritten labels in either previous
dataset. Each multi-hop question requires the intermediate page as evidence
and sets `require_grounded_path`. The gate calls the real Ask pipeline and
checks a complete ordered route, every page name in the answer, and verified
citation cards for every route node. The deterministic fake marks route
responses as synthetic. This proves wiring, not model answer correctness.

| Dataset / class | Graph off recall@5 | Graph on recall@5 | Off precision@5 | On precision@5 |
| --- | ---: | ---: | ---: | ---: |
| New graph set | 0.8667 | 1.0000 | 0.4867 | 0.6067 |
| New multi-hop (3) | 0.7778 | 1.0000 | 0.6333 | 0.8333 |
| New single-page (2) | 1.0000 | 1.0000 | 0.2667 | 0.2667 |
| Frozen regression single-page (8) | 1.0000 | 1.0000 | 0.4375 | 0.4375 |

`observation-custody-finding` needs the dawn observation (`sensor`), courier and
sealed-vial custody (`registry`), and chloride finding (`assay`). The reverse
question follows the same links backward. `observatory-to-archive` needs the
observatory's entry point, the field log's routing link, and the kiln reading.
The controls ask only for the courier or kiln temperature. Source facts reach
the graph prompt: title-only chunks are not valid route grounding, and source
text uses a bounded share of the prompt budget instead of a 160-rune preview.

`Options.CompareGraph` runs both arms in one call, retains both reports, and
rejects embeddings/custom baseline rankers. Both arms use the same K and
unchanged labels. The comparison gates require graph question gates to pass,
aggregate recall/precision not to fall, and every single-page question's recall
and precision not to fall. The lexical arm's new grounded-route gates are
expected to fail and remain visible. A lift is reported, not assumed; a
saturated dataset cannot gain recall.

The hard dataset's labels still treat `registry` as route metadata rather than
acceptable evidence for `sensor-to-assay`. Retrieving that bridge may therefore
reduce its frozen precision metric. Do not relabel it. Its warranty abstention
failure also remains a failure. Run the hard command when investigating limits,
not as a green end-to-end answer benchmark.
