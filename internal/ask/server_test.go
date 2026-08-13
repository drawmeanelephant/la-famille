package ask

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

// fixtureCorpus writes a minimal RAG archive into dir.
func fixtureCorpus(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle := "<file path=\"content/docs/faq.md\">\n<content>\n---\ntitle: FAQ\n---\n# FAQ\n\n## Install\n\nRun `go install`.\n\n## Where is the RAG archive?\n\nIt is in the rag-archive folder.\n</content>\n</file>\n"
	if err := os.WriteFile(filepath.Join(dir, "rag-content.md"), []byte(bundle), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rag-system.md"), []byte("# System\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rag-config.md"), []byte("# Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestConfigValidateLoopbackOnly(t *testing.T) {
	c := Config{Host: "0.0.0.0", Port: PortDefault, LoopbackOnly: true}
	if err := c.Validate(); err == nil {
		t.Fatalf("Validate should reject 0.0.0.0 when LoopbackOnly=true")
	}
	c2 := Config{Host: "127.0.0.1", Port: PortDefault, LoopbackOnly: true}
	if err := c2.Validate(); err != nil {
		t.Fatalf("loopback should pass: %v", err)
	}
	c3 := Config{Host: "127.0.0.1", Port: 70000, LoopbackOnly: true}
	if err := c3.Validate(); err == nil {
		t.Fatalf("out-of-range port must fail")
	}
}

func TestConfigDefaults(t *testing.T) {
	c := Config{}
	c.Defaults()
	if c.Host != "127.0.0.1" {
		t.Errorf("default host: %s", c.Host)
	}
	if c.Port != PortDefault {
		t.Errorf("default port: %d", c.Port)
	}
	if c.ProviderName != "ollama" {
		t.Errorf("default provider: %s", c.ProviderName)
	}
	if c.MaxContext == 0 {
		t.Errorf("MaxContext should default nonzero")
	}
}

func TestNewServerUnknownProvider(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	_, err := NewServer(Config{ProviderName: "openai", RagDir: corpus, LoopbackOnly: true})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("expected unknown provider error, got %v", err)
	}
}

func TestNewServerMissingRAGDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := NewServer(Config{RagDir: missing, LoopbackOnly: true})
	if err == nil {
		t.Fatalf("missing RagDir should error")
	}
}

func TestNewServerLoadsFakeProvider(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{ProviderName: "fake", RagDir: corpus, LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if srv == nil || srv.provider == nil {
		t.Fatal("server/provider nil")
	}
	if _, ok := srv.provider.(*llm.FakeProvider); !ok {
		t.Errorf("provider not *llm.FakeProvider: %T", srv.provider)
	}
}

// citeOnlyFaker implements llm.Provider for the ask server tests. It
// emits a single paragraph with a verifiable [1] key.
type citeOnlyFaker struct{}

func (citeOnlyFaker) Name() string                    { return "fake-cite" }
func (citeOnlyFaker) Available(context.Context) error { return nil }
func (citeOnlyFaker) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	key := "1"
	if len(req.Citations) > 0 {
		key = req.Citations[0].Key
	}
	body := fmt.Sprintf("According to [%s], yes — the answer is yes.\n\n— deterministic test response.", key)
	return llm.Response{Answer: body, Markdown: body}, nil
}

func TestServerAnswerWithFakeProviderCited(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{ProviderName: "fake", RagDir: corpus, LoopbackOnly: true, Model: "fake"})
	if err != nil {
		t.Fatal(err)
	}
	srv.provider = citeOnlyFaker{}
	resp, err := srv.Answer(context.Background(), AnswerRequest{Question: "Where is the RAG archive?"})
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if resp.Status != "answered" {
		t.Errorf("status=%q", resp.Status)
	}
	if len(resp.Sources) == 0 {
		t.Errorf("expected at least one verified source")
	}
	if resp.Diagnostics.ChunksRetrieved == 0 {
		t.Errorf("ChunksRetrieved=0")
	}
}

