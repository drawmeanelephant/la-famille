# Restore standalone TUI mascot screen

## Scope

Restore the `Just Raoul` menu entry and upgrade the preserved mascot into a real multi-frame dancing ASCII octopus so users can enjoy it without starting the server.

## Work items

- Add `Just Raoul` to the current TUI command menu.
- Replace the two-frame blink/wiggle with a stable-width multi-frame animation.
- Add focused regression tests proving the menu selection enters `screenRaoul`, schedules animation ticks, and exposes distinct animation frames.
- Run formatting, package tests, and vet.

## Breaking-change assessment

None. This restores and improves an interactive TUI capability and does not affect generated site output.
