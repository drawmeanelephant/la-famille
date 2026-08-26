# Dogfood/UAT bug triage and spec — issues #506–#518

Branch: `t3code/dogfood-bug-triage-and-spec`
Source: UAT findings against v0.1.0-prealpha artifacts plus the #500 black-box pass.
All findings re-verified by hand against current `master` (build from this branch) before speccing.

## Verdicts after reproduction on master

| Issue | Live? | Spec decision |
|---|---|---|
| #506 check ignores broken internal links without `.md` suffix | YES | Fix: checker resolves extension-less internal links against expected output URLs |
| #507 layout-terminal refs `/assets/img/jules-logo.png`, `/docs/index.html` that never deploy | YES | Fix: embed + install missing mascot images; remove dead Docs nav link |
| #508 `new` accepts unsafe slugs verbatim | YES | Fix: normalize slug/filename in `new`, print a notice |
| #509 check has no slug-safety warning | YES | Fix: warn on URL-unsafe filename-derived slugs |
| #510 publish-check --json prints plain text on failure | YES | Fix: structured JSON error object on stdout, exit 1 |
| #512 check summary uses ✓ with warnings present | YES | Fix: ⚠ when warnings > 0, ✓ only when clean |
| #513 init logs "Created default template" regardless of theme | YES | Fix: theme-aware log message/path |
| #515 check --asset-health no-op for template-referenced assets | PARTIAL | Flag works for content refs; templates are not scanned. Fix: scan installed templates under --asset-health |
| #516 publish-check passes artifacts containing Missing Page stubs | YES | Fix: report stub pages as warnings; hard-fail behind `--strict` |
| #517 init scaffold fails first-run check (missing descriptions) | YES | Fix: add `description:` to scaffolded frontmatter |
| #511 new "Next steps" hint uses --content abs-path | YES | Fix: project-root-consistent, CWD-relative hints |
| #514 RELEASE-QUICKSTART.md lacks SHA256 example | YES | Docs fix |
| #518 No Pages-deploy guidance for binary-only installs | YES | Docs fix |

## Static-output impact (guardrail disclosure)

- #507 changes what `init`/`build` deploy: two additional embedded images
  (`img/jules-logo.png`, `img/mascot-electric-blue.jpeg`) are added to the
  runtime asset packet; `templates/layout-terminal.html` loses its Docs nav
  button. Sites referencing those assets keep working; nothing existing is
  removed from output.
- #508 changes filenames `new` creates (normalized). Existing content is never
  touched.
- #516 adds `--strict` to publish-check; default behavior gains warnings only.
- Everything else is diagnostics/docs only; no output bytes change.

## Implementation order

1. `internal/checker`: extension-less internal-link resolution (#506), slug-safety warning (#509), template asset scan behind CheckAssetHealth (#515).
2. `cmd/la-famille/new.go`: slug normalization + notice (#508), Next-steps hints (#511).
3. `internal/publisher` + `publish_check.go`: stub findings + `--strict` (#516), JSON error path (#510).
4. `check.go` summary symbol (#512); `main.go` init log (#513); scaffold descriptions (#517).
5. Runtime assets/templates (#507): embed missing images, fix layout-terminal.
6. Docs (#514, #518).

## Tests

Same-package unit tests per fix:
- checker: broken extension-less link flagged; safe link not flagged; taxonomy links (`/tags/x/`) not false-positived; unsafe-slug filename warns; template asset ref warns under flag only.
- new: unsafe slug normalized with notice; safe slug untouched; hint text uses `--project-root`.
- publisher: stub page reported; `--strict` fails; clean artifact unaffected.
- publish_check cmd: JSON object on stdout for validation failure, exit non-zero.
- check.go: summary symbol ⚠ with warnings, ✓ clean, ✗ errors.
- runtimeassets: every bundled layout's local `/assets/img/*` reference exists in the embedded packet (contract test preventing #507 regressions).

Validation gate: `go test ./... && go vet ./...`.

## Handoff

Complete on this branch. All 13 findings addressed; every fix carries
same-package unit tests (`internal/checker/dogfood_issues_test.go`,
`cmd/la-famille/dogfood_issues_test.go`, plus updated contracts in
`check_test.go`, `new_test.go`, `scaffold_test.go`,
`internal/runtimeassets/assets_test.go` and the release-smoke manifest).

Validation: `go test ./...` (30 packages, 0 failures) and `go vet ./...` clean;
gofmt clean on all touched files. End-to-end repro of each issue re-run against
a freshly built binary — all confirmed fixed. `golangci-lint` could not run in
this environment (toolchain version mismatch building the linter itself, not a
code finding).

Intentional static-output change: released artifacts now also deploy
`assets/img/jules-logo.png`; the release-smoke frozen manifest was updated to
match.