func TestServerAnswerNoCorpusHit(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{ProviderName: "fake", RagDir: corpus, LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.Answer(context.Background(), AnswerRequest{Question: "completely off-topic xyzqq bun"})
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if !resp.NoAnswer {
		t.Errorf("expected NoAnswer=true, got %+v", resp)
	}
	if resp.Diagnostics.ChunksRetrieved != 0 {
		t.Errorf("ChunksRetrieved=%d", resp.Diagnostics.ChunksRetrieved)
	}
}

func TestServerRejectUnknownQuestion(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{ProviderName: "fake", RagDir: corpus, LoopbackOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.Answer(context.Background(), AnswerRequest{Question: "   "}); err == nil {
		t.Fatalf("blank question must fail")
	}
}

func TestHandleAskRejectsStreamingRequest(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{
		ProviderName: "fake",
		RagDir:       corpus,
		Host:         "127.0.0.1",
		Port:         0, // unused because we hit the handler directly
		DisableUI:    true,
		LoopbackOnly: true,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	body := strings.NewReader(`{"question":"hello","stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	w := httptest.NewRecorder()
	srv.handleAsk(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("stream=true should return 400, got %d (body=%q)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "streaming") {
		t.Errorf("expected rejection message to mention streaming, got %q", w.Body.String())
	}
}

func TestServerStartAndStop(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(Config{
		ProviderName: "fake",
		RagDir:       corpus,
		Host:         "127.0.0.1",
		Port:         port,
		LoopbackOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("server never opened port: %v", err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Start returned non-cancel error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("server did not shut down")
	}
}

func TestServerHTTPStatusJSON(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(Config{
		ProviderName: "fake",
		RagDir:       corpus,
		Host:         "127.0.0.1",
		Port:         port,
		LoopbackOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("server never opened port: %v", err)
	}
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/status", port))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var st Status
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	if st.Provider != "fake" {
		t.Errorf("provider=%q", st.Provider)
	}
	if st.ChunkCount == 0 {
		t.Errorf("ChunkCount=0")
	}
	if !st.LoopbackOnly {
		t.Errorf("LoopbackOnly=false on response")
	}
}

func TestServerHTTPAskHappyPath(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewServer(Config{
		ProviderName: "fake", RagDir: corpus, Host: "127.0.0.1", Port: port, LoopbackOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.provider = citeOnlyFaker{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("server never opened port: %v", err)
	}
	body := strings.NewReader(`{"question":"Where is the RAG archive?"}`)
	res, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/ask", port), "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status: %d body=%s", res.StatusCode, raw)
	}
	raw, _ := io.ReadAll(res.Body)
	var ans AnswerResponse
	if err := json.Unmarshal(raw, &ans); err != nil {
		t.Fatalf("decode: %v body=%s", err, raw)
	}
	if ans.Status != "answered" {
		t.Errorf("status=%q", ans.Status)
	}
	if len(ans.Sources) == 0 {
		t.Errorf("expected sources")
	}
}

func TestServerHTTPAskMethodNotAllowed(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	port, _ := freePort()
	srv, _ := NewServer(Config{
		ProviderName: "fake", RagDir: corpus, Host: "127.0.0.1", Port: port, LoopbackOnly: true,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("server never opened port: %v", err)
	}
	res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/ask", port))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/ask should be 405, got %d", res.StatusCode)
	}
}

func TestServerHTTPLargeBodyRejected(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	port, _ := freePort()
	srv, _ := NewServer(Config{
		ProviderName: "fake", RagDir: corpus, Host: "127.0.0.1", Port: port, LoopbackOnly: true,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("server never opened port: %v", err)
	}
	huge := strings.Repeat("a", 64<<10)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/api/ask", port),
		bytes.NewBufferString(`{"question":"`+huge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode/100 != 4 {
		t.Errorf("oversized body should be rejected, got %d", res.StatusCode)
	}
}

func TestServerDefaultsToBindLoopback(t *testing.T) {
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{
		ProviderName: "fake", RagDir: corpus, Port: 0, LoopbackOnly: true,
		// No Host set — Defaults() must fill 127.0.0.1.
	})
	if err != nil {
		t.Fatal(err)
	}
	if srv.cfg.Host != "127.0.0.1" {
		t.Errorf("defaults did not set Host to loopback: %s", srv.cfg.Host)
	}
}

// --- AI answer server: generated-site serving ---
//
// Citation links in answers ("Open source") land on generated-site paths, so
// handleIndex must serve the built site in OutputDir in addition to the Ask
// UI. These tests exercise handleIndex through httptest recorders.

func newFixtureServer(t *testing.T, outputDir string) *Server {
	t.Helper()
	corpus := t.TempDir()
	fixtureCorpus(t, corpus)
	srv, err := NewServer(Config{
		ProviderName: "fake",
		RagDir:       corpus,
		OutputDir:    outputDir,
		Host:         "127.0.0.1",
		LoopbackOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func serveIndex(s *Server, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	s.handleIndex(w, r)
	return w
}

func TestHandleIndexServesGeneratedPage(t *testing.T) {
	outputDir := t.TempDir()
	pageDir := filepath.Join(outputDir, "handbook")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pageDir, "index.html"), []byte("<h1>Handbook</h1>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "feed.xml"), []byte("<rss/>"), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := newFixtureServer(t, outputDir)

	t.Run("page directory resolves to its index.html", func(t *testing.T) {
		w := serveIndex(srv, "/handbook/")
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want 200 (body %q)", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "<h1>Handbook</h1>") {
			t.Errorf("body %q, want the generated page", w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("Content-Type %q, want text/html", ct)
		}
	})

	t.Run("plain file is served", func(t *testing.T) {
		w := serveIndex(srv, "/feed.xml")
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "<rss/>") {
			t.Errorf("body %q, want the feed file", w.Body.String())
		}
	})

	t.Run("unknown path falls through to the UI", func(t *testing.T) {
		w := serveIndex(srv, "/handbook/does-not-exist/")
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want the UI fallback", w.Code)
		}
		if !strings.Contains(w.Body.String(), "<title>") {
			t.Errorf("body %q, want the Ask UI shell", w.Body.String())
		}
	})

	t.Run("directory without index.html is not listed", func(t *testing.T) {
		empty := filepath.Join(outputDir, "empty")
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatal(err)
		}
		w := serveIndex(srv, "/empty/")
		if strings.Contains(w.Body.String(), "index of") || w.Code == http.StatusOK && strings.Contains(w.Body.String(), "<pre>") {
			t.Errorf("status %d body %q, want no directory listing", w.Code, w.Body.String())
		}
	})

	t.Run("embedded UI assets still win", func(t *testing.T) {
		w := serveIndex(srv, "/app.js")
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want the embedded UI asset", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
			t.Errorf("Content-Type %q, want the asset's type", ct)
		}
	})

	t.Run("root always serves the UI, not the site", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("<h1>Generated Home</h1>"), 0o600); err != nil {
			t.Fatal(err)
		}
		w := serveIndex(srv, "/")
		if strings.Contains(w.Body.String(), "Generated Home") {
			t.Errorf("root body %q, want the Ask UI shell, not the generated site", w.Body.String())
		}
	})
}

func TestHandleIndexBypassesMissingOutputDir(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "never-built")
	srv := newFixtureServer(t, outputDir)

	w := serveIndex(srv, "/handbook/")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want UI fallback when OutputDir is absent", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<title>") {
		t.Errorf("body %q, want the Ask UI shell", w.Body.String())
	}
}

// --- helpers ---

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForServer(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	reqURL := fmt.Sprintf("http://127.0.0.1:%d/api/status", port)
	for time.Now().Before(deadline) {
		res, err := http.Get(reqURL) //nolint:gosec // test helper targeting local test server
		if err == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return errors.New("server did not become ready in time")
}

// silence "imported and not used" when this file grows.
var _ retrieval.Corpus = retrieval.Corpus{}
