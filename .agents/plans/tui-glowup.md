# Task Plan: TUI Full Glow-Up — Cunty & Fun Edition

## Task ID
`tui-glowup` — covers the octoburger TUI polish thread (`468-tui-raoul-revival`,
`497-octoburger-core-theme`, `tui-audit-baseline`). User asked to make the TUI
"cunty and fun" (fabulous, animated, characterful).

## Library Verdict (user asked about ultraviolet / lipgloss / bubbletea)
- **ultraviolet**: low-level terminal primitives (cell renderer + input) that power
  Bubble Tea v2 / Lip Gloss v2. No tagged release exists; adds zero visual sparkle.
  **Skipped.**
- **lipgloss**: already at latest stable v1.1.0 — glow-up comes from *using more of
  it*, not upgrading.
- **bubbles** (new dep, v1.0.0): the missing sparkle — animated `spinner` and
  gradient `progress` components, compatible with bubbletea v1.3.10.

## Scope (agreed: full glow-up, items 1–5)
1. **Raoul glow-up** — multi-frame mascot (wink, squint, antenna sparkle) with
   rainbow color cycling; ✨confetti burst on successful build/serve.
2. **Spinner** during Build / RAG Export / Ask prep (bubbles `spinner`, hamburger
   spinner to match the octoburger brand).
3. **Gradient progress bar** for build phases (bubbles `progress`,
   hot-pink→violet scaled gradient) rendered alongside the existing
   `Phase: <name> (n/m)` line.
4. **Fun flavor text** per build phase, deterministic (no rand) so tests stay stable.
5. **Menu/badge polish** — ✦ cursor, cohesive pink/violet palette, pulse dots on
   serve/ask screens.

## Files
- `cmd/la-famille/tui_glow.go` (new): palette, mascot frames + rainbow tinting,
  confetti frames, flavor lines, pulse dots.
- `cmd/la-famille/tui_glow_test.go` (new): unit tests for all glow helpers.
- `cmd/la-famille/tui.go` (edit): wire spinner/progress/confetti/tick handling,
  menu cursor ✦, cohesive palette. No keybinding or lifecycle changes.
- `go.mod` / `go.sum`: add `github.com/charmbracelet/bubbles`.

## Invariants (guarded by existing tests — must not regress)
- Exact strings kept: `OCTOBURGER MENU`, `DASHBOARD STATUS`, `Cache Status: Hit/Miss`,
  `Phase: Writing assets and indexes (3/4)`, `Warning: N build errors reported`,
  `Error: <err>`, `Recovery Guidance: ...`, `Press Enter or Esc to return`,
  `http://127.0.0.1:<port>`, `Watch Mode: ENABLED (Live Reload active)`,
  `Watch Mode: DISABLED`, `Server Status: RUNNING`, `12 pages`,
  footer key hints (`d:`, `w:`, `?:`, `↑/k`, `Enter/Space`, `Press m to open`),
  help screen sections/spacing.
- Asserted substrings must stay contiguous (never split across two Render calls).
- Update/Init/lifecycle semantics unchanged (server/watcher tests keep passing).

## Static-Output Impact
None. TUI-only; no changes to generator/asset/theme pipeline.

## Tests & Verification
- New unit tests in `tui_glow_test.go` (deterministic frames, flavor mapping,
  confetti decay, rainbow tint, bar width clamping).
- `go test ./...`, `go vet ./...`, `gofmt` (format_check.sh).
- Manual: `go run ./cmd/la-famille tui` — build with confetti, serve with pulse dots.

## Status
- [x] Plan written
- [x] bubbles dependency added (v1.0.0 + harmonica; colorprofile/x-ansi/cellbuf bumped transitively)
- [x] tui_glow.go + tests
- [x] tui.go wiring
- [x] go test / go vet / gofmt green
- [ ] golangci-lint (local binary is go1.24-built, cannot lint go1.25 module — rerun in CI)
- [ ] Human sign-off
