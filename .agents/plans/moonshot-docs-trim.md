# Card 1: Moonshot docs trim

## Scope

Replace the detailed Moonshot 1–5 proposal sections in `docs/MOONSHOTS.md`
with one-line pointers to issues #579–#583, and repoint the scorecard's five
links to those issues. Preserve all other document content.

## Validation

- Review the diff to confirm only the scorecard links and sections 1–5 changed,
  apart from this task plan.
- Run the repository CI-equivalent local Go checks required by `AGENTS.md`.

## Static-output pipeline impact

None. This is a documentation-only change.
