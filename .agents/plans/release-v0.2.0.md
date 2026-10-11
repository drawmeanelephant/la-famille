# Task: cut v0.2.0 — first proper release

Task ID: `release-v0.2.0` (branch `t3/first-proper-release`)
Type: release engineering; one test-fix commit + docs/changelog update.

## Scope

1. Fix `internal/stub/stub_test.go:158` — stale `GenerateStubs` call signature
   (missing `renderer` arg) breaks `go test ./...` at master HEAD.
   Release blocker per repo rules: tests must be green before tagging.
2. Pick the version: **v0.2.0**. Rationale: `v0.1.0-prealpha` already consumed
   the 0.1.0 line; since then three feature epics landed (Change Ledger,
   Vault Mode, Corpus Packs) plus a large WOLF-review/dogfood hardening pass —
   a new feature epoch, not the same one graduating. Still pre-1.0: a breaking
   change (`refactor(ask)!`) merged this cycle, so v1.0.0 would overclaim.
3. Versioning scheme going forward (recorded in release notes + changelog):
   semver `vX.Y.Z`; while pre-1.0, minor = feature drops, patch = fixes;
   prerelease suffixes (`-prealpha`, `-rc.N`) for preview cuts, per the
   existing tag convention.
4. Update `content/docs/changelog.md` with the curated v0.2.0 section
   (convention: snapshot once per release, GitHub Releases stay source of truth).
5. Tag `v0.2.0` on the master commit containing the changelog update; push tag;
   `release.yml` tests/packages/creates the release; then replace the
   auto-generated notes with curated notes via `gh release edit`.

## Static asset generation impact

None intended. The only source change is a test-file call-site fix; no
generator, template, or pipeline code changes.

## What ships in v0.2.0 (for notes)

- Corpus Packs: build/verify/diff/apply, local + HTTPS feeds, durable watch (#612–#628+)
- Change Ledger: deterministic site manifest, build diffs, regression gate (#580)
- Vault Mode: wiki links, body hashtags, unresolved-note navigation,
  publish exclusion for unpublished notes (#589–594)
- Octoburger flagship theme; local-first self-contained themes; init showcase (#520, #561–563)
- Homestead init scaffolding; TUI glow-up (#564, #565)
- Flagship website + Cloudflare publishing (#605)
- Breaking: `la-famille ask` and the Ollama/local-assistant stack retired (#630)
- Hardening: WOLF review tiers 1–3, dogfood triage #506–518, and the
  build/pack/taxonomy/graph/markdown fix sweep landing at HEAD

## Validation

- `go test ./...`, `go vet ./...` green
- Local `go build` + smoke: `--version`, `init`, `new`, `build`
