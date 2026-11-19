# Graph explorer: Focus toggle after search pick, unlinked-node layout

## Task ID
`1795051432-graph-focus-layout` (branch `devin/1795051432-graph-focus-layout`)

## Objective
Fix two Knowledge Graph Explorer (`/graph/`) bugs found during E2E testing:

1. Picking a search suggestion selects the node but leaves the
   "Focus selection" button disabled. `selectNode()` only re-rendered the
   controls when the graph was suppressed or focus mode was already on.
2. Pages with no links pile up against the canvas edges with overlapping,
   clipped labels. The layout clamped node centres 8px inside the canvas
   (ignoring label width), unlinked nodes had no forces to settle them, and
   the SVG had no viewBox until the first window resize.

Cosmetic items (plain 404, duplicate tag-page heading, tag links not counted
as inbound) are out of scope.

## Changes
- `assets/graph/explorer.js`
  - `selectNode()` calls `renderControls()` on every path.
  - Unlinked nodes (no inbound/outbound) skip the force simulation and are
    placed as a label-sized grid below the linked cluster (`placeUnlinked`).
  - Node positions are no longer clamped to the canvas; `fitView()` fits the
    viewBox to the visible nodes and their labels (zoom-out only, above the
    legend overlay).
  - The canvas is measured before seeding the layout, and a `ResizeObserver`
    re-fits when the detail panel narrows the canvas.
- `internal/graphexplorer/graphexplorer_test.go`: regression assertions on
  the shipped JS.

## Static asset pipeline impact
None. Only the contents of the shipped `assets/graph/explorer.js` change; the
payload (`data.json`), template, and copy step are untouched.

## Validation
- `go test ./...`, `go vet ./...`, `./format_check.sh`
- Manual browser check of both flows on `la-famille serve`.
