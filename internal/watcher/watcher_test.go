package watcher

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"content", "templates", "assets", "public"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	template := filepath.Join(root, "templates", "layout.html")
	if err := os.WriteFile(template, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.Config{
		ContentDir: filepath.Join(root, "content"),
		Template:   template,
		AssetDir:   filepath.Join(root, "assets"),
		OutputDir:  filepath.Join(root, "public"),
	}
}

func TestWatchCancellation(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- watch(ctx, cfg, nil, func(config.Config) (generator.BuildResult, error) { return generator.BuildResult{}, nil }, time.Millisecond)
	}()
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("watch returned %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch did not stop after cancellation")
	}
}

func TestWatchDebouncesAndTracksNewDirectories(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var builds atomic.Int32
	built := make(chan struct{}, 8)
	done := make(chan error, 1)
	debounce := 50 * time.Millisecond
	go func() {
		done <- watch(ctx, cfg, func(generator.BuildResult) { builds.Add(1); built <- struct{}{} }, func(config.Config) (generator.BuildResult, error) {
			return generator.BuildResult{}, nil
		}, debounce)
	}()
	time.Sleep(2 * debounce)
	nested := filepath.Join(cfg.AssetDir, "new-theme")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "theme.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-built:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not rebuild for a file in a newly-created directory")
	}
	time.Sleep(2 * debounce)
	// A burst of events should result in one debounced build, not one per event.
	for i := 0; i < 4; i++ {
		if err := os.WriteFile(filepath.Join(cfg.ContentDir, "page.md"), []byte(strings.Repeat("x", i+1)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-built:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not rebuild after content changes")
	}
	time.Sleep(2 * debounce)
	if got := builds.Load(); got != 2 {
		t.Fatalf("debounce produced %d builds for two change bursts, want 2", got)
	}
	cancel()
	<-done
}

func TestLiveReloadBroadcastAndDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/livereload", nil).WithContext(ctx)
	recorder := &syncResponseWriter{header: make(http.Header)}
	done := make(chan struct{})
	go func() { LiveReloadHandler(recorder, req); close(done) }()
	deadline := time.Now().Add(time.Second)
	for clientsSnapshot() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if clientsSnapshot() != 1 {
		t.Fatal("SSE client was not registered")
	}
	BroadcastReload()
	for !strings.Contains(recorder.bodyString(), "data: reload") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !strings.Contains(recorder.bodyString(), "data: reload") {
		t.Fatal("broadcast did not send an SSE reload event")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE handler did not exit after client disconnect")
	}
	if clientsSnapshot() != 0 {
		t.Fatal("disconnected SSE client remained registered")
	}
}

func TestWatchTracksNestedNewDirectories(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	built := make(chan struct{}, 8)
	done := make(chan error, 1)
	debounce := 50 * time.Millisecond
	go func() {
		done <- watch(ctx, cfg, func(generator.BuildResult) { built <- struct{}{} }, func(config.Config) (generator.BuildResult, error) {
			return generator.BuildResult{}, nil
		}, debounce)
	}()
	time.Sleep(2 * debounce)
	nestedDir := filepath.Join(cfg.AssetDir, "sub1", "sub2")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	select {
	case <-built:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not rebuild for newly created nested directory")
	}
	time.Sleep(2 * debounce)
	file := filepath.Join(nestedDir, "style.css")
	if err := os.WriteFile(file, []byte("h1{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-built:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not rebuild for file created inside nested directory")
	}
	cancel()
	<-done
}

type syncResponseWriter struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
}

func (w *syncResponseWriter) Header() http.Header { return w.header }
func (w *syncResponseWriter) WriteHeader(int)     {}
func (w *syncResponseWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}
func (w *syncResponseWriter) Flush() {}
func (w *syncResponseWriter) bodyString() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

func clientsSnapshot() int {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	return len(clients)
}
