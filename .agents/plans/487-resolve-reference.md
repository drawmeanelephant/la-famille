# Task Plan: 487-resolve-reference

## Issue
#487 — publish-check: resolveReference doesn't handle content-to-directory/index.html mapping
Labels: bug

## Description
`resolveReference()` in `internal/publisher/manifest.go` rejected valid links like `/docs/setup.html` because generator outputs `docs/setup/index.html` (from `content/docs/setup.md`). Resolver checked exact file but didn't fall back to `foo/index.html` when `foo.html` missing.

## Current State
- Fix already landed in `f2e0276` (PR #493, merged to `origin/master` via `5e9581a`): added `.html` -> `/index.html` fallback in `resolveReference`
- `origin/master:internal/publisher/manifest.go:221-227` already contains:
  ```go
  } else if path.Ext(candidate) == ".html" {
      dirCandidate := candidate[:len(candidate)-len(".html")] + "/index.html"
      if _, ok := files[dirCandidate]; ok {
          return dirCandidate, true
      }
  }
  ```
- `internal/publisher/manifest_test.go` has no coverage for this fallback; issue reproduction is not in regression suite.

## Goal
Ensure PR for #487 exists as standalone branch (user request to "pr what you need, we can clear other branches after"). Since fix is already on master, PR will add regression coverage for the `.html` clean-URL fallback and document behavior.

## Tasks
- [x] Verify current `resolveReference` handles `/foo.html` -> `foo/index.html` (including query/fragment stripping)
- [x] Stash unrelated dirty work (483 summary-footer) to keep branch clean
- [x] Create clean branch `fix/487-resolve-reference` from `origin/master`
- [x] Add unit test `TestCheckResolvesHtmlToIndexFallback` (and `TestResolveReferenceHtmlFallback`) to `internal/publisher/manifest_test.go`
- [x] Run `go test ./...` and `go vet ./...`
- [ ] Push branch and open PR via `gh pr create` referencing #487

## Breaking Changes
None — static asset generation pipeline unchanged; only validation logic already tolerant. Added test is additive.

## Verification
- `go test ./internal/publisher -run TestCheckResolvesHtmlToIndexFallback -v` — PASS
- `go test ./internal/publisher -run TestResolveReferenceHtmlFallback -v` — PASS
- `go test ./...` — PASS
- `go vet ./...` — clean

## Handoff
After PR merged, local checkpoint branches (`t3/checkpoints/*`, stale `t3code/*`, `474-publishing-validation-summary` etc.) can be pruned per user instruction.
