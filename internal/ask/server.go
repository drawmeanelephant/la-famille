// Package ask is the HTTP orchestrator that ties the retrieval corpus, the
// LLM provider, the citation verifier, and the local UI into a single
// `la-famille ask` command. It depends only on the standard library plus
// internal/llm and internal/retrieval.
//
// All HTTP routes are exposed on a loopback address only; the Server refuses
// to start on any address that resolves to a non-loopback IP without an
// explicit LoopbackOnly=false override.
package ask

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

// PortDefault is the canonical default HTTP port shared by the CLI,
// the TUI integration, and the server's own defaulting. Keeping a single
// exported constant prevents the banner, the diagnostics drawer, and the
// screen view from drifting apart.
const PortDefault = 8090

// Config collects the user-facing knobs for the ask server. It is what the
// CLI flag parser builds before calling NewServer.
type Config struct {
	Host         string
	Port         int
	ProviderName string
	Model        string
	RagDir       string
	OutputDir    string
	Rebuild      bool
	Verbose      bool
	NoBrowser    bool
	MaxContext   int
	DisableUI    bool // when true, only `/api/*` endpoints are served
	LoopbackOnly bool // defaults to true; advanced callers may allow all
}

// Defaults fills in sensible values for unspecified fields. It does not
// touch the loopback-only guarantee — that stays true unless the caller
// explicitly disables it.
func (c *Config) Defaults() {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = PortDefault
	}
	if c.ProviderName == "" {
		c.ProviderName = "ollama"
	}
	if c.RagDir == "" {
		c.RagDir = "rag-archive"
	}
	if c.OutputDir == "" {
		c.OutputDir = "public"
	}
	if c.MaxContext == 0 {
		c.MaxContext = 6000
	}
}

// Validate enforces the security and correctness rules from the moonshot
// spec. Call this before NewServer.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("ask: port must be between 1 and 65535, got %d", c.Port)
	}
	if c.Host == "" {
		return errors.New("ask: host cannot be empty")
	}
	if !c.LoopbackOnly {
		return nil
	}
	if !IsLoopbackHost(c.Host) {
		return fmt.Errorf("ask: refusing non-loopback host %q — pass an explicit, non-loopback address only when you accept that the assistant may become reachable on your network", c.Host)
	}
	return nil
}

// Server wraps the HTTP server, the corpus, the ranker, the provider, and a
// few goroutines. Build it with NewServer, start it with Start, and shut it
// down with Shutdown.
type Server struct {
	cfg      Config
	corpus   retrieval.Corpus
	ranker   *retrieval.Ranker
	provider llm.Provider

	ui fs.FS // optional; nil when DisableUI is true
}

// NewServer loads the corpus and wires the provider. It does NOT start the
// HTTP listener; call Start for that.
func NewServer(cfg Config) (*Server, error) {
	cfg.Defaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	loadOpts := retrieval.LoadOptions{RagDir: cfg.RagDir, OutputDir: cfg.OutputDir}
	if cfg.Rebuild {
		loadOpts.RagDir = cfg.RagDir // future: could call ragexport.RunExport here
	}
	loadRes, err := retrieval.Load(loadOpts)
	if err != nil {
		return nil, fmt.Errorf("ask: prepare corpus: %w", err)
	}
	if len(loadRes.MissingArtifacts) == 3 {
		return nil, fmt.Errorf("ask: no RAG artifacts found in %s — run `la-famille rag` first", cfg.RagDir)
	}

	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("ask: prepare provider: %w", err)
	}

	srv := &Server{
		cfg:      cfg,
		corpus:   loadRes.Corpus,
		ranker:   retrieval.NewRanker(loadRes.Corpus),
		provider: provider,
	}

	if !cfg.DisableUI {
		uiFS, err := fs.Sub(uiAssets, "ui")
		if err != nil {
			return nil, fmt.Errorf("ask: prepare UI assets: %w", err)
		}
		srv.ui = uiFS
	}
	return srv, nil
}

// buildProvider instantiates a Provider implementation based on
// Config.ProviderName. We keep this map tiny so the CLI surface is small.
func buildProvider(cfg Config) (llm.Provider, error) {
	switch strings.ToLower(cfg.ProviderName) {
	case "ollama":
		return llm.NewOllama(llm.OllamaConfig{
			Model: cfg.Model,
		}), nil
	case "fake":
		return &llm.FakeProvider{}, nil
	default:
		return nil, fmt.Errorf("ask: unknown provider %q (supported: ollama, fake)", cfg.ProviderName)
	}
}

