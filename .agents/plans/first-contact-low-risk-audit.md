# Audit First-Contact Low-Risk Issues Implementation Plan

## Task ID
`first-contact-low-risk-audit`

## Objective
Audit and fix low-risk first-contact issues in La Famille:
1. **Go toolchain documentation:** Document required Go toolchain version (Go 1.24+ / go1.24.3) and installation path in `README.md` and `content/docs/setup.md`.
2. **README usability:** Ensure `README.md` is complete and usable for someone who did not clone the source repository or needs to understand usage outside raw repo clone context.
3. **Ask UI accessibility audit & fix:** Fix accessibility-label / ARIA mismatches in `internal/ask/ui/index.html` (specifically `<span class="ask-badge">` where `aria-label` was used on a non-interactive span, and `<button id="diagnostics-toggle">` / status / aria-label usages).
4. **Focused regression tests:** Add UI / asset tests for `internal/ask` verifying the accessibility labels and markup contract.
5. **Local validation:** Ensure `go test ./...` and `go vet ./...` pass.

## Proposed Changes

### Documentation
#### [MODIFY] [README.md](file:///Users/tbuddy/Documents/antigravity/la-famille/README.md)
- Clarify Go toolchain version requirement (Go 1.24.0+ / toolchain `go1.24.3`).
- Document installation paths, go version checks, and clarity for non-clone / cloned usage options.

#### [MODIFY] [content/docs/setup.md](file:///Users/tbuddy/Documents/antigravity/la-famille/content/docs/setup.md)
- Detail Go 1.24+ requirements and installation path verification (`go env GOPATH`, `export PATH=$PATH:$(go env GOPATH)/bin`).

### Ask UI
#### [MODIFY] [internal/ask/ui/index.html](file:///Users/tbuddy/Documents/antigravity/la-famille/internal/ask/ui/index.html)
- Audit ARIA attributes: Fix badge aria-label or title, fix any interactive/non-interactive accessibility label mismatches.

#### [NEW] [internal/ask/ui_test.go](file:///Users/tbuddy/Documents/antigravity/la-famille/internal/ask/ui_test.go)
- Add focused regression test verifying Ask UI HTML embedded asset contains expected accessible markup, labels, and form associations.

## Verification Plan
- `go test ./internal/ask/...`
- `go test ./...`
- `go vet ./...`
