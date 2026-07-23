# Plan: TUI Workflow Polish

## Goal
Make build, serve, watch, diagnostics, and failure states easier to understand in the TUI without redesigning the current visual identity.

## Proposed Changes
1. **Failure Messages & Recovery Guidance (`cmd/la-famille/tui.go`)**:
   - Provide actionable recovery instructions for build, server, and content errors (e.g. port conflict guidance, template/syntax error guidance).
   - Display clear status messages and next-step actions in `screenWorking`, `screenServe`, and `screenDiagnostics`.

2. **Status Transitions & Server/Watch Modes (`cmd/la-famille/tui.go`)**:
   - Improve `screenServe` to show active server URL, Watch mode status, Live Reload indicator, and exit guidance.
   - Update `renderStatusPanel` dashboard display to cleanly show watch mode, server status, build phase, cache status, and diagnostics warnings with next-step tips.

3. **Keyboard Help & Command Discoverability (`cmd/la-famille/tui.go`)**:
   - Standardize key help prompts across all active screens (`screenMenu`, `screenWorking`, `screenServe`, `screenStats`, `screenDiagnostics`, `screenHelp`, `screenRaoul`).
   - Polish `screenHelp` to present categorized, easy-to-read keyboard shortcuts.

4. **Diagnostics & Content Health Next Steps (`cmd/la-famille/tui.go`)**:
   - Add content health next-step suggestions in `screenStats` (guidance for missing descriptions, orphaned pages, missing dates).
   - Add actionable guidance per diagnostic item in `screenDiagnostics`.

5. **Unit Tests (`cmd/la-famille/tui_test.go`)**:
   - Add/update unit tests for failure recovery guidance, serve view details, content health recommendations, keyboard help rendering, and navigation return paths.

## Potential Breaking Changes
- None. Static site generation semantics and static output files are completely unaffected.

## Static Output Impact
- Static output is **explicitly unaffected** as all changes are contained within `cmd/la-famille/tui.go` and `cmd/la-famille/tui_test.go`.
