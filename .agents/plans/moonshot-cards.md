# Moonshot Epic Decomposition: #579–#583

## Task plan

Read the full bodies of issues #579–#583 and split their stated phases into
independently reviewable implementation cards. Each card below has a bounded
scope, testable acceptance criteria, size, and explicit sub-card dependencies.
This plan does not create GitHub sub-issues or implement any code.

Potential static-output pipeline changes are confined to the acceptance
criteria of the individual future cards. No static-output changes are made by
this planning PR.

## Epic #579 — The Homestead Program

### #579.1 — Build three candidate sites

**Scope:** Create three genuinely different, minimal-but-complete sites under
`sites/`, each with real content, taxonomy, assets, and a distinct structural
challenge such as a series, nested section, or chronological archive. Document
each site's audience and why it is not a copy of la-famille.

**Acceptance criteria:**
- Each candidate has at least 40 real pages, populated taxonomy, real assets,
  and one documented structural challenge.
- `sites/README.md` records each candidate's audience, pitch, and rationale.

**Size:** L
**Dependencies:** None.

### #579.2 — Publish and benchmark candidate sites

**Scope:** Publish all three candidates and add CI builds for each. Record
cold and incremental build timings and graph-explorer behavior at realistic
page counts.

**Acceptance criteria:**
- Three live URLs are recorded in `sites/README.md`.
- Each published `public/` tree contains at least 40 pages, populated
  `/tags/` and `/categories/`, a `/graph/index.html` that renders without
  freezing at the real node count, and a coherent
  `public/rag-archive/rag-content.md`.
- CI builds all three sites and the README records the requested timing and
  graph-explorer observations.

**Size:** L
**Dependencies:** #579.1.

### #579.3 — Select and harden the winning site

**Scope:** Select the destination using the bake-off and reader evidence, then
move it to a first-class top-level site with its own `config.yaml`. Add its
deployment workflow and publication checks.

**Acceptance criteria:**
- The selected destination and evidence for the choice are recorded.
- The top-level site has its own configuration and a live URL.
- Its CI runs `build`, `publish-check`, and `check` successfully.

**Size:** M
**Dependencies:** #579.2.

### #579.4 — Populate the winner and record friction

**Scope:** Write useful, non-placeholder content for the selected site and
record the generator problems that this real content exposes. Keep the list
focused on reproducible friction rather than speculative feature requests.

**Acceptance criteria:**
- A stranger can read the live site and get value from its real content.
- `HOMESTEAD.md` lists each observed gap with a reproduction and ranks gaps by
  how many pages encounter them.

**Size:** L
**Dependencies:** #579.3.

### #579.5 — Close the winner's highest-impact gaps

**Scope:** Implement the top few reproducible gaps recorded in
`HOMESTEAD.md`, in their dependency order. Add a regression fixture and
same-package unit test for every gap selected.

**Acceptance criteria:**
- Every selected gap has a regression fixture and named unit test.
- The winning site builds with the release binary, and the selected gaps no
  longer require the recorded workarounds.

**Size:** L
**Dependencies:** #579.4.

### #579.6 — Make the site maintainable by contributors

**Scope:** Document the CLI/TUI authoring loop and contribution process, and
make `check` a CI gate with a scheduled rebuild. Verify the binary-only path
using a release archive and a contributor working through a real content
change.

**Acceptance criteria:**
- CI checks broken links, orphans, and missing metadata, and a scheduled
  rebuild is configured.
- A fresh contributor can use the released binary with `--project-root` to
  submit a content correction that passes CI.
- The site has completed an end-to-end content contribution through review.

**Size:** L
**Dependencies:** #579.5.

## Epic #580 — The Change Ledger

### #580.1 — Project existing build data into a site manifest

**Scope:** Define a canonical manifest representation using the existing
  graph, checker, site-data, and search packages rather than duplicating their
  models. Include each page's identity, URL, title, date, tags, categories,
  outbound internal links, inbound link count, and asset references.

**Acceptance criteria:**
- A manifest entry exists for every built page and contains every listed
  field.
- Repeated fields and pages have a documented canonical ordering and the
  model contains no build timestamps in its diffable data.

**Size:** M
**Dependencies:** None.

### #580.2 — Write and cache the deterministic manifest

**Scope:** Serialize the canonical representation to
  `public/site-manifest.json` and retain it in the build cache. Add a known
  fixture and tests that guard both the golden representation and no-op
  determinism.

**Acceptance criteria:**
- The known fixture's manifest matches a checked-in golden file.
- Two builds of the same tree produce byte-identical manifest files.
- JSON keys and list contents are emitted deterministically.
- The manifest is retained in the build cache.

**Size:** M
**Dependencies:** #580.1.

### #580.3 — Prove manifest parity with `check`

**Scope:** Allow the checker to consume the manifest as specified by the
  epic, and compare its orphan and broken-link results with the existing
  checker path. Keep the existing checker behavior unchanged.

**Acceptance criteria:**
- A fixture with known orphans and broken links produces identical findings
  through the existing and manifest-backed paths.
- A differential test fails if either path omits or adds a finding.

