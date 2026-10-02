# PR #617 follow-up: numeric issue references are not body tags

## Baseline and scope

- Branch: `droid/review-la-famille-github`, updated to master `da1a467`.
- Pack-backed Ask is already delivered by #621; no pack or Ask changes.
- PR #617 worked around an issue reference becoming a numeric taxonomy tag.
  Fix the parser rather than requiring authors to avoid ordinary `#599` prose.
- Body hashtags must contain at least one Unicode letter. Preserve mixed
  alphanumeric and native-language tags, as well as explicit numeric
  frontmatter tags.

## Plan

1. Add same-package parser and metadata regressions for numeric references,
   valid mixed/Unicode hashtags, and explicit numeric frontmatter.
2. Add a generator regression covering archives and published metadata.
3. Require a letter when emitting a body hashtag and document the rule.
4. Run focused tests, formatting/lint/module hygiene, full tests, vet, race,
   and shuffled tests. Check the changed static-output contract.

## Static-output compatibility

Purely numeric body tokens no longer generate tag archives, page tag links,
or taxonomy metadata. Previously generated numeric-only body-tag archives may
disappear on rebuild; authors intending numeric tags should declare them in
frontmatter. Numeric frontmatter tags and other hashtag syntax remain unchanged.
No dependencies, pack formats, CLI flags, publishing policy, or deployment changes.

## Finish line

The fixed regressions and required quality checks pass. The user requested a
small PR; commit and push this bounded fix, then open a PR against master.
No issue creation, merge, release, or deployment is included.

## Completed validation

- New parser, metadata, and generator regressions reproduced the old failure
  before the fix, then passed afterward.
- `./format_check.sh`, `go test ./...`, `go vet ./...`,
  `go test -race ./...`, and
  `go test -shuffle=on -count=2 -parallel=4 ./...` passed.
- Formatting, module hygiene, debt markers, lint, file-size, agent-instruction,
  and log-scrubbing guards passed. Used the existing temporary golangci-lint
  v2.14.0 built with Go 1.27.1; no toolchain or module configuration changed.
- Repository statement coverage passed at 84.4% against the 75% floor.
- Compiled-binary fixture build and `publish-check --strict --json` passed.
  Issue-reference prose remains present, numeric reference archives are
  absent, and the explicit `2026` archive and manifest tag remain present.
- Reviewed the final diff; ready for the requested small parser-fix PR.
