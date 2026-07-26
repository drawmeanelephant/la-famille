---
title: "Release and Fixture Tests"
author: "La Famille maintainers"
date: "2026-07-26"
---

# Release and Fixture Tests

The integration tests in `cmd/la-famille` build representative sites through the real generator and inspect the resulting files. They are the publishing-contract layer above unit tests in `internal/`.

## Automatic fixture harness

`fixture_test.go` walks `assets/testdata/sites/`. A fixture with an `expected/` directory is built in a temporary output directory and compared against its expected files. The harness normalizes generated page paths under `expected/pages/` to the corresponding output location, while keeping JSON, XML, and other artifacts directly comparable.

This makes adding a fixture low ceremony: create the site input and expected output, then run the fixture test. The expected files are the contract, so intentional output changes require an explicit fixture update.

## Release smoke test

`release_smoke_test.go` builds the release fixture and checks cross-cutting output such as rendered HTML, canonical and Open Graph metadata, graph/backlink data, search, taxonomy, feed, sitemap, robots, assets, and raw or excluded content. It also checks deterministic output by rebuilding and comparing the relevant artifacts.

## Boutique dogfood test

`artisanal_ceramics_test.go` builds a more realistic boutique site with nested content, categories, tags, custom slugs, unrendered notes, and publishing metadata. It verifies that a site shape closer to real use still produces coherent HTML and supporting indexes.

## What these tests do not replace

Fixture tests do not replace focused parser, renderer, path-safety, or policy tests. They answer a different question: did the complete build pipeline produce the expected publishing surface for a representative site?

Run the focused layer with:

```bash
go test ./cmd/la-famille -run 'TestFixtures|TestReleaseSmoke|TestArtisanalCeramics'
```

Run all validation before merging:

```bash
go test ./...
go vet ./...
```