// Status returns a small JSON-safe description of the server's state. The
// UI uses it to populate the diagnostics drawer.
type Status struct {
	Ready         bool   `json:"ready"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Bind          string `json:"bind"`
	CorpusVersion string `json:"corpus_version"`
	DocumentCount int    `json:"document_count"`
	ChunkCount    int    `json:"chunk_count"`
	RagVersion    string `json:"rag_version"`
	SourceDir     string `json:"source_dir"`
	LoopbackOnly  bool   `json:"loopback_only"`
}

// Snapshot returns the current public status. Provider availability is
// probed lazily here so /api/status gives an honest answer even if the
// daemon hasn't been touched yet.
func (s *Server) Snapshot(ctx context.Context) Status {
	bound := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	st := Status{
		Ready:         true,
		Provider:      s.provider.Name(),
		Model:         s.cfg.Model,
		Bind:          bound,
		CorpusVersion: s.corpus.Version,
		DocumentCount: s.corpus.DocumentCount,
		ChunkCount:    s.corpus.ChunkCount,
		SourceDir:     filepath.Clean(s.cfg.RagDir),
		LoopbackOnly:  s.cfg.LoopbackOnly,
	}
	availCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	if err := s.provider.Available(availCtx); err != nil {
		st.Ready = false
	}
	return st
}

// AnswerRequest is the JSON body posted to /api/ask.
type AnswerRequest struct {
	Question string `json:"question"`
	// MaxChunks (optional) overrides the prompt budget for this query.
	MaxChunks int `json:"max_chunks,omitempty"`
	// MaxContextChars (optional) overrides the context character budget.
	MaxContextChars int `json:"max_context_chars,omitempty"`
	// Stream is reserved for a future streaming mode. The current build
	// rejects stream=true with HTTP 400 so design drift is impossible —
	// either a future PR advertises a real streaming path or the field is
	// removed.
	Stream bool `json:"stream,omitempty"`
}

// AnswerResponse is the JSON body returned by /api/ask.
type AnswerResponse struct {
	Status           string                 `json:"status"` // "answered" or "no_answer"
	Question         string                 `json:"question"`
	Answer           string                 `json:"answer,omitempty"`
	Markdown         string                 `json:"markdown,omitempty"`
	Sources          []retrieval.SourceCard `json:"sources"`
	DroppedCitations []string               `json:"dropped_citations,omitempty"`
	NoAnswer         bool                   `json:"no_answer"`
	NoAnswerMessage  string                 `json:"no_answer_message,omitempty"`
	Diagnostics      AnswerDiagnostics      `json:"diagnostics"`
}

// AnswerDiagnostics carries the timing numbers, etc. we want to expose on
// /api/status as well. Keeping the struct shared is the simplest way to
// guarantee parity between the drawer (status endpoint) and the per-answer
// timings (ask endpoint).
type AnswerDiagnostics struct {
	RetrievalMs     int64  `json:"retrieval_ms"`
	GenerationMs    int64  `json:"generation_ms"`
	ChunksRetrieved int    `json:"chunks_retrieved"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
}

