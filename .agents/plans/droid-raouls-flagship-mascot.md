# Raoul(s) joins the flagship site

Task ID: `droid-raouls-flagship-mascot`

## Request and boundaries

Give Raoul(s), the La Famille Multipersonality Octodeveloper, a visible place
on the flagship website. Use his existing artwork and documented personas
from `content/meta/mascot-raouls.md`.

Do not remove or replace any Sexiburger or Octoburger artwork, placement,
favicon, copy, theme, or starter-project default. Do not edit or regenerate
the original mascot images. This is an additive website change.

## Implementation

1. Add a responsive homepage introduction with Raoul(s)'s original portrait
   and the Janitor, Maestro, and Skater artwork. Keep each illustration
   uncropped, lazy-loaded, dimensioned, and described for screen readers.
2. Link to the existing mascot story from the homepage and shared footer.
   Illustrate that story with its existing, documented images.
3. Add rendering regressions for Raoul(s), root/subpath links, original
   burger placements, and generated mascot page/artifacts.

## Static asset pipeline compatibility

No breaking changes expected. Reuse the existing asset-copy pipeline and
original filenames, including the Unicode ellipses. Do not change embedded
release assets, starter configuration, publishing workflows, or output paths.
Validate root and subpath rendering and the complete generated site.

## Validation

- First run the new rendering test before changing templates and confirm
  the expected missing-mascot failure.
- Run focused rendering and full-site publishing tests.
- Run `./format_check.sh`, `go test ./...`, `go vet ./...`, and
  `go test -shuffle=on -count=2 -parallel=4 ./...`.
- Build and run `publish-check` on an isolated website artifact.
- Attempt desktop/mobile visual checks and record any browser limitation
  without claiming unperformed checks.

## Publication

Work on a fresh branch from the deployed master. The user explicitly approved
**Open PR, merge, and deploy** for this follow-up. Commit the scoped changes,
open a PR against master, wait for passing CI, and merge without bypasses.
Verify the automatic Cloudflare upload and the live mascot and burger
placements. Do not change DNS, account settings, project settings, or secrets.

## Progress

- Confirmed the portrait and Janitor/Maestro/Skater artwork visually and
  read the original mascot story.
- All existing burger placements and image files remain protected.
- New root/subpath mascot rendering tests failed first because the
  introduction and original illustrations were absent; burger-preservation
  assertions already passed.
- Added the homepage introduction, four original illustrations, shared
  story link, and illustrated existing story.
- Focused rendering and complete generated-site publishing tests pass.
- `./format_check.sh` passed using the existing compatible temporary
  golangci-lint v2.14.0: zero issues. Full tests, vet, shuffled two-run
  stability, and patch whitespace checks pass.
- A fresh isolated build generated 201 pages and passed `publish-check`
  with 242 files (before the unchanged workflow adds its content corpus).
  Existing deprecated-layout fallbacks and the intentional missing-page
  example remain unchanged.
- Original portrait and all three persona URLs returned HTTP 200 and JPEG
  signatures from the local preview, including Unicode asset paths.
- Desktop DOM retained three burgers and showed no horizontal overflow
  at width 1168. The embedded pane initially reported a zero-size viewport;
  its lazy images never loaded during the preview despite a viewport attempt.
  CLI diagnostics passed, but full screenshot/mobile visual validation is
  blocked and is not claimed. Stopped the owned local preview server.
- Confirmed no changes to original image files, the Octoburger theme,
  starter/website configurations, or publishing workflows.
- User approved the follow-up PR, normal merge, and automatic deployment.
- Upstream master remains aligned with the branch base; publication pending.
