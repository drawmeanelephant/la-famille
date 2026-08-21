# Task Plan: 476-publishing-contract-docs

## Issue
#476 — Publishing: document output contract and GitHub Pages siteUrl wiring

## Scope
Docs-only reconciliation of `content/docs/publishing.md` with generator/workflow reality:

1. Document `rag-archive/` in the artifact table: `deploy.yml` runs
   `la-famille rag --output public/rag-archive` after build, but the contract
   doc never mentions it. Note it is produced by the separate `rag` command
   (`internal/ragexport.RunExport`: `rag-system.md`, `rag-config.md`,
   `rag-content.md`; default dir from `config.yaml:rag_dir`), not by `build`.
2. Mention `rag-archive/` in the "Safe to Publish" list (present only when the
   RAG step ran).
3. Reconcile remaining artifact-table rows against generator behavior
   (verified via existing tests: feed conditional, sitemap/robots/search/graph
   always, taxonomy conditional, explorer gated on `graph_explorer`).
4. Regression test so the doc gap cannot silently recur: assert
   `content/docs/publishing.md` documents `rag-archive` (pattern follows
   `TestGitHubPagesWorkflowAudit` contract-marker checks).

## Static asset pipeline impact
None — documentation plus one read-only doc-contract test.

## Verification
- `go test ./...`
- `go vet ./...`