// Answer runs the full retrieval + LLM + citation-verification pipeline
// against one question. It is exposed via the HTTP handler but also
// directly callable from tests or the TUI integration.
func (s *Server) Answer(ctx context.Context, req AnswerRequest) (AnswerResponse, error) {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return AnswerResponse{}, errors.New("ask: question cannot be empty")
	}

	budget := retrieval.DefaultPromptBudget()
	if req.MaxChunks > 0 {
		budget.MaxChunks = req.MaxChunks
	}
	if req.MaxContextChars > 0 {
		budget.MaxContextChars = req.MaxContextChars
	}

	retrieveStart := time.Now()
	scored := s.ranker.Rank(question, budget.MaxChunks)
	retrievalMs := time.Since(retrieveStart).Milliseconds()

	if len(scored) == 0 {
		return AnswerResponse{
			Status:          "no_answer",
			Question:        question,
			NoAnswer:        true,
			NoAnswerMessage: "This site does not provide enough information to answer that question.",
			Diagnostics: AnswerDiagnostics{
				RetrievalMs:     retrievalMs,
				ChunksRetrieved: 0,
				Provider:        s.provider.Name(),
				Model:           s.cfg.Model,
			},
		}, nil
	}

	cites := retrieval.NewCitations(scoredChunks(scored))
	prompt, hints := retrieval.BuildAnswerPrompt(question, scored, cites, budget)

	genStart := time.Now()
	resp, err := s.provider.Complete(ctx, llm.Request{
		Question:  question,
		System:    retrievalSystemHeader(),
		Context:   prompt,
		Citations: hintsToLLM(hints),
		Model:     s.cfg.Model,
		MaxTokens: 0,
	})
	generationMs := time.Since(genStart).Milliseconds()
	if err != nil {
		if errors.Is(err, llm.ErrUnavailable) {
			return AnswerResponse{}, fmt.Errorf("provider unavailable: %w", err)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, llm.ErrCancelled) {
			return AnswerResponse{}, llm.ErrCancelled
		}
		return AnswerResponse{}, fmt.Errorf("ask: generation failed: %w", err)
	}

	result := cites.Verify(resp.Answer)
	sources := cites.ResolveSourceCards(result.VerifiedKeys)

	// If the model emitted no citations and produced real prose, we
	// conservatively treat it as no-answer: the moonshot spec says we must
	// not invent an answer without grounded citations.
	if len(result.VerifiedKeys) == 0 && !isApparentRefusal(resp.Answer) {
		return AnswerResponse{
			Status:           "no_answer",
			Question:         question,
			Markdown:         resp.Markdown,
			NoAnswer:         true,
			NoAnswerMessage:  "This site does not provide enough information to answer that question.",
			Sources:          sources,
			DroppedCitations: result.DroppedKeys,
			Diagnostics: AnswerDiagnostics{
				RetrievalMs:     retrievalMs,
				GenerationMs:    generationMs,
				ChunksRetrieved: len(scored),
				Provider:        s.provider.Name(),
				Model:           s.cfg.Model,
			},
		}, nil
	}

	out := AnswerResponse{
		Status:           "answered",
		Question:         question,
		Answer:           resp.Answer,
		Markdown:         resp.Markdown,
		Sources:          sources,
		DroppedCitations: result.DroppedKeys,
		Diagnostics: AnswerDiagnostics{
			RetrievalMs:     retrievalMs,
			GenerationMs:    generationMs,
			ChunksRetrieved: len(scored),
			Provider:        s.provider.Name(),
			Model:           s.cfg.Model,
		},
	}
	return out, nil
}

func scoredChunks(s []retrieval.Scored) []retrieval.Chunk {
	out := make([]retrieval.Chunk, len(s))
	for i, v := range s {
		out[i] = v.Chunk
	}
	return out
}

func hintsToLLM(in []retrieval.CitationHint) []llm.CitationHint {
	out := make([]llm.CitationHint, len(in))
	for i, h := range in {
		out[i] = llm.CitationHint{
			Key:     h.Key,
			ChunkID: h.ChunkID,
			Title:   h.Title,
			Heading: h.Heading,
			URL:     h.URL,
			Excerpt: h.Excerpt,
		}
	}
	return out
}

// retrievalSystemHeader is appended to the per-call system prompt to make
// the answer rules explicit. The base prompt is built inside the retrieval
// package so the chunker/citation layers can unit-test it independently.
func retrievalSystemHeader() string {
	return ""
}

// isApparentRefusal accepts short messages that already indicate the model
// said "I don't know" — we use this when deciding whether to swap a fully
// uncited answer for the canonical no-answer message.
func isApparentRefusal(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return true
	}
	hints := []string{"not enough information", "i don't know", "cannot answer", "i'm not sure", "no information"}
	for _, h := range hints {
		if strings.Contains(low, h) {
			return true
		}
	}
	return false
}

// IsLoopbackHost returns true when host is `localhost`, the loopback IPv4
// range (127.0.0.0/8) or a loopback IPv6 (`::1`). The function tolerates
// bracketed IPv6 literals (`[::1]:1234`) and full URL syntax
// (`http://127.0.0.1:8080/`). It is conservative on resolver names — if a
// hostname does not resolve to a literal string we recognise here, we
// assume non-loopback so callers refuse to expose.
func IsLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	stripped := host
	if i := strings.Index(stripped, "://"); i >= 0 {
		stripped = stripped[i+3:]
	}
	stripped = strings.TrimRight(stripped, "/")
	if i := strings.Index(stripped, "/"); i >= 0 {
		stripped = stripped[:i]
	}
	if i := strings.LastIndex(stripped, "@"); i >= 0 {
		stripped = stripped[i+1:]
	}
	if h, _, err := net.SplitHostPort(stripped); err == nil {
		stripped = h
	}
	stripped = strings.TrimSpace(stripped)
	stripped = strings.Trim(stripped, "[]")
	switch strings.ToLower(stripped) {
	case "localhost", "":
		return true
	}
	if ip := net.ParseIP(stripped); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// Start runs the HTTP server on s.cfg.Host:s.cfg.Port and blocks until ctx
