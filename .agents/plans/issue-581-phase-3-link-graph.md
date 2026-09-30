# #581 Phase 3: lexical link-graph expansion and multi-hop grounding

Branch: `droid/implement-phase-3-link`

## Plan

1. Start from the landed Phase 1/2 base. Preserve both existing golden JSON
   files, question labels, thresholds, and the unmodified lexical ranker.
2. Load generated outbound edges and backlinks into a deterministic, bounded
   graph retriever. Use only published corpus pages, preserve edge direction,
   prefer strongly linked relevant neighbors, and retrieve connecting routes
   between independently matching endpoints. Keep expansion opt-in and
   independent of the embedding ranker.
3. Add per-request on/off selection and a CLI default. Feed complete, cited
   route evidence to the model; verify route citations before showing an
   accessible “how I got there” visualization using the existing graph data.
4. Add separate multi-hop golden questions and a same-run lexical versus graph
   evaluation. Report recall, precision, route coverage, and cited grounding.
   Prove existing single-page precision does not regress, and retain known
   hard-dataset failures rather than weakening their gates.
5. Add package-local tests for graph loading/traversal, budgets, deterministic
   ranking, citation grounding, HTTP toggling, eval comparisons, and UI wiring.
   Run narrow tests, full `go test ./...`, `go vet ./...`, race tests, and a
   browser smoke test. Review the complete diff, commit, push, and open a PR
   against `master`; do not merge.

## Static generation and compatibility

No breaking changes to static generation. Reuse the existing `graph.json`,
`backlinks.json`, page IDs, and enriched citation URLs; do not alter their
schemas or the `/graph/` explorer output. Ask UI assets remain embedded and
local. No new dependency or remote network surface. Graph-off behavior remains
the existing retrieval path; graph comparisons explicitly use lexical scoring.

## Validation record

- Implemented graph loading, reciprocal-neighbor promotion, bounded shortest
  paths, full-source graph prompts, route citation checks, per-request toggling,
  path UI, and paired lexical-only evals. No changes to either previous golden
  file, the lexical scorer, the hybrid scorer, dependencies, or static schemas.
- New graph golden set: recall@5 `0.8667 -> 1.0000`; precision@5
  `0.4867 -> 0.6067`; all three multi-hop answers name and cite every route page.
  Both new single-page controls retain identical precision/recall.
- Frozen regression: all eight single-page questions retain recall `1.0000`
  and aggregate precision `0.4375`. The frozen hard set's known warranty
  abstention failure remains visible and is asserted by tests.
- Passed focused package tests, `go test ./...`, `go vet ./...`,
  `go test -race ./...`, and `go test -shuffle=on -count=2 -parallel=4 ./...`.
- Passed `format_check.sh` (including vet, module hygiene, technical-debt guard,
  and lint), AGENTS validation, and Ask log-scrubbing guard. Repository statement
  coverage is `83.3%`, above the unchanged `75%` CI floor.
- Passed Node syntax and dependency-free UI-controller tests: off/on/off,
  ordered A-C-B DOM, arrows, citations, hybrid-default preservation, safe text,
  and invalid/script URL handling. Node is optional for Go-only environments.
- Live browser smoke rendered the local shell and accessible toggle. Interaction
  was blocked by a disappearing desktop CDP target; creating a replacement tab
  was denied by the desktop bridge. The initial disposable build also needed
  its minimal template installed before it could run. Automated HTTP and DOM
  tests pass; no claim of a completed visual route smoke test.
- The preinstalled lint binary was built with Go 1.24 and cannot lint this
  Go 1.26 repo. Installing the CI-pinned v2.13.1 from source failed upstream
  package resolution. A private temporary v2.14.0 build ran with zero issues
  and was used by the real pre-commit/format checks. No repository/toolchain
  settings were weakened.
- Completion tests use the explicitly synthetic deterministic fake, plus a
  prompt-evidence test provider. Live Ollama reasoning quality was not measured.