**Size:** M
**Dependencies:** #580.2.

### #580.4 — Compare manifests with `la-famille diff`

**Scope:** Add `la-famille diff <a> <b>` for the input forms listed in the
  epic: output directories, manifests, or Git refs. Emit structured JSON and
  a readable summary covering page, metadata, taxonomy, link, graph, orphan,
  and broken-link changes.

**Acceptance criteria:**
- The command accepts each documented input form and has a machine-readable
  JSON mode.
- Golden fixture tests cover an edited/renamed page, a link-set and graph-edge
  change, and a newly broken link without misreporting an unrelated existing
  orphan.

**Size:** L
**Dependencies:** #580.1, #580.2, #580.3.

### #580.5 — Add the regression gate

**Scope:** Add `la-famille diff --gate` for the regression categories named
  in the epic, including newly broken links, new orphans, removed linked
  pages, vanished taxonomy terms, and lost sitemap entries. Wire the gate
  into CI in a way that distinguishes regressions from prose-only edits.

**Acceptance criteria:**
- Each listed regression category produces a non-zero gate result.
- A synthetic broken-link change fails the CI gate, while a prose-only change
  passes.
- The Homestead site can run the gate as the real-site example.

**Size:** M
**Dependencies:** #580.4; epic-level input from #579 is needed for the
Homestead-site example.

### #580.6 — Add the TUI Changes pane

**Scope:** Present the last-build comparison in a TUI Changes pane, reusing
  the diff representation instead of introducing a second diff model. Support
  page navigation and a regressions-only filter.

**Acceptance criteria:**
- A prose-only fixture displays changed pages with no regressions, and a
  broken-link fixture displays the regression.
- `j`/`k` navigation and the existing `d` action work from the pane.

**Size:** L
**Dependencies:** #580.4, #580.5.

## Epic #581 — Ask This Site: Retrieval, Revisited

### #581.1 — Add deep-link retrieval fixtures

**Scope:** Add the two new fixture sites required by the eval phase, with
  content and cross-page links that exercise a deep link structure. Keep them
  small enough for deterministic, fast test runs.

**Acceptance criteria:**
- Both fixtures build through the existing fixture/test path.
- Each fixture's linked content supports a question whose evidence spans
  multiple pages; that question is defined in #581.2.

**Size:** M
**Dependencies:** None.

### #581.2 — Check in golden questions and thresholds

**Scope:** Create checked-in JSON questions for `anchor-links`,
  `artisanal-ceramics`, `clean-urls`, and the two new fixtures. Specify
  acceptable source pages and a minimum recall@K for every fixture set.

**Acceptance criteria:**
- Each of the five fixture sites has a JSON question set with acceptable
  source-page identifiers and an explicit minimum recall@K.
- The dataset includes an unanswerable question whose answer is absent from
  its fixture corpus.
- The dataset is validated against the fixture page identities.

**Size:** M
**Dependencies:** #581.1.

### #581.3 — Implement the retrieval eval runner

**Scope:** Add the `ask --eval` command or equivalent benchmark runner to
  execute the current, unchanged lexical retriever against the golden data.
  Report per-question results and aggregate recall without requiring a live
  model.

**Acceptance criteria:**
- The runner loads every checked-in fixture/question set and evaluates the
  current retriever.
- Output includes a per-question pass/fail table and aggregate recall@5.
- The deterministic fake provider can run through the harness where a
  provider is required.

**Size:** M
**Dependencies:** #581.2.

### #581.4 — Record the baseline and test abstention

**Scope:** Record the current BM25-lite recall@5 as the Phase 1 baseline.
  Test that an unanswerable golden question uses the existing no-answer
  fallback instead of producing a guessed answer.

**Acceptance criteria:**
- The baseline number and the command that generated it are checked in with
  the dataset or eval documentation.
- An automated test verifies the unanswerable question returns the
  no-answer fallback.

**Size:** S
**Dependencies:** #581.3.

## Epic #582 — Vault Mode

### #582.1 — Resolve wiki links and generate unresolved-note stubs

**Scope:** Recognize `[[Wiki Links]]`, `[[target|alias]]`, and
  `[[target#heading]]` during transformation and resolve known targets to
  their page URLs. Route unresolved targets through stub generation and label
  those pages as unresolved notes.

**Acceptance criteria:**
- A fixture renders working links for a known target, an alias, and a heading
  target.
- An unresolved target generates a visibly titled stub that appears in the
  graph with the correct inbound edge.

**Size:** L
**Dependencies:** None.

### #582.2 — Collect inline tags into taxonomy

**Scope:** Extract body `#tags` and feed them into the same taxonomy used by
  frontmatter tags. Preserve existing frontmatter tag behavior.

**Acceptance criteria:**
- A bare body tag appears in the expected `/tags/` archive.
- A fixture page using a body tag and one using a frontmatter tag are both
  represented in the same taxonomy.

**Size:** M
**Dependencies:** None.

### #582.3 — Add stub and backlink navigation

**Scope:** Add an unresolved-notes index and backlinks to note pages, using
  the existing graph/backlink data. Ensure resolving a generated stub by
  adding its source note updates the navigation on rebuild.

