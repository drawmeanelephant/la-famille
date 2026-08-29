# Issue #542 — Decompose generator.go build()

## Task

`internal/generator/generator.go` is ~979 lines and `build()` (line ~150) handles too much:
metadata/fingerprinting, concurrent rendering, stubs, asset copying, graph, search index,
feed/RSS, health, discovery. Extract focused same-package functions so `build()` is under
200 lines. Behavior must be unchanged — this is a pure refactor verified by the existing
test suite (build_content_test.go, cache_invalidation_test.go, output_collision_test.go,
publish_contract_test.go, subpath_deploy_test.go, taxonomy_nav_test.go, build_audit_test.go).

## Approach

1. Read generator.go end to end; map the phases of `build()`.
2. Extract into same-package unexported functions, smallest blast radius first:
   - render-worker body (the per-page closure) → `renderPage(...)`
   - post-render pipeline (graph, search index, feed, health, discovery) → split by concern
   - staging/swap (publish/rollback of the output directory) → `swapStagedOutput(...)` or similar
3. Shared state gets grouped into a small struct passed by pointer rather than long parameter
   lists — only if it reads cleaner than parameters.
4. No behavior changes: same log lines, same error strings, same ordering of operations.
   Tests assert on error strings and artifact presence, so ordering matters.

## Potential breaking changes to static asset generation pipeline

None intended. Asset copying, stub generation, and the staging/swap sequence keep their
current order and error messages. If any extraction forces an ordering change, stop and
re-plan.

## Verification

```bash
go test ./...
go vet ./...
go build ./...
```

Plus a manual line count check: `build()` under 200 lines.
