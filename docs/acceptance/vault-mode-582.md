# Vault Mode #582: shipped-contract acceptance

Accepted on 2026-10-02 against master `08f9a8e` plus this PR's narrow
link-traversal fix. Fixture: `assets/testdata/vault-acceptance/`, containing
only synthetic notes.

## Accepted scope

**Per-note `publish: false` is the supported subset-selection mechanism.**
Directory exclusion patterns are deferred, not a closure requirement.
`render: false` alone publishes raw Markdown and is not a privacy control.

This acceptance is not full Obsidian compatibility, editor integration, or
a guarantee that arbitrary private vaults and attachments are safe to
publish. Included notes still contain their authored text, including visible
link labels and wiki-target names in search/RAG source text. These are not
the excluded note's body or a published target. Assets and attachments need
separate review. No Corpus Packs changes or dependencies are included.

## Fixed matrix

All rows passed. Existing tests were reused, not reimplemented.

| Row | Evidence |
| --- | --- |
| 1. Known targets, aliases, headings | Existing `TestWikiLinksResolveAliasesHeadingsAndMissingNotes` (`internal/transform/wiki_link_test.go`) and `TestBuildWikiLinksResolveAndGenerateUnresolvedNote` (`internal/generator/wiki_links_test.go`). Compiled fixture emits `birds/`, alias `maps/`, and `birds/#method`; the target has `<h2 id="method">`. |
| 2. Clickable unresolved stub and graph edge | Same existing generator test. Fixture emits `quasar-ledger/index.html`, titled `Unresolved Note: Quasar Ledger`, linked from Home; `graph.json` contains `["index","quasar-ledger"]` and a stub node. |
| 3. Shared tag archive | Existing `TestBuildBodyTagsShareFrontmatterTaxonomyArchive` (`internal/generator/body_tags_test.go`). Fixture's body-tagged birds and frontmatter-tagged maps both appear in `tags/survey/index.html`. |
| 4. Unresolved index, backlinks, fill/rebuild | Existing `TestBuildStubAndBacklinkNavigation` (`internal/generator/vault_navigation_test.go`). Fixture initially lists the stub and shows Home as its backlink. Adding the source note removes the unresolved entry, renders normal content with backlinks, and changes both graph and explorer classifications to non-stub. |
| 5. Target-only search term | Existing `TestExtractWikiLinkTargets` (`internal/search/wiki_links_test.go`) and the generator wiki-index assertions. On the compiled fixture's initial output, `quasar` occurs only in Home's `w` search field, not title/snippet/headings/tags. The shipped search handler, exercised in a browser with a synthetic input event, returns exactly `Home → /vault/`. |
| 6. Two-note RAG + Ask source cards | Existing `TestAskVaultNotesCiteEveryExpectedSource` (`internal/ask/vault_notes_test.go`) builds and exports two synthetic notes, then asks “What do the meadow bird counts and local contour maps support?” Its all-citations fake returns exactly `Bird Observation Notes → /vault/birds/` and `Contour Map Notes → /vault/maps/`. This verifies integration and citation identity, **not real-model answer quality**. |
| 7. Excluded-note absence | Expanded existing `TestBuildExcludesUnpublishedNotesAndIncomingLinks` and `TestRunExportExcludesUnpublishedNotes`. Both rendered and raw excluded notes are absent from pages/raw output, graph/explorer, backlinks, search results, metadata/manifest, taxonomy, feed, sitemap, and RAG content members. Before and after fixture rebuild, scans of all 23 generated files plus `rag-content.md` found zero body-canary matches. Private-only tags/categories and an unresolved link originating only in the excluded note produce no output. Incoming links keep their labels but create no published target, stub, or edge. |

### Narrow failure fixed

Unwrapping an excluded link during Goldmark's AST walk stopped traversal
before later links. The acceptance regression failed because subsequent
Markdown links into excluded notes retained their `href`.

The transformer now unwraps excluded links after traversal. New
`TestExcludedLinksDoNotStopTargetTraversal` in the same package covers both
wiki-first and Markdown-first exclusions, retained labels, later public
links, and later unresolved links. The generator regression also checks a
public link after multiple exclusions.

