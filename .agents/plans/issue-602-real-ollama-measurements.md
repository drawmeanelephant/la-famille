# Issue #602: real Ollama retrieval and cache measurements

## Scope and baseline

- Start from current master `6f1764839576fe45bf5f3808aa9be92f8edee29d`.
  The worktree was clean and was fast-forwarded without discarding user work.
- Read #602, AGENTS.md, Ask documentation/CLI, `internal/askeval`,
  `internal/retrieval`, and the Ollama adapter.
- Confirm merged #604 (`d13e483`) fixes lexical warranty abstention without
  changing either frozen dataset. Measure hybrid abstention independently.
- Do not change questions, labels, thresholds, scoring, retrieval defaults,
  graph retrieval, or static asset generation.
- Potential breaking changes to the static asset generation pipeline: **none**.
  Evaluation diagnostics may gain additive measurement output only.

## Execution

1. The initial loopback API check returned connection refused. The owner
   explicitly authorized installing Ollama and downloading `nomic-embed-text`.
   Homebrew inspection subsequently found an existing Ollama 0.35.1 binary
   outside PATH. Use that binary rather than reinstalling it; start a
   task-owned loopback-only daemon and use isolated model/cache directories.
2. Add only evaluation diagnostics needed to distinguish each question's
   actual mode, attempted/successful chunk and query embedding counts, provider
   time, rank time, index construction/load time, corpus identity, and index
   hashes. Test the diagnostics in `internal/askeval`, including a singleton
   chunk batch and query-time fallback. No fake embeddings are empirical evidence.
3. Compile once, then run both unchanged datasets with explicit
   `--no-embeddings`, real `--embeddings`, repeat hybrid cache-hit runs, and
   embeddings requested while the task-owned daemon is stopped. Use separate
   fresh task-owned cache directories; never clear existing user caches.
4. Capture every question, aggregates, gates/exit codes, model version/digest,
   compiler/source identity, platform, exact commands, and measurement conditions.
   Prove byte-identical warm index reuse and zero chunk re-embedding.
5. Write a reproducible report in `docs/` and a concise status/link in
   `content/docs/ask.md`, correcting its remaining stale warranty claim.
6. Run focused tests, `go test ./...`, `go vet ./...`, `./format_check.sh`,
   race/shuffle checks required by the quality skill, and `git diff --check`.
   Review the complete diff, including new files, before handoff.

## Measurement limitations and handoff

- Embedding HTTP time, ranker time, fixture/CLI time, and completion are separate.
  Eval uses `--provider fake` only to isolate retrieval/canonical no-answer checks;
  this is not a live-completion or reasoning benchmark.
- A cold vector cache is not necessarily a cold OS/model cache. State model
  residency and run order explicitly; no artificial sleeps simulate latency.
- An honest uplift, unchanged result, or regression meets the measurement goal.
  Failed hybrid abstention or other frozen gates must remain visible.
- Stop task-owned processes after measurement. Publication requires explicit
  authorization; no deployment, issue comments/closure, or existing-cache changes.
- Finish with measured outcome, files, validation, blockers, a proposed issue
  completion comment, and readiness for owner review if acceptance is met.

## Completed evidence and verification

- Eight compiled-CLI runs on both frozen datasets recorded 68 question results:
  lexical, cold/warm real hybrid, and actual unavailable-provider fallback.
- `nomic-embed-text:latest` on Ollama 0.35.1, digest
  `0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f`.
- Lexical baselines matched 1.0000 / 0.5833, including fixed warranty abstention.
  Hybrid hard recall rose to 1.0000; original recall stayed 1.0000 with reduced
  precision. Hybrid failed all three strict controls; failures remain visible.
- All seven warm indices were directly byte-compared and SHA256-verified.
  Warm chunk calls/inputs/vectors were zero; query embeddings remained 9 / 8.
- Task-owned loopback daemon stopped; existing user caches untouched. First
  Ollama startup generated its standard local identity in `~/.ollama`, disclosed
  in the report. Model download remained in the isolated task directory.
- Report: `docs/ask-retrieval-602.md`; unabridged raw evidence:
  `docs/ask-retrieval-602-results.json`; Ask status link and stale claim corrected.
- Focused/full tests, vet, race, shuffle, formatting/module hygiene/quality, and
  whitespace checks passed. Initial format check selected an obsolete linter;
  the already-installed compatible Homebrew linter passed with zero issues.
- Complete intended change and evidence payload reviewed. No measurement
  blockers remain. No deployment, issue comment, or issue closure.

## Authorized publication

- After reviewing the measurement outcome, the owner explicitly authorized
  committing, pushing this branch, and opening an evidence PR against `master`.
- Include the entire measured outcome and change scope in the PR, not just the
  instrumentation. Leave #602 open for owner review and closure; do not merge,
  deploy, comment on the issue, or use an automatic issue-closing keyword.
- Recheck branch/staged diff, run the checked-in hook with the compatible
  already-installed linter, preserve the configured Git identity, and include
  the required Factory co-author trailer.

## PR #628 file-size guard follow-up

- CI's only failing step was `Check tracked file sizes`: the single raw evidence
  JSON had 3108 lines, above the unchanged 1500-line limit. Required Go lint,
  tests, website validation, and the semantic gate passed. Complexity reporting
  was advisory. The local format check does not include this separate guard.
- The owner explicitly authorized fixing this failure and pushing a follow-up
  commit to PR #628. Do not change the guard or any measurement.
- Split each of the eight complete run objects into a readable JSON file under
  `docs/ask-retrieval-602-runs/`. Keep metadata, verification receipts, and
  ordered relative run-file references in the existing raw-evidence index.
- Update the report to explain the index and provide a reconstruction command.
  Compare the reconstructed JSON with the committed pre-split evidence for
  exact data equality, including every stdout/stderr byte and timing.
- Stage the new files before running the tracked-file guard. Run every blocking
  lint-workflow script, standard tests/vet/formatting, and staged diff checks.
  Review the complete follow-up, commit with the required co-author trailer,
  push, and inspect the new CI result. Do not close or merge the issue/PR.
- Split completed: 101-line index plus eight complete 324–428-line run files.
  Exact reconstructed JSON equals the original `e44f620` evidence, preserving
  all eight runs / 68 results and every captured output/timing value.
- Local format/lint/module hygiene, full tests/vet, technical-debt, tracked-file
  size, AGENTS.md, Ask log-scrubbing, and working/staged whitespace guards passed.
  No quality limit, Go source, dataset, or measured result changed.
