# Plan: Add CLI flags unit test

1. Refactor `cmd/la-famille/main.go` to extract the `setupRootCmd(cfg config.Config) *cobra.Command` function to allow testing command flags.
2. Add `TestCommandFlags` in `cmd/la-famille/main_test.go` to verify the presence of `--content`, `--output`, `--template` flags on `buildCmd` and `--port`, `--watch` flags on `serveCmd`.
3. Verify changes through unit tests (`go test ./...`).
4. Ensure no breaking changes to static asset generation pipeline (none expected as this is mostly a refactoring to make the CLI testable).
