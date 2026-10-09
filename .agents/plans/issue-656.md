# Issue #656 — pack pull exits 0 when interrupted (SIGINT/context cancel)

## Problem
`setupPackPullCmd` RunE (`cmd/la-famille/pack.go`) maps `ctx.Err() ==
context.Canceled` to `return nil`, so an interrupted pull exits 0 with no pack
and no message — indistinguishable from success in scripts/CI. `pack watch`
correctly treats cancel as clean shutdown; the asymmetry is the bug.

## Fix
When `pack.PullContext` returns an error and the command context was canceled
(or timed out), return a non-nil `pull interrupted: <ctx err>` error instead of
nil. Cobra/main then exit non-zero and log the interruption via the canonical
slog error path. Real (non-interrupt) errors still propagate unchanged;
successful pulls still exit 0.

## Behavior changes / breaking notes
- `pack pull` interrupted by SIGINT/SIGTERM or context cancellation now exits
  non-zero (previously 0) and logs `pull interrupted`.
- No change to the static asset generation pipeline.

## Tests
- New `TestPackPullCanceledContextFails`: drive the real `setupPackPullCmd`
  through `ExecuteContext` with an already-canceled context and a valid local
  feed; assert non-nil error mentioning interruption, no output pack on disk,
  and no success output. Include a non-canceled control pull that succeeds.

## Validation
`go test ./...`, `go vet ./...`.
