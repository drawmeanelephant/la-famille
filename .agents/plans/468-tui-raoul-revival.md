# Task Plan: 468 TUI Raoul Revival — World-Class Octopus

## Task ID
`468-tui-raoul-revival` — covers `2. TUI Workflow Polish` `roadmap.md:29` keep mascot stable + milestone 2 `#468` serve states polish. The dancing octopus was orphaned when `initialModel:152` dropped `{"Just Raoul"}` (handler `tui.go:354` dead code, `screenRaoul:83` unreachable). Current `staticRaoul:986` / `animatedRaoul:994` are 5-line placeholders, not world-class.

## Objective
Restore and elevate Raoul(s) to world-class — dedicated `Just Raoul` screen reachable from menu, dancing on `screenServe:941` and `screenAsk:961`, with polished ASCII that looks intentional in `lipgloss` accent style, not stupid.

## Scope
- `cmd/la-famille/tui.go`:
  - Restore `{"Just Raoul"}` to `initialModel` choices (after `Help` or near `Stats`)
  - Replace `staticRaoul()` + `animatedRaoul(frame)` with larger, symmetrical, 8-arm octopus with expressive eyes and 2-frame dance (tentacle wave + eye blink), fits 38–42 col left panel and full-width `screenRaoul` view (`boxBorder.MaxWidth`)
  - Keep `tickMsg` 500ms loop `tui.go:473`, no new deps
- No change to generator/asset pipeline, no theme system expansion (deferred `roadmap.md:49`)
- Optional: keep `accentStyle` rendering, ensure `View()` wrapping respects `m.width`

Out of scope: image mascot re-render, changing `screenServe` logic, watcher/build cache.

## Static-Output Impact
None. TUI-only.

## Ownership
Agent + human visual approval. No parallel dependencies.

## Tests & Verification
- `go test ./...` + `go vet ./...`
- Manual: `go run ./cmd/la-famille tui` → `Just Raoul` shows 2-frame dance, `Serve Site` still dances, `Esc`/`q` returns to menu, narrow/wide layout not broken
- Add `tui_test.go` case that `initialModel` includes `Just Raoul` and `Update(enter)` on that cursor yields `screenRaoul`

## Status
- [x] Branch `468-tui-raoul-revival`
- [ ] Draft world-class art
- [ ] Re-add menu entry
- [ ] Tests + vet
- [ ] Human sign-off