**Acceptance criteria:**
- The stub index lists every generated stub and links to each one.
- Each note page shows its inbound links.
- Filling a stub removes it from the index and makes it a normal graph page.

**Size:** M
**Dependencies:** #582.1.

### #582.4 — Include wiki-link targets in search

**Scope:** Make search index wiki-link targets so readers can find a page by
  the note name it links to, even if that text is not otherwise indexed. Keep
  the change within the existing search output and retrieval path.

**Acceptance criteria:**
- Searching for a term present only in a `[[wiki-link]]` target returns the
  source page.
- Existing search behavior remains covered by package tests.

**Size:** M
**Dependencies:** #582.1.

### #582.5 — Support Ask citations for vault notes

**Scope:** Run RAG and Ask over a fixture vault and present source cards named
  and linked as notes. Reuse the existing citation and `SourceCard` path.

**Acceptance criteria:**
- A multi-note fixture question returns a citation card for every expected
  source note.
- Each card uses the note title and generated note URL.

**Size:** M
**Dependencies:** #582.1.

### #582.6 — Publish a selected vault subset

**Scope:** Add the exclusion or selection mechanism required to publish only
  chosen vault directories. Ensure excluded notes do not appear in generated
  graph, search, or link outputs.

**Acceptance criteria:**
- A fixture exclusion removes the selected note from the graph and search
  index and removes links into that note from the published subset.
- `check` reports zero orphans for the published-subset fixture.

**Size:** L
**Dependencies:** #582.1.

## Epic #583 — Corpus Packs

**Epic-level prerequisites:** Issue #583 says not to start before #579
(Homestead) and #580 (Change Ledger) provide a real site and proven
diff/snapshot primitives. The sub-card dependencies below are in addition to
those epic-level prerequisites.

### #583.1 — Define the pack format and build command

**Scope:** Define a documented, versioned `.lfpack` container with a
manifest, content root, and per-member path, size, and hash. Build a pack from
the RAG payload and the graph, backlinks, search, metadata, and taxonomy
artifacts.

**Acceptance criteria:**
- `la-famille pack build` creates one `.lfpack` from the
  `artisanal-ceramics` fixture.
- The manifest records schema version, site identity, build provenance, and
  each member's path, size, and hash.
- The pack's content root is derived deterministically from its members.
- The container choice uses the standard library and is documented.

**Size:** L
**Dependencies:** Epic prerequisites #579 and #580; no sub-card dependencies.

### #583.2 — Verify pack integrity and compatibility

**Scope:** Add pack verification that checks member hashes against the
manifest and make the reader tolerate unknown future members. Keep the
integrity error actionable by naming the affected member.

**Acceptance criteria:**
- `la-famille pack verify` accepts a valid pack and reports each edited
  member with its expected and actual hash.
- A reader test proves an unknown future member does not prevent opening a
  valid pack.
- Rebuilding the same fixture content produces byte-identical packs.

**Size:** M
**Dependencies:** #583.1.

### #583.3 — Diff packs and apply deltas

**Scope:** Compare packs by member, report member and graph/taxonomy changes,
and produce a delta pack containing changed members and an updated manifest.
Support applying the delta to a base pack.

**Acceptance criteria:**
- A one-page edit produces a delta with only that page's chunk and the
  affected manifest/graph members.
- Applying the delta yields a pack byte-identical to a full rebuild of the
  edited fixture.
- A golden fixture covers the diff-and-apply round trip.

**Size:** L
**Dependencies:** #583.2.

### #583.4 — Let Ask read a verified pack

**Scope:** Teach Ask to load a `.lfpack` directly so a subscriber can query a
versioned corpus without unpacking or building `public/`. Preserve local
citation behavior for pack members.

**Acceptance criteria:**
- Ask loads a verified fixture pack without a loose `rag-archive/` or
  `public/` directory.
- A question about edited pack content returns a citation linked to the
  corresponding source page.

**Size:** M
**Dependencies:** #583.2.

### #583.5 — Pull and update subscribed packs

**Scope:** Add on-demand pulls from local directories, Git refs, or explicitly
opted-in HTTPS sources. Apply available deltas safely and report added,
changed, and removed pages.

**Acceptance criteria:**
- A subscriber fixture pulls an upstream edit by delta and reports the
  changed page using an on-demand pull.
- Local-directory and Git-ref sources are covered; HTTPS fetching is covered
  as explicit opt-in.
- A corrupted delta is rejected without changing the local pack, which
  continues to verify.
- Remote HTTPS fetching requires explicit opt-in; local sources work without
  network access.

**Size:** L
**Dependencies:** #583.3, #583.4.

### #583.6 — Publish packs and watch for updates

**Scope:** Publish a pack as a build artifact that a second machine can fetch
  by URL. Add `pack watch` to poll for updates and summarize changed pages.

**Acceptance criteria:**
- A second-machine test fetches the published pack by URL.
- `pack watch` reports changed page names within one configured poll interval
  after an upstream rebuild.

**Size:** M
**Dependencies:** #583.5.
