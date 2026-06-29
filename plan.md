1. **Add WatchMode to Config:** Update `internal/config/config.go` to include a `WatchMode` boolean field so that other packages know when to inject livereload scripts.
2. **Implement Livereload SSE Server:** Create `internal/watcher/livereload.go` using only the standard library (`net/http`, `sync`) with a `LiveReloadHandler` and `BroadcastReload()` function.
3. **Trigger Reload on Build:** Modify `internal/watcher/watcher.go` to call `BroadcastReload()` right after a successful `generator.Build()`.
4. **Inject JS Payload:** Modify `internal/render/render.go` so that if `cfg.WatchMode` is true, it writes the template output into a buffer, replacing `</body>` with an SSE event listener script + `</body>`.
5. **Update Server Setup:** Modify `cmd/la-famille/main.go` and `cmd/la-famille/tui.go` to use an `http.ServeMux` to handle both `/` and `/livereload`, trigger an initial build so files contain the injected JS payload, and set `cfg.WatchMode = true`.
6. **Testing and Verification:** Run `go test ./...` and `go vet ./...` to verify everything works correctly. Also, make sure no asset pipelines are broken.
7. **Complete pre-commit steps:** Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.
8. **Execution Log:** Write a standard execution log to `content/jules/reports/livereload-setup.md` as requested.
