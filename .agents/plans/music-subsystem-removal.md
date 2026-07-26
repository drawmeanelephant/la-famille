# Music subsystem removal and documentation strategy

## Scope

Remove the abandoned soundtrack/music content, metadata, template presentation, agent workflow instructions, and music-only assets while preserving unrelated visual themes such as `synthwave`.

## Work items

- Remove `SoundtrackTheme` from content metadata, page models, generator wiring, templates, fixtures, and tests.
- Remove soundtrack navigation and music-specific copy from bundled layouts and project content.
- Remove `content/soundtrack/` and music-only mascot assets.
- Search for residual music/audio/soundtrack references and classify any intentional historical references before deleting them.
- Run `gofmt`, `go test ./...`, `go vet ./...`, and a clean site build.

## Documentation follow-up

- Add package-level GoDoc and exported API comments where they explain public behavior.
- Document build, parsing, rendering, TUI/watch, RAG/search, and GitHub workflows by package/flow rather than by individual file.
- Keep user-facing material in `content/docs/` and contributor/architecture material in repository developer docs.
- Verify documentation claims against tests and executable commands.

## Breaking-change assessment

Removing `soundtrack_theme` is a content frontmatter schema removal and may affect sites using that field. The generated output loses soundtrack-specific navigation and devlog display, but the static asset generation pipeline is otherwise unchanged.
