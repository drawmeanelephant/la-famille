1. **Create `internal/search` package**: Add `search.go` and `search_test.go` to handle stripping markdown and serializing search index objects into minified JSON.
2. **Update generator pipeline**: Inject `searchIndex` generation inside `internal/generator/generator.go` so it collects `search.SearchItem` during file loop execution and writes them out to `search.json`.
3. **Verify functionality**: Ensure tests pass and the behavior handles properly using standard JSON unmarshal comparisons.
4. **Complete pre-commit steps:** Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

Breaking Changes:
- None. `search.json` is a new file addition into the pipeline output.
