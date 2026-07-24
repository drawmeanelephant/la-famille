# TUI Baseline Audit Plan

## Task ID
`tui-audit-baseline`

## Objective
Perform a baseline TUI audit across standard site workflows, watch mode, and error scenarios (missing/invalid template, malformed content) without modifying themes or committing changes.

## Audit Scenarios
1. **Start TUI**: Launch TUI from repository root (`config.yaml`, `content/`, `templates/`).
2. **Normal Build**: Trigger "Build Site" option from menu.
3. **Stats & Diagnostics**: Open Stats dashboard and Diagnostics drawer (`d`).
4. **Serve Site**: Trigger "Serve Site" option, verify HTTP server on port 8080, exit via `q`/`Esc`.
5. **Watch Mode**: Toggle Watch mode, modify a content file, observe live rebuild, exit.
6. **Missing/Invalid Template Error (in temp copy)**: Remove template, run Serve in TUI, record behavior, guidance, server cleanup, menu return.
7. **Malformed Content Error (in temp copy)**: Corrupt frontmatter/markdown in content file, run Build/Serve in TUI, record behavior, guidance, server cleanup, menu return.

## Verification & Recording Method
- Inspect TUI model code (`cmd/la-famille/tui.go`) and execution outputs under temporary directories (`.tmp_tui_audit_*`).
- Use Go verification scripts / tests and terminal execution to validate screen transitions, error handling, server lifecycle, and recovery guidance.
- Document step-by-step findings in an Audit Report.
