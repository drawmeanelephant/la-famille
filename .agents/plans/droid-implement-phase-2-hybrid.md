# Issue #581 Phase 2: optional hybrid retrieval

## Scope and baseline

Fast-forwarded from `origin/master` before changing files. The unchanged
Phase 1 harness reports recall@5 1.0000 (frozen, saturated) and 0.5833
(hard, with the documented warranty abstention failure). Do not edit either
dataset, its questions or its thresholds to claim improvement.

Add explicit `--embeddings` and `--embedding-model` opt-in, with
`--no-embeddings` overriding the opt-in. Ollama `/api/embed` shares the
loopback-only rule used by completions. Build a deterministic per-chunk flat
vector index alongside the private build cache; validate the build
fingerprint, model, chunk IDs and corpus digest on load. For eval's temporary
builds, use a stable user-cache location keyed by site/dataset and the
fixture's corpus digest instead of retaining generated fixtures. Fuse
lexical and cosine-similarity rankings with reciprocal rank fusion;
unavailable embeddings fall back to unchanged BM25-lite.

## Verification

- Unit tests in llm, retrieval, ask, askeval and CLI packages cover network
  guard, malformed model response, fusion ordering and ties, cache hit and
  invalidation, opt-in/offline fallback, and flag precedence.
- Rerun both golden harnesses with `--no-embeddings`, compare exact numbers
  and offline behavior; run with embeddings if a local Ollama model is
  available and record the measured result, not a manufactured lift.
- Prove an unchanged index on a second run and fewer embedding calls /
  shorter time; run `go test ./...` and `go vet ./...`.

## Measured outcome in this worktree

The frozen set stays 1.0000 and the hard set stays 0.5833 with
`--no-embeddings`. Local Ollama is not installed/running here; opted-in eval
correctly reports lexical fallback, also 0.5833, not a claimed embedding
lift. The hard set's pre-existing warranty-abstention failure remains. A
deterministic delayed embedder confirms index reuse (first construction
~152 ms, second ~67 µs, byte-identical index, zero re-embedded chunks),
but is not a real-model quality measurement. Run the documented unchanged
hard eval with a local Ollama embedding model to determine actual recall
and precision; RRF alone cannot promise monotonic recall.

## Static-output impact

No changes to the generator, fingerprint algorithm or published files. A
private `.la-famille-vectors.json` file beside `.la-famille-cache.json`
contains local text-derived vectors and must be gitignored and never
published. The publisher now rejects that file if accidentally copied into
the static output, an intentional change to the publish validation contract,
not to the static generation pipeline. Rebuilding after a fingerprint change
invalidates it. The
only potential compatibility change is that opt-in hybrid ranking can change
the selected chunks and citations; default and `--no-embeddings` remain
lexical.
