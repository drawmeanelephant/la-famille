# La Famille — Moonshots

> **Retirement update (#629, 2026-10-04):** Ask This Site and its model,
> retrieval, and evaluation surfaces are retired. The Ask proposals and
> assistant-dependent phases below are historical, not active roadmap work.
> Corpus Pack generation, verification, diffing, and delivery continue in #583
> independently of an assistant. The original proposals and reviews are
> retained for context.

> **Read the [adversarial review](#adversarial-review-2026-09-29) before acting
> on this catalogue.** The original proposals below are preserved so their
> claims, costs, and ranking can be inspected. The review challenges them
> against the code and current publishing evidence; the old scorecard is not
> a validated investment recommendation.
>
> Next-step brief: [Homestead's first rung — the existing bilingual site](#homestead-first-rung-the-existing-bilingual-site).
> Evidence now located: [launch-input register](#homestead-launch-input-register).

*Strategic, moonshot-level bets for la-famille. This document proposes; it
implements nothing.*

A moonshot is not a feature. It is a **multi-phase subsystem or product bet**
— weeks of work, an entire new surface, and a demo you can show rather than a
flag you can flip. Everything below is sized in rough **agent-days** (one
agent-day ≈ one focused working day of implementation plus tests, docs, and
review). Costs are deliberately honest: they include the boring 40% that makes
a feature shippable.

Every proposal is grounded in code that already exists. Where a phase would
extend a real package, the package is named.

---

## The honest anchor

The repository's own assessment is that the next honest step is **shipping a
real site built with la-famille**, and that no destination has been selected
yet. That is the correct read, and this document treats it as the spine.

The evidence:

- `content/` today is the project's *own* docs, showcase, `jules/` devlog,
  and `meta/` roadmap. It is a self-referential project site. It has never
  built a site about ceramics, a church newsletter, a research lab, or a
  recipe archive.
- The content model is a **fixed 17-field struct** (`internal/content.FileMeta`:
  `Title, Author, Date, Tags, Categories, Slug, Layout, Description, Image,
  Render, VideoScript, AnimationCues, SoundtrackTheme, ComplianceModal,
  RelPath, Content, Rest, Warnings`). A grep for `Pagination`, `Series`,
  `Redirect`, `Alias`, `I18n`, custom-frontmatter passthrough, and data files
  returns **zero** non-test hits. A real site hits all of these in week one.
- The issue tracker is fully burned down: **0 open issues, 0 open PRs**. There
  is no queue. Strategy is now the bottleneck, not execution.

So: the generator is unusually capable, and untested against a real workload.
**A real site is the cheapest, highest-information moonshot available**, and
Moonshot 1 below is built entirely around that.

---

## How to read this

Twelve candidates. Each gets:

1. **What** — one paragraph.
2. **Why la-famille specifically** — not a generic SSG feature.
3. **Phases** — 3–5, each with what it delivers and an **observable
   done-criterion** (something you can run, open, or diff — not "works well").
4. **Cost** — rough agent-days.
5. **Biggest risk.**

The top 5 are ranked by `(impact × fit with la-famille's identity) / cost`.
Blue-sky proposals are labeled **BLUE-SKY**. Everything unranked is in
[Cut list](#cut-list) with a reason.

### Scorecard

| # | Moonshot | Impact | Identity fit | Cost (agent-days) | Ratio |
|---|----------|--------|--------------|-------------------|-------|
| 1 | [The Homestead Program](https://github.com/drawmeanelephant/la-famille/issues/579) | Very high | Very high | 22–30 | **1.0** ★ |
| 2 | [The Change Ledger](https://github.com/drawmeanelephant/la-famille/issues/580) | High | High | 10–14 | **0.85** ★ |
| 3 | [Ask This Site: Retrieval, Revisited](https://github.com/drawmeanelephant/la-famille/issues/581) | High | Very high | 14–20 | **0.8** ★ |
| 4 | [Corpus Packs](https://github.com/drawmeanelephant/la-famille/issues/583) | Very high | Very high | 16–24 | **0.75** ★ BLUE-SKY |
| 5 | [Vault Mode](https://github.com/drawmeanelephant/la-famille/issues/582) | High | High | 12–16 | **0.7** ★ |
| 6 | [The Studio](#6-the-studio--an-editorial-console-in-the-tui) | Med-high | Very high | 16–22 | 0.6 |
| 7 | [Time Travel](#7-time-travel--the-site-as-a-time-indexed-archive) | High | High | 12–18 | 0.55 |
| 8 | [Media Pipeline](#8-media-pipeline--assets-as-first-class-build-input) | Med-high | Medium | 10–14 | 0.5 |
| 9 | [Routines](#9-routines--the-self-improvement-loop-as-a-product) | Medium | Very high | 14–20 | 0.45 |
| 10 | [The Workshop](#10-the-workshop--a-layout-sdk-and-design-system) | Medium | Medium | 14–20 | 0.35 |
| 11 | [Verified Corpus](#11-verified-corpus--provenance-and-signatures) | Medium | High | 10–14 | 0.3 |
| 12 | [Shared Brain](#12-shared-brain--collaborative-annotations) | High | Low-med | 28–40 | 0.15 |

### The top five, in one line each

1. **The Homestead Program** — pick and ship a real site, and grow the
   generator only where that site pushes back. Highest information per agent-day
   in the entire document.
2. **The Change Ledger** — make every build answer "what changed?" It is
   load-bearing for a real site, for CI, and it is the foundation Time Travel
   sits on.
3. **Ask This Site: Retrieval, Revisited** — hybrid lexical+dense retrieval,
   link-graph expansion, and a golden-question eval harness. Turns the most
   distinctive feature from demo into measured system.
4. **Corpus Packs** — the site as a signed, versioned, diffable, subscribable
   machine-readable object. The most ambitious and the most la-famille-shaped.
5. **Vault Mode** — point la-famille at a folder of notes. Real audience,
   real demand, and `internal/stub` already does half the hard part.

---

## 1. The Homestead Program — Ship a Real Site

See [#579 Homestead Program](https://github.com/drawmeanelephant/la-famille/issues/579).

## 2. The Change Ledger — What Did This Build Actually Change?

See [#580 Change Ledger](https://github.com/drawmeanelephant/la-famille/issues/580).

## 3. Ask This Site: Retrieval, Revisited

See [#581 Ask This Site: Retrieval, Revisited](https://github.com/drawmeanelephant/la-famille/issues/581).

## 4. Corpus Packs — The Site as a Versioned Object

See [#583 Corpus Packs](https://github.com/drawmeanelephant/la-famille/issues/583).

## 5. Vault Mode — la-famille for Knowledge Bases

See [#582 Vault Mode](https://github.com/drawmeanelephant/la-famille/issues/582).

## 6. The Studio — An Editorial Console in the TUI

### What

The TUI is already a site *manager* (build, serve, watch, stats, diagnostics,
RAG export, Ask This Site). The Studio turns it into an **editorial console**:
a workspace where content changes — authored by a human or proposed by a local
model over the site's own RAG corpus — are reviewed, diffed against the live
site, accepted or rejected, and turned into a build and a PR, all without
leaving the terminal. It is the natural product surface for a repo that is
already run by agents (`AGENTS.md`, `.agents/plans/`, `la-famille pr sync`,
Jules-as-author). Concretely: a proposal inbox, a rendered side-by-side diff
viewer, an accept/reject/edit flow bound to `la-famille new` and the build
cache, and a `pr sync`-backed handoff.

### Why la-famille specifically

Every prerequisite is already in the binary:

- The **Bubbletea TUI** (`cmd/la-famille/tui.go`) with its menu, keybindings,
  diagnostics drawer, and mascot is the shell; the Studio is new screens in it,
  not a new app.
- **`la-famille pr sync`** (`internal/github`) already has a full PR policy
  engine (`EvaluatePR`, author allowlist, required label, head-prefix gates,
  check-run gating, dry-run/apply). The Studio's "ship it" button is
  essentially a reviewed, human-gated version of the policy that already
  governs this repo.
- **`ask`** provides the local model that can propose content, grounded in the
  site's own corpus with citations, running loopback-only — so a "draft a page
  about X from these notes" flow is a prompt away and never leaves the
  machine.
- The **incremental build cache** and **watcher** make accept-then-rebuild
  instant, which is what makes an in-terminal review loop feel good.
- The **checker/diagnostics** surface is what the proposal view annotates:
  "this edit fixes 1 broken link, creates 1 orphan," shown before you accept.

This is the only moonshot whose primary user is the project's own maintainers
— which is exactly why it fits: la-famille is a repo that is literally run by
agents through a documented, checked-in loop. The Studio productizes that loop.

### Phases

**Phase 1 — Proposal inbox in the TUI (5–6 agent-days).**
Represent a content proposal as a first-class object (target path, patch,
rationale, optional citations from `ask`). A "Proposals" TUI screen lists
pending proposals with a `j/k` cursor, status (proposed/accepted/rejected),
and the author (human or local model). `la-famille propose` (or the `ask` UI's
"save as page") creates one, writing a proposal file under a working
directory, not into `content/` yet.

*Done-criteria:* `la-famille propose "page about kiln drying" --notes
a,b,c` creates a listed proposal; it appears in the TUI Proposals screen with
its rationale and the three cited note links; quitting and relaunching the TUI
shows the proposal still listed (it is persisted, not in-memory).

**Phase 2 — Review and diff view (6–7 agent-days).**
In the TUI, open a proposal to see a side-by-side diff of the proposed page
against the current one (and a "new file" view for additions), with the
checker's live findings annotated onto the diff (new broken links, new orphans,
missing frontmatter). Accept writes the file into `content/`, triggers an
incremental rebuild, and updates the status; Reject records the decision with
a one-line reason; Edit opens the file in `$EDITOR` as an escape hatch.

*Done-criteria:* a proposal that adds a link to a nonexistent page shows that
broken link highlighted in the diff *before* accept; accepting it rebuilds
incrementally and the Diagnostics drawer shows the new finding; a proposal
that only rewords prose shows "0 regressions" via the Moonshot 2 ledger if
available. Reject-with-reason persists and removes it from the inbox.

**Phase 3 — Ship it (5–6 agent-days).**
On accept, offer "ship": commit the accepted change on a branch, push, open a
PR via the existing `internal/github` client, and let `pr sync`'s policy engine
gate it exactly as it gates the agents' PRs today. The human stays in the loop
(the PR is a real PR, reviewable on GitHub), but the tedious local steps
(branch, commit, push, PR) are one keystroke.

*Done-criteria:* from the TUI, a content change becomes an open PR on the
remote with the standard title/body and a `publish-check` CI result, using
the same allowlist/label policy `pr sync` enforces. The PR body links the
proposal and its citations. A dry-run mode shows the exact git/GitHub actions
without performing them.

### Cost

**16–22 agent-days** (5–6 / 6–7 / 5–6). Bubbletea UI work is the bulk; the
policy and GitHub layers are already built.

### Biggest risk

**A TUI that embeds a diff viewer and an editor is a notoriously large,
finicky surface**, and the payoff is internal (la-famille's own maintainers)
rather than user-facing. It is the most likely of the top ten to be a time
sink that produces a demo rather than a used tool. It ranks sixth for that
reason. If attempted, the honest scope is Phase 1–2 only and "shells out to
`git diff` / `$EDITOR`" rather than hand-rolling a full TUI diff widget.

---

## 7. Time Travel — The Site as a Time-Indexed Archive

### What

Time Travel makes the site's *history* a first-class, queryable, browsable
artifact. Every build writes a content-addressed snapshot of the site manifest
(Moonshot 2's manifest) plus a timestamp, so a site accumulates a timeline. On
top of that: a scrubber that shows what the site looked like on any past date,
a diff between any two dates, a "how did this page evolve" view that stitches a
page's revisions into a readable changelog, and — the la-famille twist — a
time axis in the knowledge-graph explorer and a time-filtered `ask` ("what did
I believe about X in March?").

### Why la-famille specifically

- The repo is **already full of dated, longitudinal content** — a deep
  `content/jules/` devlog with per-date entries, nightly-maintenance reports,
  and a changelog. The content type that motivates this already exists; it has
  just never been treated as an archive.
- The **build cache** (`internal/generator/cache.go`) already fingerprints
  content/templates/assets and stores per-file metadata; snapshots are that
  structure promoted from "cache invalidation" to "a retained, queryable
  history."
- The **link graph** (`graph.json`) is already a versioned deterministic
  payload; a snapshot series is a time series of exactly that payload, and the
  graph explorer is already a static, no-runtime page that can render one —
  or several — without a server.
- **`ask`** already indexes by chunk with stable IDs; giving a chunk a
  validity interval turns "what does the site say" into "what did the site say
  *then*," which no static site can answer because no static site keeps its
  own history.

The competitive framing matters: this is not "add version control" (git
exists). It is *the generator retaining and publishing its own build history as
a browsable product surface*, which is precisely the kind of thing an
authoring tool for a knowledge graph can do and a document store cannot.

### Phases

**Phase 1 — Snapshot retention (4–5 agent-days).**
Persist a dated, content-addressed snapshot of the site manifest (and
optionally the graph JSON) per build into a retained `.la-famille/snapshots/`
(or beside the build cache), with a retention policy (keep daily/weekly
forever, thin out old granularities). `la-famille history` lists snapshots.

*Done-criteria:* building the same site across N simulated days produces N
snapshots, `la-famille history` lists them newest-first, an old snapshot file
can be opened and re-verified against its hash, and the retention policy
demonstrably thins a 90-day synthetic series to a bounded set without losing
the daily endpoint.

**Phase 2 — Time-diff and the scrubber (4–6 agent-days).**
`la-famille diff --at <date> --against <date>` (reusing the Moonshot 2 diff
engine over snapshots) and a static, no-JS-framework **timeline scrubber**
page: pick a date, see the site's page list, added/removed/changed pages at
that point, and click into a past version of a page rendered from that
snapshot.

*Done-criteria:* the scrubber, opened from `file://` like the graph explorer,
lets you drag/select two dates and shows the exact added/removed/changed page
lists between them, and renders a past page's content at the earlier date. It
requires no server (parity with `/graph/index.html`).

**Phase 3 — Graph time axis and time-filtered ask (4–7 agent-days).**
Add a time axis to the graph explorer: a date slider that shows which edges
and nodes existed at a chosen snapshot. Extend the RAG chunks with validity
intervals from the snapshot series, and add `la-famille ask --as-of <date>` to
restrict retrieval to chunks valid at that date.

*Done-criteria:* in the graph explorer, moving the time slider changes node
and edge visibility to match the selected snapshot's graph. `la-famille ask
--as-of` on a vault/site with a revised page returns the *earlier* wording with
a citation to the old snapshot, and today's date returns the current wording —
the same question, two eras, two cited answers.

### Cost

**12–18 agent-days** (4–5 / 4–6 / 4–7). Depends on the Change Ledger
(Moonshot 2) for its diff engine and snapshot substrate — **do not build this
before Moonshot 2.**

### Biggest risk

**Storage and retention rot.** A site built thousands of times accumulates
snapshots; if retention is wrong this becomes a disk problem, and if the diff
engine isn't already solid (Moonshot 2) the whole surface inherits its
correctness risk. The hard dependency on Moonshot 2 is the mitigation — this
moonshot is deliberately sequenced after it, and should not be started first.

---

## 8. Media Pipeline — Assets as First-Class Build Input

### What

Today `assets/` is copied verbatim. The Media Pipeline makes media a real
build input: an image asset is fingerprinted, probed, and emitted in
responsive derivatives with intrinsic dimensions, lazy-loading defaults, and
auto-generated OpenGraph/social cards; a content page referencing an image via
frontmatter `image:` or a new `gallery:` block gets a real, responsive, correct
figure. This is what a real, image-bearing site (Moonshot 1) needs and what
the current asset copy cannot give it.

### Why la-famille specifically

The alignment is real but moderate, and the doc should say so honestly:

- The **frontmatter `image:` field** already exists on every `Page` and drives
  OpenGraph tags, and `check --asset-health` already audits asset references.
  A media pipeline slots into both.
- The **Octoburger/editorial theme** story is a *visual* identity — a flagship
  theme and 20 showcase layouts — so image handling is core to the product's
  felt quality, not an afterthought.
- The **asset health checker and the build cache fingerprint** already treat
  assets as a tracked input class.
- stdlib `image/jpeg`/`image/png`/`image/gif` cover decode + resize + encode
  for the common formats with **no new dependency** (respecting AGENTS.md's
  minimal-deps rule); WebP/AVIF would be out of scope or optional.

Where it is *not* la-famille-specific: every modern SSG or image pipeline does
responsive images. This is a **table-stakes** moonshot, not a
differentiator — which is why it ranks eighth, and why the honest framing is
"do it because the real site needs it," not "do it as a bet."

### Phases

**Phase 1 — Probe and index (3–4 agent-days).**
At build time, probe every image asset (dimensions, format, EXIF orientation,
dominant color) into a `public/assets/manifest.json`; `check` warns on
oversized or unprobed images; templates can query intrinsic width/height to
prevent layout shift.

*Done-criteria:* the manifest for a fixture with PNG/JPEG assets records
correct dimensions and dominant color; a page rendering an image emits
`width`/`height` attributes from the probe (verifiable in output HTML);
`check` reports a warning for a deliberately oversized image.

**Phase 2 — Responsive derivatives (4–5 agent-days).**
Emit width-constrained, content-hashed derivatives of each image (PNG/JPEG via
stdlib) and wire a `<picture>`/srcset helper into templates, with lazy-loading
defaults.

*Done-criteria:* a single source image emits ≥3 width variants with stable
content-hashed names; a template using the picture helper renders correct
`srcset`/`sizes`; changing the source image changes the derivative hash and
invalidates the cache for exactly that image (no full rebuild).

**Phase 3 — Auto social/OG cards (3–5 agent-days).**
Compose a default OpenGraph card (site name, page title, the page's `image:` or
a site default) as a generated image at build time, so every page has a real
social preview without hand-designing one.

*Done-criteria:* every rendered page's `og:image` resolves to a generated
card file that exists in `public/`; `publish-check` and the subpath/base-path
logic (already hardened for `og:url`) correctly rewrite the card URL under a
`siteurl` subpath; a page with no `image:` still gets a valid card.

### Cost

**10–14 agent-days** (3–4 / 4–5 / 3–5). No new runtime dependency required for
the PNG/JPEG path; WebP/AVIF would add a dependency and should be deferred per
AGENTS.md.

### Biggest risk

**Scope creep into dependency bloat.** Doing WebP/AVIF/HEIC well essentially
requires a C-backed or large pure-Go codec, which collides directly with
AGENTS.md's "keep external dependencies strictly minimal." The discipline is to
ship the stdlib PNG/JPEG path and explicitly defer modern codecs rather than
quietly add a large dependency. Secondary risk: this is table-stakes, so doing
it well yields no differentiation — it should be justified by the real site
needing it, not pursued as a moat.

---

## 9. Routines — The Self-Improvement Loop as a Product

### What

This repo runs on recurring "routines" — nightly maintenance, docs-reality
passes, frontmatter normalization, stub-closing, refactor-one-seam, cat-facts
— executed by agents and filed as dated reports under `content/jules/reports/`.
Routines turns that ad-hoc operating model into a **first-class, shipped
feature**: declarative routine specs (YAML) describing a check-and-act task
(what to scan, what "good" means, what change to propose), a sandboxed runner
that executes a routine against a site, and a quality-gated PR producer that
only opens a PR when the routine's own check output measurably *improves* (or
strictly holds). It is the project eating its own dog food, formalized.

### Why la-famille specifically

- The routines are **literally this repo's development history** — 58 task
  plans and a deep `content/jules/` routine library. Routines formalizes an
  observed, proven process rather than inventing a new one.
- The **checker** (`internal/checker`) already produces the structured
  findings (orphans, broken links, missing metadata, asset health) that
  "improvement" would be measured against; a routine's gate is a comparison
  of checker output over time.
- The **TUI** is the natural place to review a routine's proposed change, and
  the **Change Ledger** (Moonshot 2) is the natural "did this improve
  anything visible" oracle.
- The **PR automation** (`pr sync`, `internal/github`) already enforces the
  agent-authored-PR policy, so a routine is just another producer into a
  governed pipeline.

The fit is excellent on identity and weak on product value: this is a tool for
people running agent-driven content repos, which is a real but small audience —
and la-famille is largely that one repo today.

### Phases

**Phase 1 — Routine spec + runner (5–6 agent-days).**
A `routines/` YAML schema (id, what to scan, the check, the proposed fix, the
gate metric) and a `la-famille routine run <id>` that executes a routine in a
sandboxed copy, emits a proposed change plus a before/after check report, and
never touches the working tree without `--apply`.

*Done-criteria:* a shipped `stub-closer` routine, run against a fixture site
with known stubs, produces a proposed change and a before/after checker report
where the orphan count is lower, in a sandbox — and with the flag off, the
working tree is byte-identical afterward.

**Phase 2 — Quality gate (4–5 agent-days).**
A routine may only propose a PR if its gate metric (a checker count, a ledger
diff class) improves or holds; regressions are rejected automatically. Wire the
gate so a routine can never auto-merge, only ever propose.

*Done-criteria:* a routine that would *introduce* a broken link is refused by
the gate with a reason, on a fixture engineered to trigger it; a routine that
strictly improves is allowed to propose. The gate is a unit-testable pure
function.

**Phase 3 — Routine library + PR loop (5–7 agent-days).**
Ship a starter routine library (frontmatter normalizer, docs-reality check,
link-health pass) and the loop: run routine → gated proposal → PR via
`internal/github` → `pr sync` policy. A TUI screen lists routine history.

*Done-criteria:* running a shipped routine end-to-end on a fixture produces a
gated PR on a test repo that passes `pr sync`'s policy checks; the TUI routine
history shows each run's before/after gate numbers; a routine that regresses
never produces a PR across a 10-run soak on the fixture.

### Cost

**14–20 agent-days** (5–6 / 4–5 / 5–7). Roughly half is productizing
processes the repo already performs manually, so the "code" is cheaper than it
looks; the cost is in making the gate trustworthy enough that PRs are welcome.

### Biggest risk

**This can become an AI-slop factory** — a machine that manufactures low-value
"improvements" that cost human review time and pollute the history. It is also
the moonshot most likely to be seen as "build a tool for ourselves" rather
than "build for users." The quality gate is the only real defense, and it must
be genuinely strict (a routine that can't prove improvement doesn't ship a
PR). If the gate is ever loosened, this moonshot should be killed rather than
tuned.

---

## 10. The Workshop — A Layout SDK and Design System

### What

There are 4 active layouts and 20 deprecated ones, plus ad-hoc CSS
(`theme-foundations.css`, `theme.css`, `layout-*.css`). The Workshop turns
templating into a real SDK: a **component and slot system** (layouts declare
composable regions; pages fill named slots via frontmatter), a **design-token
layer** (one source of truth for color/spacing/type feeding every theme), a
**live theme preview** in the TUI or browser, and an **accessibility/design
contract** enforced by regression tests so a new theme cannot ship broken.
The output is a real design system with a component gallery, not a folder of
near-duplicate HTML files.

### Why la-famille specifically

Partially, and honestly:

- The **Octoburger flagship theme** and the **template showcase gallery**
  (`content/showcase/`, one page per layout) are already a design-identity
  product; a component SDK and token layer is the natural maturation of that.
- The **TUI** already does light-reload and previews; a theme previewer is a
  natural screen.
- The repo already has **template contract/accessibility regression coverage**
  and a self-contained-theme push (the recent "deprecate CDN themes, alias
  `theme:dark`" work) — there is real momentum to consolidate here.

Where it is not la-famille-specific: a component system and design tokens are
a frontend-platform concern, not an SSG differentiator. The 20 deprecated
layouts are arguably a maintenance liability, not a platform to grow. This
ranks tenth because it is the least connected to the corpus/graph/RAG identity
that actually distinguishes la-famille — it makes the *themes* better, not
the *generator* more la-famille.

### Phases

**Phase 1 — Token layer (4–5 agent-days).**
Extract all themes onto a single token source (color ramps, type scale,
spacing, radii) with a documented token contract; every layout consumes tokens
instead of hardcoded values.

*Done-criteria:* a token-only redefinition (e.g. change the accent ramp)
visibly re-themes every active layout with no per-layout edits; a contract
test fails if any active layout hardcodes a color that has a token.

**Phase 2 — Slots and components (5–6 agent-days).**
Declarative named slots in layouts, filled from frontmatter/shortcodes;
a small library of reusable components (card, figure, callout, table) that
layouts and pages both use.

*Done-criteria:* a page frontmatter fills a named slot in a slot-aware layout
and renders it; a component used in two layouts produces identical markup
(asserted in a golden test); existing layouts with no slots are unaffected
(byte-identical output for the current fixtures).

**Phase 3 — Preview + gallery (5–7 agent-days).**
A live theme preview (TUI or browser) rendering any layout/token combo on
demand, and a regenerated showcase gallery page that is a real component
gallery rather than per-layout screenshots.

*Done-criteria:* a preview screen renders a chosen layout/token combo without
a full build; the showcase gallery reflects the token system and every
component; the accessibility contract test runs across every active layout
and is a CI gate.

### Cost

**14–20 agent-days** (4–5 / 5–6 / 5–7).

### Biggest risk

**This is a maintenance sink masquerading as a platform.** la-famille's 4
active layouts are a manageable, deliberate set; a component SDK invites
abstraction that a small generator does not need and that will drift from the
templates it is supposed to serve. If the real site (Moonshot 1) does not
demand slots or tokens, this should stay cut. Treat it as a *response to a
real need*, never as proactive polish — the roadmap already defers large
theme redesigns, and this doc should not quietly overturn that.

---

## 11. Verified Corpus — Provenance and Signatures

### What

`ask` already refuses to cite anything it cannot ground in the corpus
(`internal/retrieval`'s `Citations` verifier drops hallucinated keys and falls
back to a no-answer). Verified Corpus extends that honesty guarantee from the
*runtime* to the *artifact*: every RAG archive and every Corpus Pack (Moonshot
4) carries per-chunk content hashes and a manifest-level **signature** over
the manifest, so a consumer can prove (a) the corpus has not been tampered
with since publication and (b) any particular cited chunk is exactly the text
the publisher signed. `ask` gains a `--verify` mode that refuses to answer
from an unverified corpus, and the UI shows a verified/unverified badge per
source.

### Why la-famille specifically

- The **citation verification machinery is already the centerpiece of `ask`**
  — la-famille has already invested in "don't make things up," and this moonshot
  extends that exact instinct one layer up, from runtime answers to the shipped
  artifact.
- The **`ragfmt` escaped-envelope format** and the **deterministic, cache-
  fingerprinted build** make per-chunk hashing and deterministic manifests
  natural; a build that is already reproducible is the precondition for
  signing, and la-famille's transactional/deterministic build largely
  satisfies it.
- A **local-first, citation-grounded assistant** has a real threat model here
  that a hosted chatbot does not: if you run `ask` over a corpus you did not
  author (a friend's site, a downloaded pack), you currently have no way to
  verify integrity. Verified Corpus is the answer to "how much do I trust
  this?" for a privacy tool.

The limitation to state plainly: signatures need a key-distribution story
(who signs, what the trust root is), and la-famille is a solo-ish project with
no PKI. So the honest version is **integrity hashes + optional
detached-signature verification with a user-supplied public key** (e.g.
minisign/age-style via stdlib crypto/ed25519), not a full trust ecosystem.

### Phases

**Phase 1 — Deterministic chunk hashing (3–4 agent-days).**
Give every RAG chunk a stable content hash derived from its exact archived
text; write a per-archive chunk-hash index. `la-famille rag verify` checks an
archive's chunks against the index and reports drift.

*Done-criteria:* on a fixture archive, `rag verify` passes clean; editing one
byte of one chunk in the archive makes `rag verify` fail naming that chunk and
its expected/actual hash; chunk hashes are stable across two builds of
unchanged content.

**Phase 2 — Signed manifest (4–5 agent-days).**
A signed manifest over the archive/pack (ed25519 via stdlib `crypto/ed25519`),
`la-famille sign` to produce a detached signature and `la-famille verify
--pubkey` to check it. Bundled test keypair for fixtures; real keys are
user-managed.

*Done-criteria:* a signed archive verifies against its public key; a
tampered archive fails signature verification (not just hash verification);
a wrong public key fails cleanly; the fixture keypair is test-only and clearly
labeled.

**Phase 3 — `ask --verify` (3–5 agent-days).**
`ask` verifies the corpus at startup (or refuses with a clear error) and
threads a per-source verified/unverified badge into the existing
`SourceCard` UI. An unverified corpus still answers but is clearly marked, or
can be hard-refused with a flag.

*Done-criteria:* `la-famille ask` over a tampered archive either refuses (strict
mode) or answers with every source marked unverified (lenient mode), both
demonstrable on a fixture; a clean archive shows all sources verified; the
per-source badge reuses the existing `SourceCard` rendering (visible in the
UI, asserted in a UI smoke test).

### Cost

**10–14 agent-days** (3–4 / 4–5 / 3–5). Mostly stdlib `crypto`; the cost is
designing a trust story that doesn't overclaim.

### Biggest risk

**This can overclaim security it doesn't deliver.** Signing proves the
archive hasn't changed since signing — it does not prove the *content* is
true, and a solo project has no authority to sign anyone else's corpus. If
shipped with sloppy language ("verified" implying "trustworthy"), it is worse
than nothing. The mitigation is strict, narrow wording (integrity, not
truth) and resisting the urge to build a key-distribution system la-famille
has no standing to run. Realistically this is a nice-to-have that matters
most once Corpus Packs (Moonshot 4) exist and people consume archives they
did not build.

---

## 12. Shared Brain — Collaborative Annotations

**BLUE-SKY**

### What

Shared Brain layers **collaborative annotations** onto a static la-famille
site: readers highlight passages, leave notes, and ask questions in place;
those annotations are stored as static, mergeable data and rendered back onto
the pages *and* fed into `ask` so the assistant can answer from the
conversation, not just the source. The site stops being a read-only document
and becomes a shared, evolving knowledge surface where the reading itself is
part of the corpus.

### Why la-famille specifically

- The **citation-grounded `ask`** and the **stable chunk IDs** are exactly the
  substrate a highlight/note/ask-in-place layer needs — annotations anchor to
  the same stable chunk/page IDs the assistant already cites, so an annotation
  and an AI answer reference the same addressable unit.
- The **taxonomy and graph** could, in the limit, index annotations as
  first-class nodes, making the graph explorer a map of the *conversation*
  about the site, not just the site.
- The **local-first, loopback-only privacy stance** gives Shared Brain a real
  differentiator: a self-hosted, exportable version of collaborative reading
  that does not require a SaaS and does not phone home.

Why it is still last: it requires solving **multi-writer synchronization and
conflict resolution over static files** — the genuinely hard, generic problem
that has nothing to do with what makes la-famille la-famille. It also
contradicts the project's privacy posture (it is inherently multi-user and
networked), and it demands a backend/identity model the project has no
interest in running.

### Phases

**Phase 1 — Local annotation model + rendering (6–8 agent-days).**
A static annotation data format (CRDT-friendly: per-annotation immutable
records with stable IDs, anchored to chunk/page IDs), a `la-famille annotate`
local flow, and rendering of annotations back onto pages (highlight, margin
note, inline question) as pure static output.

*Done-criteria:* annotating a fixture page (a highlight and a note) persists
as static data, renders on rebuild, anchors to the correct stable chunk ID
(verified against the RAG archive's chunk ID), and the site works fully
offline/loopback.

**Phase 2 — Sync and merge (10–14 agent-days).**
A merge/conflict story for multi-writer static data (append-only CRDT merge
over the immutable records), plus a minimal transport (a folder of JSON files
merged via a shared location — git, a synced folder — with no server).

*Done-criteria:* two independently-authored annotation sets merge to a
superset with no data loss on a fixture; the merge is commutative and
idempotent (property-tested); a duplicated edit (same ID) is deduped, not
duplicated.

**Phase 3 — Annotations in the graph and in ask (6–8 agent-days).**
Index annotations into the graph (optional annotation nodes) and extend `ask`
to answer from annotations ("what did readers ask about X here?").

*Done-criteria:* an annotation appears in the graph explorer (or an annotation
view) anchored to its page; `la-famille ask` returns a question-annotation as a
cited source when relevant, alongside page sources.

**Phase 4 — A usable sync/share UX (6–10 agent-days).**
A real share/read/answer flow (even minimal): a shareable static bundle, a way
to submit an annotation, and a TUI or browser view of the merged result.

*Done-criteria:* a second person can annotate a published site and merge
their annotations back without a server, end-to-end, on two machines.

### Cost

**28–40 agent-days** (6–8 / 10–14 / 6–8 / 6–10). By far the most expensive
proposal here, and the least connected to the identity.

### Biggest risk

**It solves the wrong hard problem.** The hard part is distributed
synchronization, which is a solved-at-enormous-cost, non-la-famille problem
(CRDT libraries, identity, transport). la-famille would spend its largest
budget in the least distinctive area to produce a feature that damages the
project's "local-first, private, minimal" identity. It is listed to show the
idea was considered, and it is cut for exactly these reasons.

---

## Ranking the top five

Ranked by `(impact × fit with la-famille's identity) / cost`.

### #1 — The Homestead Program

The highest information per agent-day in this document, and the only proposal
that resolves the project's stated honest anchor. Every other moonshot is
speculative about what a real site needs; Homestead *finds out*. It also
derisks the rest: the Change Ledger, the Media Pipeline, and the Workshop only
earn their cost if a real site actually wants them, and Homestead is what
tells you. A real, live site is also the single most persuasive artifact for
adoption ("here is a site la-famille built" beats any feature list), and it
exercises the differentiating subsystems — graph, RAG, ask, taxonomy, cache —
at a scale and messiness the 30-page doc site never will. Fit is maximal
because it is the project's own stated next step. Cost (22–30 days) is the
highest in the top five, but the payoff compounds across every other proposal.

### #2 — The Change Ledger

Impact is high and unusually *concrete*: once it exists, every build says what
changed, CI can gate on regressions, and the TUI, the real site, and CI all
consume it. Fit is high because it is nearly free — the manifest is a
projection of data la-famille already computes (graph, metadata, search,
checker) and then discards, so it is a *combinatorial* win on shipped
primitives. Cost is low (10–14 days) for that impact, which is why it beats
more ambitious proposals. It also earns its keep inside the Homestead ladder
(regression gate for the real site) and is the prerequisite for Time Travel, so
it is the best "unblocks other things" investment on the board. It ranks second
only because it is enabling infrastructure rather than a visible product.

### #3 — Ask This Site: Retrieval, Revisited

Impact is high and the fit is maximal — this is la-famille's single most
distinctive feature, and retrieval is the one part still at demo-grade. The
"why la-famille" case is the strongest in the document: the design seam for
embeddings is already documented, the link graph needed for graph-aware
retrieval is already emitted at build time, and the citation verifier (the hard
part most hybrid systems get wrong) is already solved. It produces a
*measurable* improvement (an eval harness with a recorded baseline), which is
the kind of proof the project currently lacks everywhere else. It ranks third
because it is the most expensive of the top-five "real" features (14–20 days)
and its headline payoff — better recall — depends on local model quality that
is not fully in la-famille's control.

### #4 — Corpus Packs

The highest-ceiling moonshot and the most la-famille-shaped idea here: it turns
the site into a signed, versioned, diffable, subscribable object, composed
almost entirely from subsystems la-famille has already built, with `ask` as an
immediate, already-shipped consumer. It is the proposal that most directly
extends the "site as machine-readable knowledge corpus" identity into its
natural endpoint. It ranks fourth only because of cost (16–24 days) and, more
honestly, because the *market* for subscribing to someone's site corpus is
unproven — it is the most speculative of the top five despite the highest
ceiling. Its quality depends on Change Ledger (#2) landing first, and its
value depends on a real site existing (#1) to be worth subscribing to.

### #5 — Vault Mode

High impact (a real, underserved audience already keeps their notes in this
format) and a high fit score for an underrated reason: `internal/stub` already
implements Obsidian's signature "unresolved link creates a note" behavior, and
the graph explorer is already the correct navigation for a vault. It is also
the cheapest substantial proposal (12–16 days). It ranks fifth, not higher,
because it is the one top-five bet whose *audience* is adjacent to, rather
than an expression of, the current site-generator identity — it risks reading
as a pivot toward "notes tool," even though structurally it is the same
pipeline pointed at a different content root. That risk is manageable and
Phase 1 is cheap enough to test quickly.

### Why this order

The first three are all "make what exists honest and measurable" — the
Homestead finds the real requirements, the Change Ledger makes change
observable, and Retrieval makes the flagship feature measured. That is a
deliberate bet: la-famille's problem is not a shortage of ideas (this document
is proof) but a shortage of *evidence* about which idea matters. The last two
(Corpus Packs, Vault Mode) are the biggest bets on new surface area, and they
are placed where a cheaper, more grounded foundation can catch them if they
miss.

---

## Cut list

Everything below is deliberately **not** in the top five. Each was considered
on the same criteria and cut for the stated reason.

- **The Studio (editorial TUI)** — *cut from top five, kept warm.* Excellent
  identity fit (agent-forward repo, existing PR-policy engine) but the payoff
  is internal, the diff-viewer/editor TUI surface is a known time sink, and it
  is the most likely to produce an impressive demo that nobody uses daily. If
  attempted, cap it at Phases 1–2 and shell out to `git diff`/`$EDITOR`.
- **Time Travel** — *cut as a standalone, folded into the roadmap.* Strong and
  on-identity, but it is a strict superset of the Change Ledger's diff/snapshot
  engine with extra cost. It should be re-evaluated *after* the Change Ledger
  lands and proves the diff substrate, not started independently.
- **Media Pipeline** — *cut as a bet, kept as table-stakes.* Real sites need
  it, so the Homestead ladder may pull it in as a task, but it is not a
  differentiator and the modern-codec temptation collides with the
  minimal-dependency rule. Do it when the real site demands it; do not lead
  with it.
- **Routines** — *cut on values/identity grounds.* Codifies this repo's own
  maintenance loop, but the failure mode (an AI-slop PR factory) is serious
  enough that it should not be prioritized without a demonstrably strict
  quality gate and a human who wants it. Revisit only if a routine's output is
  welcomed rather than merely tolerated.
- **The Workshop** — *cut.* Least connected to the corpus/graph/RAG identity;
  a component SDK is a frontend-platform concern and a maintenance risk for a
  deliberately small set of layouts. The roadmap already defers large theme
  redesigns; this should not overturn that without a real site demanding
  slots/tokens.
- **Verified Corpus** — *cut as premature.* Correct instinct (extend `ask`'s
  existing honesty guarantee to the artifact) but it only becomes valuable once
  people consume archives/packs they did not build — i.e. after Corpus Packs
  (#4). Do it as a follow-on to #4, not before.
- **Shared Brain** — *cut hard.* The hard problem is distributed
  synchronization, which is generic, enormous, and orthogonal to what makes
  la-famille la-famille — and it undercuts the local-first, private, minimal
  identity. Listed for completeness; the recommendation is to not revisit it.

---

## How these fit together (a possible order)

Not a commitment — just the dependency structure, if someone wanted to pull
this thread:

1. **The Homestead Program** (find out what is real) — anchors everything.
2. **The Change Ledger** (make change observable; gate the real site) —
   unblocks Time Travel and Routines.
3. **Ask This Site: Retrieval, Revisited** (make the flagship feature measured)
   — independent of 1–2, can run in parallel.
4. **Corpus Packs** (the big bet) — depends on 1 (something to pack) and 2
   (diff/snapshot primitives).
5. **Vault Mode** (real audience) — depends on 1 for a demonstration site, but
   is otherwise self-contained and cheap enough to test early.

**The one-sentence version:** ship a real site, make every build explain
itself, measure the assistant, then bet on the corpus — and let the real site
tell you which bets were worth it.

---

## Adversarial review: 2026-09-29

**The assignment here is the attack, not another list.** The original run
considered **12 candidates and ranked 5**. This review examines all twelve,
retains their names, and asks whether the original top five deserve resources.
Proposed commands remain proposals. No subsystem is implemented by this PR.

### Verdict

**The direction survives; the certainty, estimates, and claimed leverage do
not.** Homestead still wins as a decision about what to do next, but its
three-site bake-off is the wrong first move. The Ledger and retrieval eval
are useful experiments, not yet validated moonshots. Packs and Vault have
plausible audiences, but their write-ups borrow capabilities the code does
not have and omit the costliest boundary work. Nothing below justifies five
parallel implementation epics.

The worst vapor is not a far-fetched idea. It is the repeated argument that
“the hard part already exists” when the named package solves a materially
different problem. The code is useful raw material, not a prepaid future.

### What this review checked

README, the full original catalogue, `content/docs/`, the roadmap and
aspirations, the actual `cmd/la-famille` and `internal/` layout, relevant
implementation and regression tests, deployment/PR workflows, and current
GitHub issues. Code references below are relative to the repository root.

The project's own [Pages URL](https://drawmeanelephant.github.io/la-famille/)
returned HTTP 200 during this review. Open issues
[#570](https://github.com/drawmeanelephant/la-famille/issues/570),
[#571](https://github.com/drawmeanelephant/la-famille/issues/571), and
[#572](https://github.com/drawmeanelephant/la-famille/issues/572) report a real
bilingual site's RAG path-resolution failure, dropped CJK taxonomy, and unused
default-theme assets. These are issue-reported observations; this review has
not reproduced or root-caused those three bugs. The reports do not establish
an independently verified production URL for that site.

`go test ./...` and `go vet ./...` pass on the reviewed checkout. This verifies
the current implementation, **not** the original feasibility or cost claims.

### First, remove the false premises

| Original claim | What contact with the code/evidence actually establishes |
| --- | --- |
| “0 open issues” / “no queue” | Stale as of this review: #570–#572 are concrete workload feedback. A momentarily empty tracker never establishes that strategy, rather than execution, is the bottleneck. |
| “Never built a site about anything else” | Too strong: the bilingual-site reports are counterevidence. A repository survey cannot prove a negative about external users. The honest unknown is independent production use and ongoing usefulness. |
| “Fixed 17-field struct” | `internal/content/metadata.go` defines **18** fields in `FileMeta`; the original document even lists eighteen. Missing pagination/series/translation support is real, but “every real site hits all of these in week one” is an invented requirement. |
| “Incremental” cache as per-page reuse / retained world model | `internal/generator/cache.go` holds one fingerprint, generated-file hashes, warnings, counts, and health. `Build` skips the whole build on a hit and runs the build pipeline on a miss. It is not a per-page dependency index, old source store, or semantic revision history. |
| “Citation grounding already solved” | `internal/retrieval/citation.go:Citations.Verify` checks numeric key membership. `internal/ask/server.go:Answer` accepts prose with a valid key; neither checks that the cited passage supports the assertion. Valid citation identity is not entailment or truth. |
| “Stable chunk IDs” as anchors through history | `chunker.go:chunkID` uses page path, heading text, and position. Same-input determinism is useful; rename a page or insert an earlier section and identities can change. An annotation, delta, or historic answer needs a separate identity policy. |
| `render: false` as a way to publish a private subset | **Unsafe.** `content/docs/raw.md` and generator behavior say raw Markdown is copied into output. Graph and metadata retain it; RAG content export is not a private-public filter. Retrieval excluding raw pages does not make the published bytes private. |
| Explorer “just open the file” as a general portability guarantee | `internal/graphexplorer` writes `graph/data.json`; `assets/graph/explorer.js` loads it with `fetch`. Static HTTP hosting needs no application backend, but browser `file://` restrictions still matter. Test browser behavior or use loopback serving; do not inherit the prose claim. |

These corrections matter more than the scorecard. Privacy and citation claims
are promises to readers; they cannot be hand-waved as implementation detail.

### The cost model is not honest enough to fund

“Dishonest” here describes the accounting, not the prior author's intent.
The document claims to include the boring 40%, but never decomposes that work,
assigns an owner, measures throughput, or includes maintaining what ships.
Multiplying subjective “High” labels and dividing by days does not produce
ratios such as **0.85**. Those numbers are decoration, not analysis.

The most consequential omissions:

- **Homestead, 22–30 days:** three sites with 40+ real pages each, assets,
  taxonomy, deployment, and reader evidence are priced as 6–8 first-phase
  days. Content authoring is excluded later while “full content” is mandatory
  earlier. The largest input is simultaneously required and off-budget.
  Waiting for readers, choosing an owner, and retaining editorial attention
  are calendar/human costs, not interchangeable agent-days.
- **Ledger, 10–14:** a basic manifest diff might fit a small job. Git-ref
  reconstruction, reliable identity, semantic changes, regression policy,
  source-to-output consistency, CI baselines, and an accessible TUI are not
  “mostly plumbing.” `publisher.Manifest` already provides a file list, but
  no existing package supplies the whole proposed comparison model.
- **Retrieval, 14–20:** human question labeling, held-out evaluation,
  unsupported-claim assessment, multilingual segmentation, machine/model
  variability, and embedding cache/version behavior are missing. Calling a
  fake provider cannot measure generative answer quality. Embeddings need
  a new capability; `llm.Provider` exposes `Complete`, not `Embed`.
- **Packs, 16–24:** the estimate is for an archive, a new compatibility
  contract, a member/chunk diff format, a delta application system, multiple
  transports, an updater, an `ask` reader, and hostile-input handling. The
  exporter currently emits monolithic bundle files: a one-page edit changes
  `rag-content.md`, not an independently addressable per-page member. The
  promised tiny delta depends on a payload redesign not priced in phase one.
- **Vault, 12–16:** the cheap version is a restricted wiki-link importer.
  Duplicate note names, relative resolution, heading/block references,
  aliases, embeds, attachments, migration diagnostics, and safe selective
  publication are a different bill. “Already does half the hard part” is not
  a justified effort estimate; generated stubs are not editable source notes.
- **Studio / Routines / Shared Brain:** the UI, repository mutations,
  metric-gaming defenses, synchronization, durable anchors, contributor
  identity, and untrusted-input boundaries are product work. A happy-path
  demonstration is not the supported lifecycle the proposals price.

**Do not replace these ranges with equally invented larger ranges.** Before
funding a full phase, name one bounded artifact, its compatibility/security
work, the human review input, and what is explicitly deferred. Record actual
elapsed implementation and review effort from that slice; re-estimate the
rest afterward. Full delivery and a phase-one proof have different budgets.

### Top-five attack: what survives and what must be cut

#### Homestead: keep #1; kill the bake-off

Its core is the most honest proposal: a real site makes requirements stop
being guesses. Its first phase is the least honest expression of that core.
Three simultaneous destinations, 120+ real pages, and reader-based selection
turn “ship one thing” into a miniature content studio. Page count exercises
scale, not demand. “Publish all three” also assumes destinations and ownership
that have not been chosen.

The requirement to remove **every** workaround before finishing phase three
is another trap: workarounds can be reasonable, and a requirement like i18n
might be disproportionately costly. The listed data/pagination/redirect
features preselect the solution before the promised discovery happens.

**Repair, using the same bet:**

1. **Publish or verify one independent site now**, preferably the bilingual
   workload if its owner and source are available; otherwise pick one narrow
   corpus the maintainer actually wants to maintain. Use the release binary
   and existing `check`, `build`, `rag`, and `publish-check`. Ship useful
   content at an actual HTTPS URL, recording source revision, compiler
   version, nonempty intended archive coverage, and known workarounds.
   A six-page useful release can beat three forty-page performances.
2. **Observe use and close only encountered gaps.** Record a reader completing
   a task, a real content correction, and reproducible generator failures.
   Fix high-impact friction with same-package tests; no requirement to erase
   every workaround, and no speculative helper layer before a workload needs it.
3. **Demonstrate maintenance by someone else.** A second contributor makes
   a correction through the released-binary path; the owner keeps the URL
   current through two review cycles. A dated evidence log is the artifact,
   not a promise of perpetual scheduled content generation.

That is the original ladder to a shipped site, made executable. There is no
claim this docs PR has shipped the independent site. **Stop** if no person
will own the corpus or no reader gains anything from it. The project already
publishes itself; another self-referential showcase does not clear this gate.

**What it obsoletes:** speculative backlog generation and showcase-only
credibility. Shipping a site alone is not a new subsystem or a moonshot under
the document's own definition; sustained agent-maintained useful publishing
is the larger bet. Keep the first step deliberately small.

#### Change Ledger: keep #2 as an experiment, not cheap inevitability

The strongest leverage claim is partially right: page URLs, metadata, graph
adjacency, categorized findings, and actual output files are available. A
reviewer can benefit from a before/after report. But the cache does not retain
the prior content model, and `search.json` carries snippets/headings, not full
page text. Existing outputs cannot magically explain every semantic edit.

The proposed phase-one manifest records title/date/taxonomy/links, **not page
body content hashes**. Editing a paragraph can leave it unchanged, despite
the phase-two “modified page” claim. CSS/template-only output changes also
need a byte-level signal. A graph diff is not a render diff.

The “broken link and new orphan” example is wrong as a general gate: a missing
target is an unresolved edge/stub; the newly added source is an orphan only
if nothing links **to it**. One condition does not imply the other.
`publisher.Check` lists stubs separately from artifact errors. A stub can make
an HTML target resolve while `checker.Validate` still reports missing source;
choose and report both semantics rather than conflating them. Neither checker
output nor an artifact alone is the entire world model.

**Repair:** (1) compare two explicit source/output snapshots, with source-body
and output-byte hashes plus current metadata; (2) report known page/link/stub
and content changes, leaving ambiguous renames and unknowns explicit; (3)
add opt-in regression policy with intentional-change acknowledgments, then a
TUI view only if reviewers use the CLI report. Archive/ref reconstruction is
separate work. Each rung yields a usable report without the next.

**Gate / kill:** can the report expose a real missed consequence and make ten
actual agent patches easier to review? If not, stop at the small diff tool.
Do not fail publication merely because a taxonomy term was deliberately
retired or a legitimately isolated page is new. The current table elevates
policy preference to regression fact.

**What it obsoletes:** reviewers reconstructing reader impact from scattered
artifacts—not Git diff. Alone it is a substantial feature; its moonshot ceiling
is evidence-based human oversight of agent-maintained publications, which the
original ranking barely names.

#### Retrieval, Revisited: keep #3; attack the alleged solved half

The golden-question harness is the best independently shippable first phase
in the original feature proposals. The chosen headline—better retrieval—is
not sufficient. `Citations.Verify` allows a model to attach a valid `[1]` to
an unsupported assertion. “Refuses to make things up” is an instruction and
fallback policy, not a property established by key checking.

There is a more immediate bottleneck: `BuildAnswerPrompt` in
`internal/retrieval/prompt.go` sends **160-rune excerpts**, not complete
selected chunk text. Increasing recall will not help if the supporting fact
is beyond the excerpt. Its context budget does not make omitted text appear.
Audit the evidence the model actually sees before buying a new ranker.

Additional underpriced seams:

- The graph is untyped `[source, target]` link pairs, and `LinkTransformer`
  currently records internal `.md` link destinations. It is not a complete
  navigational graph or a semantic evidence graph. Walking A→C→B does not
  establish that C answers the question or that A and B agree.
- A hybrid fusion method cannot guarantee embeddings “can only help.” A
  graph expansion cannot promise precision remains flat on every question.
  Improvements are measurements, including failures, not design axioms.
- Corpus size is not bounded “by construction.” No site-page cap justifies
  flat-vector latency/memory promises. Set a supported envelope and benchmark.
- The tokenizer splits Unicode words by whitespace; it does not supply CJK
  word segmentation. The bilingual workload is a reason to test real questions,
  not assume an English fixture set represents readers.
- Chunk identity is repeatable for unchanged inputs, not stable under editing;
  `Corpus.Version` is loaded as the literal `v1`, not an edition/content hash.
  A displayed version does not currently prove archive freshness.

**Repair:** (1) commit real/held-out questions and source expectations,
including unanswerable cases, and measure retrieval without a model; (2)
measure assertion support, abstention, evidence coverage, URL correctness,
and latency with pinned model/machine details, fixing the prompt evidence
budget first; (3) try bounded graph expansion and optional embeddings only
against that baseline. Keep lexical/evidence-only use available. A fake
provider tests wiring, not support judgments.

**Gate / kill:** a reader should complete a task more reliably than with
ordinary search, not just see recall@5 rise. Include a valid citation with a
false assertion in evaluations. If synthesis makes audit harder, keep the
retrieved evidence and stop polishing the answer. Quote existence, semantic
support, and the source's factual truth need distinct labels.

**What it obsoletes:** “cited means correct” and demo-only retrieval decisions.
It need not obsolete browsing or require the maintainer to run a public AI
endpoint. The current assistant is a local author/reader companion, **not**
the hosted site's search service; Homestead cannot silently assume every
visitor has a daemon and model installed.

#### Corpus Packs: keep #4 on probation; most of the transport story is vapor

A coherent portable publication is a plausible identity-changing bet. The
original leaps from exporter-plus-cache to a signed, versioned subscription
system as though the format were nearly done. It is not:

- RAG exports are monolithic bundles generated separately from HTML. There
  is no same-revision binding between the source corpus and output tree.
  #570 illustrates why “files created” is not even proof of content coverage.
- The build cache is private implementation state, includes operational path
  inputs, and is not an interchange contract or an edition archive. `graph.json`
  has no explicit schema-version field; the explorer payload's deterministic
  sorting is not proof every generated artifact is globally reproducible.
- A file-member delta must include a changed monolithic content bundle unless
  phase one creates finer-grained members. The promised one-page chunk delta
  cannot be obtained from the existing payload by merely diffing files.
- Fetching a full current pack **and then** diffing locally does not save
  download bytes. A real delta protocol needs published base/target IDs,
  delta availability, mismatch handling, deletion semantics, and a full-fetch
  fallback. “95% shared” is not a transport implementation.
- `ask` uses a loose-file loader and generated metadata for URL enrichment.
  A pack reader must preserve that contract and serve or map its citation
  targets; it is not just a filename change. Signatures appear in the headline
  but not in these four phases; the cut Verified Corpus proposal owns them.

**Repair:** (1) ship an ordinary downloadable archive of one real site with
explicitly public source, output, license, edition identity, and content
hashes; (2) demonstrate local reading/search/graph/citations on a second machine
without contacting the origin, using loopback serving if needed; (3) preserve
and compare two complete editions. Only afterward propose updates/deltas if
actual reader usage and size justify them. Do not invent a new extension or
protocol as the price of testing whether anyone wants the files.

**Gate / kill:** a second person must actually use a downloaded edition. If
that does not happen, no updater. Archive extraction needs path/symlink and
size limits; downloaded active HTML needs origin isolation; source selection
must exclude nonpublic repository/config bundles. Checksums prove integrity,
not publisher identity or truth. This boundary work is absent from the price.

**What it obsoletes:** incoherent downloads and dependence on one live host,
not a hosted workspace merely because an archive has a special extension.
The portability spine survives; the signed subscription platform remains an
unearned extrapolation.

#### Vault Mode: demote #5 from funded bet to conditional reserve

A folder of linked notes is a credible workload. The claim that the generator
already implements Obsidian's core loop is not. `internal/stub/stub.go` writes
“Missing Page” **output HTML** and a graph node. It does not create a note in
the source vault, edit that note, or synchronize missing-note state back to an
editor. Filling a generated stub is not a supported authoring loop.

The graph explorer also does not expose a browsable URL for raw nodes/stubs
without `pageOutputs` entries. “Unresolved index links every stub” needs new
identity/output-path work. Existing normalization drops non-Latin taxonomy;
#571 is relevant to real vaults too. Body `#tags` need actual parsing rules
that do not confuse headings, code, URLs, and punctuation with taxonomy.

A native vault importer additionally needs to decide how duplicate basenames,
relative links, alias syntax, attachment embeds, heading/block links, and
existing assets resolve. Choose a restricted subset and show unsupported
syntax; “any folder” is an interoperability promise, not a minimum phase.

**Most dangerous gate:** phase three treats `render: false` or exclusions as
though safe selective publication already exists. Raw bytes can ship; the
RAG exporter walks source content independently of rendering. Correct public
selection must apply before every output writer and export, with tests for
private text, titles, paths, metadata, graph edges, and linked assets. A
zero-orphan requirement is unrelated to preventing leakage.

**Repair:** (1) import one volunteer's explicitly public vault non-destructively,
with a documented supported-syntax subset and ambiguity report; (2) produce
useful linked/searchable output and a source-level unresolved-note report;
(3) demonstrate local questions or publication of that public corpus. Private
subset publication is a separately scoped security promise, not a flag in
that third rung. Stop if the author does not prefer the output to the existing
publishing workflow.

**What it obsoletes:** a hand-maintained duplicate of that author's public
notes. It does not obsolete Obsidian or existing vault publishers. The claim
that no Markdown SSG/editor can publish navigable linked notes is unsupported
and unnecessary. An importer alone is a feature, not a product transformation.

### The seven runners-up: short verdicts, not second proposals

| Original candidate | Attack and disposition |
| --- | --- |
| **The Studio (16–22 days)** | Real operator fit, but it assumes incremental accept/rebuild, editing, durable proposal state, diff review, and safe publishing are nearly assembled. TUI menus/diagnostics are a shell, not an editorial console. `RunSync` local publishing stages **all** working-tree changes via `AddAll`; it is not a scoped “ship this proposal” API. The merge policy can actually merge with approval gates; “never auto-merge” needs an explicit path. Keep a CLI receipt/editor escape hatch; fund UI only after an observed review bottleneck. |
| **Time Travel (12–18)** | A manifest plus hash/timestamp cannot reconstruct a previous page's prose, assets, or rendering inputs. Its scrubber needs retained content or full editions, identity across renames, retention semantics, and correct old URLs. `--as-of` cannot infer historical beliefs from current chunk IDs. Keep behind real edition retention; the file-opening guarantee is unproven here too. |
| **Media Pipeline (10–14)** | Useful table stakes, explicitly not a moonshot. `image/jpeg`, `image/png`, and `image/gif` decode/encode; they do not provide a resizing filter, EXIF parser/orientation policy, or text layout for social cards. Dependency minimalism does not make those algorithms free. Asset-specific invalidation also does not exist in the whole-build cache. Do the workload-required slice, not the advertised three-phase subsystem on its stated budget. |
| **Routines (14–20)** | Can game findings by deleting pages, descriptions, or links. Closing a stub does not inherently lower orphan count. “Improves or holds” admits arbitrary churn; a sandboxed copy is not an OS/process/network sandbox. The existing publishing/merge loop is not a prove-value gate. Kill unattended content production; keep reproducible diagnostics and bounded human-reviewed corrections. |
| **The Workshop (14–20)** | Theme tokens, slots, and a previewer are the “chores with ambition cosplay” the brief forbids. No audience or concrete plugin/component demand, and the roadmap explicitly defers large redesigns. The TUI has serving/watch, not a layout/token preview abstraction. Cut as a strategic bet. |
| **Verified Corpus (10–14)** | Integrity is worth doing when consuming somebody else's edition, but hashing must use content hashes distinct from position-derived IDs. Key custody, trust roots, revocation/rotation, and failure modes are unpriced. Signing cannot establish factual truth or eliminate prompt injection. Fold ordinary integrity into a real pack; optional signatures follow a real verification need. It is not an independent moonshot. |
| **Shared Brain (28–40)** | Source edits can break today's chunk anchors; immutable annotation records alone do not define edit/delete/conflict behavior or a CRDT. Distributed merge needs a specified operation model. Identity, contribution transport, moderation, licensing, and prompt-injected annotation evidence are absent. Local-first collaboration is not inherently a privacy contradiction—the original argument is overstated—but this still solves too much unrelated product work. Cut until actual collaborators ask for it. |

### What the original document is missing

1. **A named reader and owner, not just a plausible market.** Who maintains
   the first independent corpus? Which task warrants local questions, a pack,
   or a vault importer? “Real demand” in the catalogue is asserted, not shown.
   A human directing agents remains responsible for editorial decisions;
   cheap code does not create cheap attention or reliable content.
2. **The difference between source, public output, and assistant corpus.**
   They are built through different paths and include/exclude different
   material. `publish-check` checks the output contract/references and reports
   stubs; it does not require RAG or verify its completeness. A valid artifact
   can still contain a stale/empty corpus or exposed source. This needs explicit
   coverage and same-revision evidence, not a green build banner.
3. **Identity and compatibility policy.** Page paths, slugs, chunk positions,
   content hashes, publisher identity, and historical edition identity are
   different things. New manifests must be additive/opt-in, versioned, and
   tested with old consumers. Packs do not require Ledger to land first merely
   because both need hashes; avoid inventing dependencies that block a demo.
4. **Threat models at the new boundaries.** Local loopback hosting constrains
   exposure, not prompt injection from corpus text. Import/download/update
   adds hostile paths, active HTML, oversized files, privacy leaks, and origin
   trust. A routine spec that can execute arbitrary checks adds a different
   boundary again. “No SaaS” is not a security proof.
5. **Adoption, migration, and continuing ownership.** Install/upgrade paths,
   supported corpus sizes and machines, plain-file export, retirement of
   unused surfaces, and time spent reviewing agent output belong in the budget.
   The repository's black-box release review (#500) already found real friction;
   repeated external-style validation is better grounding than uniqueness claims.
6. **Falsifiable promotion/stop rules and what gets obsoleted.** The original
   candidates have risks but mostly no point at which we stop. More files,
   higher recall, or more PRs are not reader outcomes. Put the old workflow and
   the proposed artifact side by side; retain the latter only if someone uses it.

These are holes in the existing bets, not a request for six additional epics.

### Does the ranking survive?

| Revised priority | Original candidate | One-line verdict after the attack |
| --- | --- | --- |
| **1 — do the first rung** | Homestead | Still beats everything: one owned, useful live site answers questions that another speculative subsystem cannot; its three-site bake-off does not survive. |
| **2 — bounded proof** | Change Ledger | Actual reviewer-visible consequences use the code well; keep the comparison experiment, drop automatic policy and TUI scope until someone benefits. |
| **3 — measure before extending** | Retrieval, Revisited | Eval-first survives, but prompt evidence and unsupported assertions outrank embeddings; no “grounding solved” claim survives. |
| **4 — demand-gated** | Corpus Packs | Portable coherent editions have real leverage; signed subscriptions and deltas are vapor until a second reader uses an ordinary archive. |
| **5 — reserve, not funded** | Vault Mode | A public-vault trial remains plausible, but existing stubs are not a note workflow and selective publication is unsafe as specified. |

**The ordinal order narrowly survives as a sequence of experiments. It does
not survive as a cost-adjusted top-five investment case.** No runner-up earns
promotion merely because Vault weakens; leaving the fifth slot unfunded is
better than manufacturing certainty. The original twelve include several
ordinary features and enabling layers; meeting the count did not mean twelve
moonshots were actually found. Studio could move up if real review time,
rather than public reading, becomes the demonstrated bottleneck—not because
agent-forward branding makes an inbox automatically valuable.

**Recommendation to the owner:** authorize Homestead's one-site first rung,
keep current bug fixing ordinary, and demand actual reader/reviewer evidence
before authorizing the next phase of any other proposal. Do not build Ledger
before allowing the site to ship; do not require a new pack format before
allowing anyone to download it; do not improve retrieval before observing
what evidence the model was given. If the first site is useful, these bets get
better inputs. If it isn't, the stop is information—not a failed pitch deck.

---

## Homestead first rung: the existing bilingual site

**Decision:** use the existing bilingual workload behind #570–#572. Do not
start another field manual, run a bake-off, migrate to a new theme, or generate
content to meet a quota. This is a bounded **delivery brief**, not proof that
the site has shipped and not authorization to deploy it.

The issue reports identify a local `sites/zai` project with **46 content pages**
on 2026-09-29. Subsequent source discovery located the public Z.ai Field Guide
repository and a live URL, with **58 Markdown files (29 EN + 29 ZH)** at the
reviewed revision. The [launch-input register](#homestead-launch-input-register)
below supersedes the earlier unknown-source/URL inventory, distinguishes source
from deployment evidence, and records the remaining owner, rights, compiler,
export-scope, and reader gates. The site exists; Homestead acceptance is not
therefore complete.

### Outcome and hard boundary

Publish—or verify an already published—independently useful site at **one
actual HTTPS URL**, using a pinned released La Famille binary. Preserve its
approved existing content and navigation. Demonstrate one real reader task
in each language without requiring the reader to install Go or a model.
The artifact is the live site plus a short, reproducible launch record in
**the site's repository**, not another generator feature.

**Included:** source/publication inventory, reproducible build, public-safe
content export, targeted URL/browser checks, one approved publication or
verification of the existing deployment, and reader evidence.

**Excluded:** fixing #570–#572 in this task; i18n/translation machinery;
pagination, redirects, plugins, new themes, image processing, Ledger, packs,
retrieval upgrades, public `ask`, automatic publishing, and a second site.
Needed generator fixes become separate issues/PRs with same-package tests.
The content needs no new translations to pass; the selected slice must
already exist in both languages. Do not silently drop the other pages.

### Gate 0: do not build blind

Record these inputs before implementation starts. An unresolved item means
**blocked**, not “an agent will choose later.”

| Required input | Acceptance |
| --- | --- |
| Source | Exact site repository or accessible source path and immutable starting revision; identify the content/config/templates/assets actually used. |
| Owner | Named human who owns editorial claims, public/private selection, and ongoing availability. A coding agent is not the editorial owner. |
| Audience | One sentence naming who reads the site and what they should accomplish; one concrete task in each actual language, with expected answers/pages chosen before testing. |
| Public destination | Existing or owner-approved HTTPS origin **including its deployment subpath**, publishing mechanism, authorized deployment operator, and a way to restore the last known-good artifact if one exists. Verify ownership/access; do not register anything or switch providers as part of this brief. |
| Publication scope and rights | Explicit intended page/asset list, omissions with reasons, reuse/license decisions, and any material that must not leave the machine. Confirm whether the reported 46 pages are still the right inventory. |
| Compiler | Pinned release tag, platform archive, verified matching SHA256SUMS digest, binary build identity, and help output for the commands/flags below. The release list currently identifies `v0.1.0-prealpha` as latest; that is an observed candidate, not a claim it supports every current-source behavior. |
| Approval path | Where the human reviews source/artifact evidence and how the approved revision reaches the host. No agent-PR policy or cron job substitutes for launch approval. |

If the site is already live, first record its current URL, source revision,
and representative behavior; avoid a redeploy merely to manufacture a launch.
If source, rights, ownership, or public destination cannot be established,
stop here with the missing-input list. Do not fork a new destination instead.

### Bounded work and spending cap

After Gate 0, allow **two focused implementation sessions, at most six hours
of agent work in total**, plus one **up-to-one-hour human review**. These are
scope caps, not feasibility estimates. Record actual effort separately for
inventory, artifact validation, publication, and review. If a session ends,
leave its files and failures reproducible; do not burn time expanding scope.

Use three checkpoints:

1. **Inventory and reproducible artifact.** Preserve a starting revision and
   build a release workspace with only approved public inputs. Keep the site's
   layout, URLs, language structure, and existing taxonomy conventions. Record
   the intended output and export coverage before building. Produce checked
   output plus all warnings, failures, and workarounds. This is useful even
   if deployment is blocked.
2. **Human-approved publication or live verification.** Review the exact
   source/artifact pair, run the launch checks below, and obtain explicit
   approval before an authorized operator publishes. Verify the actual URL
   after upload; the local artifact passing is not the remote result.
3. **Reader evidence, within seven calendar days of publication/verification.**
   A consenting person who did not author the selected pages attempts a task
   in each language. One bilingual reader can do both; otherwise use one
   reader per language. Record completion, the page/path used, confusion, and
   any observed failure without recording unnecessary personal information.
   One focused correction and recheck can fit the remaining session budget.
   A lack of available readers remains “reader validation pending,” not a
   successful task inferred from an agent walkthrough.

At the cap, stop and hand off either an accepted first release or a concrete
blocked/failed record. Do not extend the deadline, manufacture readers, or
quietly pull generator work into the sessions. Any continuation needs a new
bounded decision. No new content-production budget is hidden off the books.

### Known friction: ordinary work, explicit dispositions

Statuses below were checked on 2026-09-29; all three reports remain open.
Recheck against the **selected binary and site**, not just generator master.

| Report | First-release disposition | Stop condition |
| --- | --- | --- |
| #570: empty RAG with relative project root | Use the report's known working form: invoke from the site root with `--project-root .` and an absolute archive output. Independently check intended document coverage; an exit code or nonzero file size alone is insufficient. Record whether the selected release actually exhibits the problem. | Intended content is missing, empty, stale, or unparseable after the bounded workaround. Do not publish a knowingly incomplete archive or call `publish-check` proof it is sound. |
| #571: dropped non-Latin taxonomy | Preserve a documented ASCII taxonomy vocabulary **only if the site already uses it and its owner accepts it**; preserve native-language titles and prose. Verify archives for the intended terms. | Native-language taxonomy is an owner-required launch condition or important meaning/navigation is lost. Record that requirement; do not silently transliterate, invent translations, or reduce a bilingual site to English. |
| #572: unused fallback theme assets | Accept the reported roughly 1.5 MB overhead for the first release if the host budget allows it. Record measured artifact size and unused files; retain a prune step only if it already belongs to the site's reproducible publishing process and validate afterward. | Host limits fail or pruning breaks referenced runtime files. Do not add a new asset-selection subsystem or an unrecorded post-build deletion step. |

Documented warnings can be accepted individually with owner/reason; malformed
content, broken required paths, leaked material, and missing intended export
content cannot. Neither a zero-warning dashboard nor erasing every workaround
is the goal.

### Build/export recipe and public-data boundary

Use a clean **public-only release workspace** rooted at an immutable source
revision. Inventory what goes into it; ordinary Git ignores and
`render: false` are not privacy controls. Approved raw Markdown is public,
and its presence in graph/meta must be intentional. Referenced assets and
source filenames can also disclose information. If public/private selection
is uncertain, block publication rather than rely on removal after generation.

Retain config/layout choices; set the release config's `siteurl` to the actual
approved URL including subpath. Record any release-config changes and the
exact config/input hashes alongside the source revision; a revision alone does
not identify an edited workspace. Freeze these inputs before validation.
Do not run `init --force` over the existing site. Evidence files live outside
source inputs and outside `public/`.

The following is a **command template, not an executed transcript**. Fill
absolute paths and the approved URL, verify flags against the pinned binary's
help, and record commands, exit codes, and both output streams. If the chosen
release does not support the required contract, record a compatibility blocker
rather than silently using `go run` or inventing flags. **The deployed
`v0.1.0-prealpha` does not support the final strict-check command below; the
[launch-input register](#homestead-launch-input-register) records that blocker.**

```bash
set -eu
BIN=/absolute/path/to/verified/la-famille
SITE_ROOT=/absolute/path/to/public-only-release-workspace
RUN=/absolute/path/to/evidence-directory
SITE_URL=https://actual-owner-approved-origin/actual-subpath

# Prepare RUN outside SITE_ROOT and public/, and set siteurl in the release config.
cd "$SITE_ROOT"
"$BIN" --version --json
"$BIN" --project-root . check --asset-health
"$BIN" --project-root . build --output "$SITE_ROOT/public" --site-url "$SITE_URL"
"$BIN" --project-root . rag --output "$RUN/rag-archive"
```

After `rag`, compare the parsed `rag-content.md` document paths and representative
text against the approved expected corpus—not merely its byte count. Record
any exporter-specific exclusions explicitly. Use an existing archive reader
or a small site-local verification helper that understands marker escaping;
a line-count/grep approximation is not a parser. Do not add a generator command
for this launch. Require evidence that pages from both languages are included
and belong to the frozen inputs and source revision. Review
raw-page inclusion for public exposure even when `ask` would exclude it.
System/config bundles stay in the private evidence workspace; **do not publish
`rag-system.md` or `rag-config.md` in this first release**. Publish only the
reviewed content bundle at `public/rag-archive/rag-content.md`, copied using
the site's documented release procedure. This is a downloadable content
artifact, not a pack or a public assistant service.

```bash
# After reviewing and copying ONLY the content bundle into public/rag-archive/:
"$BIN" --project-root . publish-check \
  --output "$SITE_ROOT/public" --site-url "$SITE_URL" --strict --json
```

`check` must have no errors; record accepted warnings. `publish-check --strict`
must pass, including no generated missing-page stubs. It does **not** validate
archive coverage, source truth, or hosting. Run the final check after any
existing prune/copy step; do not rebuild after copying the corpus without
repeating the export/copy/validation sequence. Keep the same source revision
throughout and preserve the reviewed output digest/file inventory. Any source,
config, template, or asset change restarts artifact validation before approval.

### Launch acceptance: every row needs evidence

| Check | Required evidence |
| --- | --- |
| One independent site | Actual approved HTTPS URL, source revision, binary identity, and deployment/artifact identity. “Would deploy” and a local screenshot do not pass. |
| Source and artifact agree | Intended page/asset inventory with actual outputs and documented omissions; no silent content loss, unexpected raw private files, repository notes, or extra system/config export. |
| Bilingual navigation | Test six representative existing paths, three per language, including each language entry and a content page. Test existing cross-language links if present; lack of translation-switch machinery alone is not a launch blocker. All previously published intended paths stay reachable. |
| Reader discovery | A task finds its target by ordinary navigation or search. Test known CJK/native-language queries if applicable; unsupported results are a recorded failure/workaround, not a guessed pass. No local model needed. |
| Existing static features | If enabled, load graph explorer over HTTP and select a known page; if intentionally disabled, record that decision rather than enabling it for this exercise. Verify intended tag/category archive links when the site uses taxonomy. Confirm canonical/sitemap URLs and asset URLs use the deployed subpath correctly. |
| Public content export | Parsed document coverage and representative text from both languages match the approved revision; download the deployed content bundle and confirm it matches the reviewed local artifact. |
| Remote artifact works | Fetch/inspect each representative path, search/graph payload, and referenced runtime resources at the real URL; check status, expected content, and browser console/network behavior. HTTP 200 alone can be an error page or host fallback. |
| Approval and recovery | Human approval tied to the source/config/artifact identity; authorized publisher and last known-good artifact/recovery procedure recorded. If remote checks fail, stop further uploads and restore only through the authorized operator. Never add the automated merge label as a substitute. |
| Reader outcome | Attempted task and observed outcome in each language, with failures preserved and no success inferred from build logs. |

No site restructuring or theme overhaul is permitted to make the check matrix
look impressive. Reuse existing features; the question is whether this site
works, not whether every La Famille capability has a demo.

### The launch record and finish line

Commit a short `HOMESTEAD.md` to the **site repository**, outside its rendered
content directory. Keep transcripts/manifests beside that record or link to a
retained CI artifact. Sensitive evidence stays private; publish only sanitized
summaries with permission. Capture:

- Named owner, audience, actual languages, public URL, source revision,
  verified release/archive identity, and publication date/operator.
- Intended public page/asset/export inventory and licensing decisions.
- Exact command/exit-code evidence, final artifact inventory/digest, and
  acceptance-matrix results with representative URLs.
- Warnings and #570–#572 dispositions, accepted workarounds with reasons,
  reader task outcomes, and any correction/recheck.
- Actual agent work and human review effort, next review date, and an honest
  status: **accepted first release**, **live but validation pending**,
  **blocked before publication**, or **failed acceptance**.

**First-rung done:** the approved live URL and artifact are verified, intended
content is preserved, both language tasks have observed outcomes meeting the
chosen expectations, and the launch record is committed. Reader failure may
produce a useful live release, but not an unqualified Homestead success.

**Later rungs are not silently included:** schedule an owner check about two
weeks after launch, then decide whether to fund a second-contributor correction
and two maintenance cycles. Those would demonstrate durability; this first
release does not. If nobody owns continued usefulness, archive or retire the
experiment honestly rather than schedule an agent to manufacture activity.

### Homestead launch-input register

**Status: existing site is live; Gate 0 and first-rung acceptance are not
complete.** This is an evidence register in the generator repository, not a
second launch log or a modification to the site. Update the site's existing
[`docs/HOMESTEAD.md`][zai-homestead] for subsequent site-local execution evidence.
Its [`docs/VERIFICATION.md`][zai-verification] records a prior claim audit;
that record's assertions are not independently re-certified by this review.

The reviewed immutable source is
[`drawmeanelephant/z.filed.fyi@888f502588a12624d90b9be1f7df0c7cd7b397a3`][zai-source]
(default branch `main`). All source links below are pinned to that revision.
Mutable branch heads, URLs, and CI artifact availability must be rechecked
before any later implementation. This pass used read-only repository/API/log
inspection and fetched public pages. It did **not** run site scripts, rebuild,
deploy, retrieve credentials, test an interactive browser, or observe readers.

#### Gate-by-gate findings

| Launch input | Evidence located | Disposition |
| --- | --- | --- |
| Source and revision | Public [site source][zai-source]; `config.yaml`, `content/`, `templates/`, `assets/`, and `scripts/` identified at the revision above. | **Located.** No longer rely on the old local `sites/zai` path. |
| Human owner | GitHub repository is owned by `drawmeanelephant`; authored pages identify “Z.ai Field Guide,” not a named human editorial owner. | **Unconfirmed.** @drawmeanelephant is the candidate owner, not an accepted accountability assignment. Owner-confirmation questions received no selections. |
| Languages and audience | [README][zai-readme] and source establish English at `/`, Simplified Chinese at `/zh/` (`zh-CN` layouts). Public about pages describe an independent guide to Z.ai/Zhipu, its models, products, history, and sources. | **Languages established.** Proposed audience: English- and Chinese-reading developers/readers comparing the GLM model line and product surfaces. Owner adoption of that audience/tasks remains pending. |
| Existing destination | [Config][zai-config] pins `siteurl: https://z.filed.fyi`; public `/` and `/zh/` return HTTP 200 with expected guide content. No URL subpath. README identifies Cloudflare Pages project `z-filed`, alias `https://z-filed.pages.dev/`. | **Live destination observed.** No new hosting recommendation or provisioning needed. Human authority over publishing/domain/account remains unconfirmed. |
| Public inventory | Source tree has 58 Markdown files, four layouts, ten partials, and 15 files under the site's asset directory. Two Markdown files deliberately use `render: false`. | **Inventoried, not signed off.** Candidate public scope is all current content, intentional raw examples, referenced assets, and content-only RAG; it is not owner-approved yet. |
| Rights | README says “Licensing placeholders: TBD by the repository owner”; no top-level site license was located. Shipped font notices and colophon provenance exist. | **Unresolved.** Public visibility and source citations do not establish ownership or grant a reuse license. Owner confirmation for authored text/artwork was not received. |
| Compiler and build | [Deployment workflow][zai-deploy] pins `v0.1.0-prealpha` Linux amd64; [successful run][zai-run] records SHA256SUMS verification, binary commit, source SHA, and completed build/export/postprocessing. | **Current compiler identified; brief compatibility blocked.** Release lacks the required strict publish-check flags; details below. |
| Approval and recovery | Existing workflow deploys every push to `main` and supports manual dispatch. Successful run records a deployment-specific URL and retained `site` artifact. No explicit human launch/recovery gate is defined in that workflow. | **Mechanism located, authority/signoff unresolved.** Do not push to site `main` or dispatch it as part of input resolution. Candidate recovery artifact is not an approved rollback procedure. |

The user selected this existing bilingual site, but did not answer the later
owner/scope/rights selections. Treat the documentation request and eventual
review of this PR as permission to record evidence—not as blanket rights
attestation, launch acceptance, or authorization to deploy.

#### Freeze the actual source set, not the old page count

The pinned source tree contains **29 English + 29 Chinese Markdown files**.
Twenty-eight per edition render as HTML; `la-famille/raw-sample.md` in each
edition is deliberately public raw Markdown. The 46-file count in the earlier
issue reports and audit introduction is a historical snapshot, not today's
coverage denominator. CI reports **85 generated pages** including taxonomy;
that build counter must not be relabeled as 85 source documents.

The 29 relative paths in each edition are:

```text
index.md                         api/index.md
products/index.md                chat/index.md
code/index.md                    autoclaw/index.md
autoglm/index.md                 bigmodel/index.md
qingyan/index.md                 zcode/index.md
ecosystem/index.md               company/index.md
company/open-source.md           history/index.md
history/origins.md               history/chatglm-era.md
history/pivot.md                 history/engineering-era.md
models/index.md                  models/earlier.md
models/glm-4-x-era.md             models/glm-5-family.md
models/glm-5-3.md                 models/glm-5-3-flash.md
meta/about.md                    la-famille/index.md
la-famille/how-this-site-works.md la-famille/colophon.md
la-famille/raw-sample.md
```

English paths are under `content/`; prepend `content/zh/` for Chinese.
Preserve all intended existing routes; the six reader-test URLs below are
samples, not permission to publish only six pages. Source identity is the
pinned Git tree: content tree `10e58d5a49a6718e3f1a3f43840c541117855304`, config
blob `e43df812ddca450b1ac90fcb82506e5dd3a47813`, templates tree
`224bcaf11249995f16f5a93b7939bd727720e104`, assets tree
`cf0f065875150bf98005319b6ac259372c6d2bae`, and scripts tree
`459070d18edb288543b9e463c71cbc9cbb1fd3e9`. These are **Git object IDs**, not
independent SHA256 attestations of local input bytes. Any edited workspace
still needs its final input/config hashes recorded before approval.

The [font inventory][zai-fonts] identifies four WOFF2 files, General Sans FFL
and JetBrains Mono OFL notices, and a system CJK font stack. Assets also include
site CSS/JS, a favicon, an OG image, and three abstract images. The [colophon][zai-colophon]
says the three abstract images were generated for this project; that is a
provenance assertion, not proof of ownership. The shipped FFL text permits
own-site self-hosting but separately restricts redistribution. Do not infer
permission for a new downloadable font-containing pack or blanket licensing
of third-party files; no such package is part of this task.

#### Existing publishing receipts and compiler incompatibility

[Run 36623067663][zai-run] built the exact reviewed source revision and completed
deployment on **2026-09-29 at 19:59:24 UTC**. Its log records:

- `la-famille_0.1.0-prealpha_linux_amd64.tar.gz: OK` against the release
  `SHA256SUMS`; binary commit `896ec96a51f988b0108aeda4e86ae75451dc3ac8`,
  built with `go1.25.0`. [Release asset metadata][zai-release] publishes the archive
  SHA256 `0731dc7cb750e9a77f1ed72c61016ddf1a935002863d5cd3cbbc7dbf5c22e684`.
  This pass inspected those receipts, not a fresh downloaded binary checksum.
- `check`: 0 errors, 0 warnings; `build`: 85 generated pages, cache miss;
  `publish-check`: valid artifact, 118 files. CI also checks a nonempty content
  bundle and generated `404.html`.
- Existing site-local sequence: check → build → RAG export into `public/` →
  strip internal `nofollow` → enhance artifact → prune unused fallback assets
  → publish-check → upload → deploy. The pinned enhancement script alters
  hreflang, graph styling, search data/client, taxonomy presentation, sitemap,
  404, and asset URL versions. These scripts are part of the actual publishing
  inputs; “unmodified generator” does not mean “unmodified output.”
- Deployment-specific URL reported by CI:
  [`https://935e8988.z-filed.pages.dev/`](https://935e8988.z-filed.pages.dev/).
  Its homepage and the custom domain returned expected guide text in this
  pass. That correspondence is not a byte-for-byte comparison of every live
  file with the artifact.
- Retained [artifact 11059661647][zai-artifact], name `site`, compressed size
  980,022 bytes, API digest
  `sha256:375b6083d7d61761d1215e68e6b79a88dff42d446c7a8868d04124c55201e489`,
  recorded expiry **2026-10-06 19:59:05 UTC**, not expired when inspected.
  The digest is for the uploaded artifact, not a parsed public-tree manifest
  or a verified rollback. Copy/retain it only through the approved evidence
  process before expiry; do not depend on seven-day retention indefinitely.

The compiler gate is **not** closed by this successful run. At the recorded
release commit, [`publish_check.go`][zai-release-check] defines only `--output`
and `--json`; it has **no `--strict` or `--site-url` flags**. Its JSON shape is
also the older manifest, not the current valid/stubs/errors report. The site
workflow calls plain `publish-check`, not the newer command in the brief.
Release source confirms this incompatibility; binary help/new-flag failure
was not executed here. Never present master-only flags as release evidence.

Keep the reviewed release pin as the **existing deployment input**. Before
first-rung acceptance, the owner must either choose a new released binary
that supports the required gate and validate it, or explicitly amend the
brief to require an independently tested site-local no-stub/reference check
with the existing binary. The latter is a proposed decision, not a check we
have implemented or an approved downgrade. A silent `go run` fallback or a
merely renamed “strict” step does not resolve compatibility.

#### Public corpus boundary: found, but not accepted

The successful manifest lists **all three** RAG bundles. Public requests to
`/rag-archive/rag-content.md`, `/rag-archive/rag-system.md`, and
`/rag-archive/rag-config.md` returned HTTP 200. The fetched system excerpt
contains public workflow source with symbolic secret references—not retrieved
credential values. The config bundle contains asset/template inventory.
Their existence is an observable mismatch with this brief's **content-only
export** boundary; it is not proof that credentials have leaked.

Choose a reviewed content-only public export in a **separate site change**, or
obtain an explicit owner-approved exception after reviewing the complete
bundles and updating the scope. Neither has happened here. Removing public
bytes is also a deployment-affecting act and requires that site's authorized
operator; this docs PR removes nothing remotely.

The workflow's `test -s rag-content.md` and this pass's truncated readable
excerpt establish nonempty output, **not full coverage**. No parsed 58-document
comparison, per-document source-byte match, or complete private-material audit
was performed. The two intentionally raw examples must be considered in the
public export scope even though the assistant would exclude them. Approving
current public visibility by inference would erase the very gate we added.

#### Proposed tasks for the later reader test

**Proposed audience:** developers and researchers who need to distinguish
GLM product/model capabilities and license constraints without treating
vendor claims as independent findings. This wording follows the site's
existing purpose; a human owner still needs to adopt it. These tasks read the
guide; they do not recommend or integrate a vendor service.

| Language | Starting point and prompt | Proposed expected result |
| --- | --- | --- |
| English | Start at `https://z.filed.fyi/`. “Does this guide say GLM-5.3 and GLM-5.3-Flash have the same license? Find the difference and a primary source to check it.” | Reach `/models/` then `/models/glm-5-3/` (or a useful search result); identify the guide's custom/non-MIT vs MIT distinction and follow its model-card/license source. Preserve the guide's caveat that exact terms belong to the license text. |
| Simplified Chinese | Start at `https://z.filed.fyi/zh/`. “GLM-5.3 与 GLM-5.3-Flash 的许可相同吗？请在手册中找出差异，并指出可核对的一手来源。” | Reach `/zh/models/` then `/zh/models/glm-5-3/` (or a useful search result); explain the same distinction and identify a linked primary model-card/license source. Do not substitute the English page for native-language navigation. |

These expectations come from the pinned guide, **not an independent legal or
model-fact certification**. Source facts can change; retain attribution and
recheck authoritative terms when acting on them. The six URLs above all
returned HTTP 200 with matching English/Chinese guide titles and relevant
content during read-only fetching. That is a representative page check, not
an observed reader path, search interaction, or console/network audit.

A `/tags/glm-5-3/` fetch listed English and Chinese pages. A `/graph/` fetch
returned explorer HTML; its readable extraction includes hidden loading/error
labels, so **do not report a graph runtime error or graph success from that
text**. JS graph interaction, native-language search, assets, canonical URLs,
and all intended historical routes still need the brief's browser/artifact
checks. No person was recruited, no task outcome recorded, and the seven-day
reader window has not been declared completed or newly restarted by this PR.

#### Remaining decisions before calling the first rung accepted

1. **Human accountability and authority:** name the editorial owner and
   authorized publisher/domain operator; define how source/artifact review
   is approved. Current push-to-main deployment is an existing fact, not
   human approval of Homestead. Select a known-good recovery artifact and
   operator-controlled rollback procedure before changing production.
2. **Scope and rights:** approve the 58-file content inventory and referenced
   assets, confirm publication rights for authored text/artwork, preserve font
   notices, and decide content-only export versus a documented reviewed
   exception. No new downstream reuse license is granted by this register.
3. **Compatible acceptance checks:** resolve the release/strict-flag mismatch
   explicitly; then run the selected binary/help and required artifact checks
   on frozen inputs. Do not retrofit our newer acceptance wording onto an
   older green CI run.
4. **Corpus and remote verification:** compare parsed intended source/export
   coverage, record final byte/input evidence, and perform browser/network
   checks. Use the site's existing scripts/launch record rather than invent
   another generator subsystem or redeploying just to claim a new launch.
5. **Actual readers and upkeep:** have a consenting non-author attempt both
   language tasks, record outcomes, and assign an owner review date. Existing
   freshness automation checks some source claims; it cannot establish
   usability, full editorial correctness, or human ownership.

**Handoff result:** the site/revision/languages/URL/publishing pipeline and
compiler identity are resolved by evidence. Owner, rights, public-scope signoff,
strict compatibility, complete corpus/browser verification, reader outcomes,
and recovery authority remain explicit gates. The correct current label is
**live; Homestead validation pending**, not “launch blocked because no URL”
and not “Homestead complete.”

[zai-source]: https://github.com/drawmeanelephant/z.filed.fyi/tree/888f502588a12624d90b9be1f7df0c7cd7b397a3
[zai-readme]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/README.md
[zai-config]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/config.yaml
[zai-deploy]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/.github/workflows/deploy.yml
[zai-homestead]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/docs/HOMESTEAD.md
[zai-verification]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/docs/VERIFICATION.md
[zai-fonts]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/assets/fonts/README.md
[zai-colophon]: https://github.com/drawmeanelephant/z.filed.fyi/blob/888f502588a12624d90b9be1f7df0c7cd7b397a3/content/la-famille/colophon.md
[zai-run]: https://github.com/drawmeanelephant/z.filed.fyi/actions/runs/36623067663
[zai-artifact]: https://github.com/drawmeanelephant/z.filed.fyi/actions/runs/36623067663/artifacts/11059661647
[zai-release]: https://github.com/drawmeanelephant/la-famille/releases/tag/v0.1.0-prealpha
[zai-release-check]: https://github.com/drawmeanelephant/la-famille/blob/896ec96a51f988b0108aeda4e86ae75451dc3ac8/cmd/la-famille/publish_check.go