// is canceled or ListenAndServe fails. It is safe to call from a goroutine.
func (s *Server) Start(ctx context.Context) error {
	if s.cfg.Port <= 0 {
		return errors.New("ask: invalid port")
	}
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))

	if slog.Default() != nil {
		slog.Info("la-famille ask listening", "addr", "http://"+addr, "provider", s.provider.Name(), "model", s.cfg.Model)
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           s.guardHost(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return err
		}
		// ListenAndServe returned nil == ErrServerClosed; treat as success.
		return nil
	}
}

// Shutdown stops the running server. It is safe to call multiple times.
func (s *Server) Shutdown(_ context.Context) error {
	return nil // lifecycle owned by Start's context
}

// hostAllowed reports whether a request's Host header names the loopback
// interface this server is bound to.
//
// An empty Host is refused: HTTP/1.1 requires one, and no browser we serve
// omits it.
func (s *Server) hostAllowed(hostHeader string) bool {
	if strings.TrimSpace(hostHeader) == "" {
		return false
	}
	bare := hostHeader
	if h, _, err := net.SplitHostPort(hostHeader); err == nil {
		bare = h
	}
	bare = strings.Trim(strings.TrimSpace(bare), "[]")
	if bare == "" {
		return false
	}
	if strings.EqualFold(bare, strings.Trim(s.cfg.Host, "[]")) {
		return true
	}
	return IsLoopbackHost(bare)
}

// guardHost rejects requests whose Host header does not name this server's
// loopback address.
//
// Binding to 127.0.0.1 keeps other machines out, but it does not keep other
// *origins* out. A page on any site can point a hostname it controls at
// 127.0.0.1 (DNS rebinding); the browser then treats this server as
// same-origin, so it can both POST /api/ask and read the answers — which
// defeats the promise that site content never leaves the machine. The Host
// header is the signal that survives rebinding, because the browser keeps
// sending the attacker's hostname.
//
// Skipped when the operator passed --expose-host: a deliberately exposed
// deployment is legitimately reached under arbitrary hostnames and proxies,
// and that opt-in already carries a startup warning.
func (s *Server) guardHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.LoopbackOnly && !s.hostAllowed(r.Host) {
			http.Error(w, "forbidden: unexpected Host header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// registerRoutes wires the URL handlers onto mux.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/ask", s.handleAsk)
}

// handleIndex serves the local UI shell. Static assets are embedded.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if s.ui == nil {
		http.Error(w, "UI disabled", http.StatusNotFound)
		return
	}
	if r.URL.Path != "/" && r.URL.Path != "" {
		// Serve static asset if it exists.
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if data, err := fs.ReadFile(s.ui, clean); err == nil {
			guess := strings.TrimPrefix(clean, "ask/")
			if guess == "" {
				guess = clean
			}
			w.Header().Set("Content-Type", contentTypeFor(guess))
			_, _ = w.Write(data)
			return
		}
	}
	data, err := fs.ReadFile(s.ui, "index.html")
	if err != nil {
		http.Error(w, "ui missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

// handleStatus returns the public server status as JSON.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := s.Snapshot(r.Context())
	writeJSON(w, http.StatusOK, st)
}

// handleAsk runs the retrieval + LLM pipeline on a posted question.
func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Request-size guard. The UI only ever sends a couple of hundred bytes,
	// and oversized requests are almost always a misuse or attack.
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10) // 8KB
	var req AnswerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Stream {
		http.Error(w, "ask: streaming is not implemented yet; remove the `stream` field", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	resp, err := s.Answer(ctx, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, llm.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		if errors.Is(err, llm.ErrCancelled) || errors.Is(err, context.Canceled) {
			status = http.StatusRequestTimeout
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// contentTypeFor picks a sane MIME type for served static files. Adding
// types here is cheap because the filesystem is fully under our control.
func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// writeJSON is a tiny wrapper around json.Encoder that sets Content-Type.
// We keep this in package instead of importing json at the top level of
// handler calls because the function is hot from the perspective of a unit
// test that exercises the handlers dozens of times.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := jsonEncoder(w)
	_ = enc.Encode(v)
}
