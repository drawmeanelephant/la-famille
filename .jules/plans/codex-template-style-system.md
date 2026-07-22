# Template style system

- Add a small shared stylesheet for focus, overflow, typography, and responsive media defaults.
- Link it from every bundled layout so themes share accessibility foundations without losing their visual identity.
- Add a static test that all layouts reference the stylesheet and define a viewport.
- Run the Go test, race, vet, and formatting checks.

Potential pipeline impact: one additional copied CSS asset in rendered sites; no generator semantics change.