## Compiled-binary demonstration

Run from the repository checkout. The fixture remains unchanged; only its
temporary copy is filled.

```sh
DEMO=$(mktemp -d /tmp/la-famille-vault-582.XXXXXX)
go build -o "$DEMO/la-famille" ./cmd/la-famille
cp -R assets/testdata/vault-acceptance "$DEMO/site"
BIN="$DEMO/la-famille"
SITE="$DEMO/site"

"$BIN" --project-root "$SITE" build
"$BIN" --project-root "$SITE" check
# Expected exit 1: the deliberately unresolved Quasar Ledger source link.
"$BIN" --project-root "$SITE" rag
"$BIN" --project-root "$SITE" publish-check --json
cp -R "$SITE/public" "$DEMO/public-before"
cp -R "$SITE/rag-archive" "$DEMO/rag-before"

cp "$SITE/resolved/quasar-ledger.md" "$SITE/vault/quasar-ledger.md"
"$BIN" --project-root "$SITE" build
"$BIN" --project-root "$SITE" check
"$BIN" --project-root "$SITE" check --manifest public/site-manifest.json
"$BIN" --project-root "$SITE" rag
"$BIN" --project-root "$SITE" publish-check --strict --json

go test ./internal/ask \
  -run '^TestAskVaultNotesCiteEveryExpectedSource$' -count=1 -v
```

Observed results:

- Initial build: exit 0, 6 pages, cache miss, 0 build warnings.
- Initial source check: exit 1, exactly the deliberate unresolved link at
  `index.md:15`; 0 warnings and 0 orphaned pages. This diagnostic is expected,
  not ignored. Initial `publish-check`: exit 0, `valid: true`, 23 files.
- Rebuild: exit 0, 7 pages, cache miss, 0 build warnings. Both source and
  manifest-backed checks: exit 0, 0 errors/warnings/orphans.
- Both RAG exports: exit 0. Content members initially: birds, Home, maps;
  after filling: those three plus Quasar Ledger. Neither excluded note is a
  member.
- Final strict `publish-check`: exit 0, `valid: true`, 23 files.
- Two-note Ask integration: PASS using the existing deterministic fake.
- Canary `VAULT582_UNPUBLISHED_BODY_CANARY_4d1f9a`: zero matches in initial
  output, rebuilt output, and either RAG content bundle.

Local evidence root: `/tmp/la-famille-vault-582.lpw6bG/`.
Initial output is `public-before/`; rebuilt output is `site/public/`.
RAG content is `rag-before/rag-content.md` and
`site/rag-archive/rag-content.md`. Logs are `build-{before,after}.log`,
`check-{before,after}.log`, `check-manifest-after.log`,
`rag-{before,after}.log`, `publish-{before,after}.json`, and
`ask-integration.log`. Structured assertions are `evidence-before.json`,
`evidence-after.json`, and `search-browser.json`.

The search browser used a temporary loopback-only server at
`http://127.0.0.1:18782/vault/`, serving the initial snapshot. The server was
stopped. No release, live deployment, or Ollama model was used.

## Repository validation

Go `1.27.1`, darwin/arm64. All final checks passed:

```sh
GOBIN="$DEMO/tools" go install \
  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
PATH="$DEMO/tools:$PATH" ./format_check.sh
go test ./...
go vet ./...
go test -race ./...
go test -shuffle=on -count=2 -parallel=4 ./...
git diff --check
```

The initially installed linter could not load the repository's Go 1.26
target because it was built with Go 1.24. A task-local `golangci-lint 2.14.0`
built with Go 1.27.1 was used via `PATH`; the complete quality script passed
with 0 lint issues, including formatting, vet, module hygiene, and debt-marker
checks. No repository dependency or quality configuration changed.

## Closure

Review and merge this acceptance PR, then close #582 against the scope above.
Directory patterns and other compatibility ideas are separate work, not
additional criteria for this acceptance.
