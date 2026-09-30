# Issue #580 Phase 1: Snapshot and normalize

## Scope

Generate a deterministic `public/site-manifest.json` for content pages from
the existing content, graph, metadata, and checker data. Include identity,
public URL, title, date, sorted tags/categories, outbound internal-link
references, inbound-link count, and asset references. Retain the generated
manifest snapshot in the local build cache, and let `check --manifest` use its
recorded broken links and inbound counts while preserving the other existing
content checks.

## Validation

- Add a golden manifest for a known fixture.
- Prove two clean builds of the same source tree emit byte-identical
  manifests.
- Differentially compare manifest-backed and source-backed broken-link and
  orphan findings.
- Run `go test ./...`, `go vet ./...`, and `format_check.sh`.

## Static-output pipeline impact

Additive only: builds gain `public/site-manifest.json`. Existing HTML, URLs,
and generated assets remain unchanged. The private local build-cache schema
will retain the manifest snapshot for later diff work.
