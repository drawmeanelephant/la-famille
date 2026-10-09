# Issue #659 — Serve screen footer advertises dead "w" key

## Task ID
`issue-659`

## Problem
The serve screen footer (`cmd/la-famille/tui.go` screenServe view) tells the
user to "Press w to toggle watch", but the key handler
(`cmd/la-famille/tui.go:401-405`) only honors `w` on menu/stats/diagnostics/help
screens. Pressing `w` on the serve screen is a no-op.

## Approach
The issue permits either making `w` work on the serve screen or stopping the
advertisement. I am removing the advertisement:

- The help screen already scopes `w` to "(menu/stats/diagnostics)", so the
  footer is the outlier.
- Making `w` toggle watch mid-serve honestly is not a small change: enabling it
  needs a watcher spawn plus a livereload endpoint that was never registered,
  and disabling it blocks in `http.Server.Shutdown` while SSE clients drain —
  reintroducing exactly the event-loop stall issue #662 removes. Restarting the
  whole serve flow for a footer key is disproportionate.
- Update the serve footer to drop "Press w to toggle watch" and update
  `TestTUIMenuFooterShowsAllKeys` expectations.

## Tests (red first)
- `cmd/la-famille/tui_test.go`: assert the serve screen footer no longer
  advertises a watch toggle key.

## Breaking changes to the static asset pipeline
None.
