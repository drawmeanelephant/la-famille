# Agent task plans

This directory contains task-scoped plans. It is the coordination layer for parallel agents.

## Rules

- Use one file per task: `.agents/plans/<task-id>.md`.
- Prefer a task ID based on the GitHub PR or issue number, followed by a short slug.
- Never reuse another agent's task-plan file.
- Record scope, ownership, dependencies, potential static-output impact, tests, and status.
- Update the task plan as the task progresses; do not use the root `plan.md` as a scratchpad.
- The root `plan.md` is reserved for the stable human-facing roadmap and planning policy.
