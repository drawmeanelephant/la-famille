# Task Plan: 469 TUI Keyboard Help & Command Discoverability

## Task ID
`469-tui-help` — GitHub Issue #469, Milestone 2 TUI Workflow Polish
Closes: https://github.com/drawmeanelephant/la-famille/issues/469

## Objective
New operator discovers keys without reading code — improve keyboard help and command discoverability in `cmd/la-famille/tui.go` without expanding visual theme system (roadmap.md:29 deferred).

## Context
- Current `tui.go:699-735` View footer for screenMenu shows subset `↑/k, ↓/j: Navigate • Enter: Select • m: Toggle menu • d: Diagnostics • q: Quit` but omits `w` watch toggle and `?`/`h` help. Other screens show inconsistent subsets (`screenStats:840`, `screenWorking:931`, `screenServe:951`, etc.).
- `tui.go:288-472` Update handles `d` globally, `m` on menu, but has no bindings for `w`, `?`, `h`. Issue requires `?` or `h` to open one-line legend (or footer expands) and `w` to toggle watch mode.
- `tui.go:704-713` selected menu item uses `Foreground("212").Bold(true)` only — lacks `focus-visible` a11y state. Reference `templates/layout.html:21-22,32` which uses `focus-visible:outline`, `focus-visible:ring-2`, `focus-visible:outline-primary`.
- Docs out of sync: `README.md:93` lists only `up/down, j/k, enter/space, q/esc` and Active Server Views `q/Esc`. `content/docs/tui.md:24-30,95` similarly omits `d,w,?,h`. `content/docs/cli.md:45-55` covers `tui` command but not keybindings.

## Scope

### In scope
- `cmd/la-famille/tui.go`:
  - Add key handling in `Update` for `w` (toggle watch mode, anywhere on menu), `?` and `h`/`H` (toggle Help screen). Help should be reachable from any screen via `?`/`h` and close via `?`, `h`, `q`, or `Esc` (return to previous screen via `helpReturn` or `diagnosticsReturn` pattern).
  - Expand footers on every screen to show its valid keys:
    - screenMenu open: `↑/k ↓/j: Navigate • Enter/Space: Select • m: Menu • d: Diagnostics • w: Watch • ?: Help • q: Quit`
    - screenMenu closed: `Menu closed. Press m to open • d: Diagnostics • w: Watch • ?: Help • q: Quit`
    - screenDiagnostics: `↑/↓ Navigate • c: Clear • d/Esc/q: Return • ?: Help`
    - screenStats, screenWorking, screenServe, screenAsk, screenRaoul: `d: Diagnostics • ?: Help • Esc/q: Back` (+ `w: Watch` where toggling watch is valid)
  - Ensure every screen’s footer includes its valid keys (acceptance: Every screen shows its valid keys).
  - Implement `helpReturn screen` field similar to `diagnosticsReturn` to return from Help to prior screen; `?`/`h` toggles.
  - Enhance menu selection style for focus-visible: selected item uses `Bold + Underline + Background(235)` or `Border` to mirror `focus-visible:outline` pattern from `templates/layout.html`. Keep mascot/visual language stable — no new theme system.
- `content/docs/tui.md`: Add Keybindings table/section documenting `j/k, up/down, enter/space, q/esc, d, w, ?, h, m, c` and per-screen footers.
- `content/docs/cli.md`: Expand `tui` section to reference keybindings and `d/?/w` shortcuts.
- `README.md:93-98` TUI Navigation & Controls: add bullets for `d` diagnostics drawer, `w` watch toggle, `?`/`h` help, and `m` menu toggle + `c` clear in diagnostics.

### Out of scope
- New visual theme system, mascot theme changes, alternate animations (roadmap.md:49 deferred).
- Changing generator, watcher, or build cache logic.
- HTML template redesign beyond focus-visible pattern reference.

## Static-Output Impact
None — TUI-only. No generated HTML/asset changes except docs rendering of updated markdown.

## Breaking Changes
None — additive keybindings; existing keys (`j/k`, `q/esc`, `m`, `d`, `c`) unchanged.

## Tests & Verification
- Add `cmd/la-famille/tui_test.go` cases:
  - `TestTUIHelpToggleViaQuestionAndH` — `?` and `h` from menu → `screenHelp`, second press returns.
  - `TestTUIWatchToggleViaWKey` — `w` from menu toggles `cfg.WatchMode` and updates `workMsg`.
  - `TestTUIMenuFooterShowsAllKeys` — `View()` on each screen contains its expected footer substrings (`d`, `w`, `?`, `q` etc.).
  - `TestTUIFocusVisibleSelectedStyle` — selected menu cursor renders with focus-visible attributes (or verify View contains selected marker with distinct style).
- Existing tests must still pass (`TestTUICommandMenuOpenNavigationAndEscape`, etc.).
- `go test ./...`, `go vet ./...`, manual `go run ./cmd/la-famille tui` navigation check: each screen footer visible, `?` opens help, `w` toggles watch, focus-visible on menu list.

## Steps
1. Add `helpReturn screen` field and focus-visible style to `tui.go`.
2. Implement `w`, `?`, `h` key handlers + footer expansions for all screens.
3. Update docs: `README.md`, `content/docs/tui.md`, `content/docs/cli.md`.
4. Add/adjust unit tests per above.
5. Validate `go test ./... && go vet ./...`.

## Status
- [x] Plan created
- [x] Keybindings implemented
- [x] Footers updated (every screen)
- [x] Focus-visible style applied
- [x] Docs synced
- [x] Tests + vet passing
