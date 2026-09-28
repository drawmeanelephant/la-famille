# Agent Readiness Waves

## Scope

Implement the requested repository-readiness punch list for Wave 1 and the
practical parts of Wave 2: ownership and contribution templates, repository
hygiene, local hooks, CI quality gates, test/build observability, agent
instructions, a devcontainer, flaky-test process, and sensitive-log
protection for the local Ask assistant.

SaaS-shaped signals explicitly excluded by the request remain out of scope:
distributed tracing, metrics backends, Sentry/error tracking, PagerDuty or
alerting, deploy dashboards, feature flags, product analytics, error-to-
insight automation, and progressive rollout.

## Potential static-output impact

None intended. Changes are limited to repository metadata, CI, developer
tooling, documentation, and Ask request logging. The static-site generator
and generated publish artifacts are unchanged.

## Validation

- Run all new shell checks directly.
- Run `go test ./...`, `go test -race ./...`, `go test -shuffle=on -count=2 ./...`,
  and `go vet ./...`.
- Run the build-duration and coverage gates locally.
- Validate the devcontainer JSON and review the complete diff.

## Status

Complete. Wave 1 and the practical Wave 2 items are implemented and
validated. The explicitly skipped SaaS-shaped signals remain out of scope.
