# La Famille binary quickstart

Verify the archive with its matching `SHA256SUMS` entry, then run the binary
without a Go toolchain or source checkout:

```bash
./la-famille --version --json
./la-famille --project-root /path/to/site init
./la-famille --project-root /path/to/site new index --title Home
./la-famille --project-root /path/to/site build
./la-famille --project-root /path/to/site publish-check --output public
```

Relative paths come from `--project-root`. The generated `public/` directory
is the complete static artifact; incremental cache state stays beside the
project root.

## Release & changelog convention (decided 2026-08-20, #466)

Hybrid: **GitHub Releases are the source of truth**.

- `release.yml:165` publishes each tag via `gh release create --verify-tag --generate-notes` after proving the tag commit equals the built commit (`.github/scripts/release/tag.sh:resolve_and_checkout`). Notes are auto-generated from merged PR titles between tags.
- `content/docs/changelog.md` is a **curated snapshot** updated once per release (not per PR). Copy the highlights from the generated Release notes, de-dupe, and add human context. Keep the header's link to `https://github.com/drawmeanelephant/la-famille/releases`.
- No per-PR changelog lint gate; `jules` nightly maintenance and small fixes do not need a changelog entry. If you want a check, watch for drift between `content/docs/changelog.md` and the latest Release during release.
- This keeps `plan.md:15` and `content/meta/roadmap.md:20` honest: one convention, automated where it matters (provenance + notes generation), curated where humans add value.
