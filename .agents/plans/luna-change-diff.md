# Task plan: `luna/change-diff` (#580.4)

1. Compare canonical site manifests in a new package, reporting page and
   metadata changes, taxonomy and link deltas, graph edges, orphan transitions,
   and newly broken links.
2. Add `la-famille diff <a> <b>` with output-directory, manifest, and Git-ref
   inputs, plus readable and JSON output.
3. Add golden CLI fixtures covering a renamed/edited page, changed links and
   graph edges, and a new broken link while preserving an unrelated orphan.
4. Document the command and its input forms.

**Static-output/CLI impact:** No generated files or build/cache behavior
changes. This adds a read-only CLI command and documents its contract.
