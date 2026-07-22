# Codex Plan: Template Contract Harness

## Goal
Build a reusable regression harness for every bundled La Famille layout/theme that renders a representative Page fixture and verifies layout contract rules without modifying generator semantics or adding external dependencies.

## Scope & Target Files
- `.jules/plans/codex-template-contract-harness.md`: Plan file for tracking implementation steps.
- `internal/render/template_contract_harness_test.go`: Comprehensive regression test harness verifying layout contracts for all bundled templates in `templates/`.

## Contract Rules & Assertions
For each bundled template found in `templates/*.html`:
1. **HTML Structure & Landmarks**: `<!DOCTYPE html>`, `html lang`, `<head>`, `<body>`, `<main id="main-content">`, `<nav id="site-navigation" aria-label="...">` (or equivalent landmark), `<a href="#main-content">` (skip link).
2. **Title**: Exactly one `<title>` element with meaningful content and exactly one `<h1>` element on the page.
3. **Viewport Metadata**: `<meta name="viewport" ...>` tag present.
4. **Navigation/Menu Targets**: Skip link (`href="#main-content"`) targets existing element with `id="main-content"`; site links and navbar targets resolve properly.
5. **Canonical & og:url**:
   - When `Page.CanonicalURL` is configured, `<link rel="canonical" href="...">` and `<meta property="og:url" content="...">` appear.
   - When `Page.CanonicalURL` is empty, neither tag appears.
6. **Stylesheet & Asset References**: Theme/foundations CSS (`/assets/css/theme-foundations.css` or theme stylesheets) and required asset scripts exist.
7. **Heading Hierarchy**: Exactly one `<h1>`, with subsequent headings (`<h2>`, `<h3>`, etc.) following a sane hierarchy without skipped levels.
8. **Visible Focus Styles & Accessible Labels**: Focus indicators (`:focus`, `:focus-visible`, or focus utility classes) and accessible labels (`aria-label`, `alt` tags on images, `aria-hidden` rules) are intact.
9. **Emoji Kitchen Output**: Representative content with Emoji Kitchen HTML structures (e.g., `<img class="emoji-kitchen" ...>`) renders intact without tag corruption or escaping issues.
10. **Panic-free Optional Fields**: Rendering a zero-value / empty `page.Page{}` produces no template panics or evaluation errors.

## Verification & Execution
- `gofmt -w internal/render/template_contract_harness_test.go`
- `go test -count=1 ./...`
- `go test -race ./...`
- `go vet ./...`
