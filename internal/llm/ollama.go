package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// OllamaConfig describes how to reach a local Ollama daemon. The defaults
// assume the standard install (http://127.0.0.1:11434).
type OllamaConfig struct {
	// Endpoint is a full URL, e.g. "http://127.0.0.1:11434". When empty, the
	// constructor defaults to "http://127.0.0.1:11434".
	Endpoint string
	// Model is the model identifier (e.g. "llama3.2"). Required for Complete.
	Model string
	// Timeout caps each Complete call. Zero means "no timeout", which is almost
	// always the wrong answer — callers should set a sane value (e.g. 60s).
	Timeout time.Duration
	// HTTPClient, if non-nil, replaces the default client. Useful for tests
	// that want to stub the network layer.
	HTTPClient *http.Client
}

// NewOllama builds an Ollama provider. It does not contact the network; use
// Available separately to ping the daemon.
func NewOllama(cfg OllamaConfig) *Ollama {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://127.0.0.1:11434"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Ollama{cfg: cfg}
}

// Ollama is a Provider implementation that talks to a local Ollama daemon
// over HTTP. It issues /api/generate calls in non-streaming mode. Streaming
// can be added later without changing the Provider interface — the Ask
// server will simply not switch on streaming for Ollama until that happens.
type Ollama struct {
	cfg OllamaConfig
}

// Name returns "ollama".
func (o *Ollama) Name() string { return "ollama" }

// Available pings /api/tags to confirm the daemon is alive and exposes at
// least one model. It deliberately does NOT verify that cfg.Model exists —
// Ollama's /api/generate returns 404 on a missing model and we surface that
// as a precise error from Complete.
func (o *Ollama) Available(ctx context.Context) error {
	if o.cfg.Endpoint == "" {
		return fmt.Errorf("%w: no endpoint configured", ErrUnavailable)
	}
	// Refuse anything that is not loopback. We never want to leak questions
	// off the user's machine by accident.
	if !isLoopbackHost(o.cfg.Endpoint) {
		return fmt.Errorf("%w: %s is not a loopback address", ErrUnavailable, o.cfg.Endpoint)
	}
	client := o.client()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, strings.TrimRight(o.cfg.Endpoint, "/")+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%w: ollama returned status %d", ErrUnavailable, resp.StatusCode)
	}
	// We don't actually need the model list — just confirming reachability.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Complete performs a single non-streaming /api/generate call.
func (o *Ollama) Complete(ctx context.Context, req Request) (Response, error) {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = o.cfg.Model
	}
	if strings.TrimSpace(req.Model) == "" {
		return Response{}, fmt.Errorf("%w: model not configured", ErrUnavailable)
	}

	prompt := buildPrompt(req)
	body, err := json.Marshal(map[string]any{
		"model":      req.Model,
		"prompt":     prompt,
		"stream":     false,
		"keep_alive": "5m",
	})
	if err != nil {
		return Response{}, err
	}

	client := o.client()
	httpCtx, cancel := context.WithTimeout(ctx, o.cfg.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, strings.TrimRight(o.cfg.Endpoint, "/")+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Response{}, ErrCancelled
		}
		return Response{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Response{}, fmt.Errorf("%w: read body: %v", ErrUnavailable, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return Response{}, fmt.Errorf("%w: model %q not found on ollama daemon", ErrUnavailable, req.Model)
	}
	if resp.StatusCode/100 != 2 {
		return Response{}, fmt.Errorf("%w: ollama status %d: %s", ErrUnavailable, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Response{}, fmt.Errorf("parse ollama response: %w", err)
	}
	if !parsed.Done || strings.TrimSpace(parsed.Response) == "" {
		return Response{}, fmt.Errorf("%w: ollama returned empty response", ErrUnavailable)
	}
	return Response{
		Answer:     parsed.Response,
		Markdown:   parsed.Response,
		TokensUsed: estimateTokens(parsed.Response),
	}, nil
}

func (o *Ollama) client() *http.Client {
	if o.cfg.HTTPClient != nil {
		return o.cfg.HTTPClient
	}
	return &http.Client{Timeout: o.cfg.Timeout}
}

// buildPrompt glues the system prompt, retrieved context, citation hints, and
// question into a single string. We deliberately keep this simple — Ollama
// models respond well to a flat prompt with explicit sections.
func buildPrompt(req Request) string {
	var sb strings.Builder
	if req.System != "" {
		sb.WriteString(req.System)
		sb.WriteString("\n\n")
	}
	if len(req.Citations) > 0 {
		sb.WriteString("You may cite sources using bracketed numeric keys like [1], [2]. The mapping is:\n")
		for _, c := range req.Citations {
			title := c.Title
			if title == "" {
				title = c.ChunkID
			}
			heading := c.Heading
			if heading != "" {
				heading = " — " + heading
			}
			fmt.Fprintf(&sb, "  [%s] %s%s\n", c.Key, title, heading)
		}
		sb.WriteString("\n")
	}
	if req.Context != "" {
		sb.WriteString("Source material (do not invent citations):\n")
		sb.WriteString(req.Context)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Question: ")
	sb.WriteString(strings.TrimSpace(req.Question))
	sb.WriteString("\nAnswer:")
	return sb.String()
}

func isLoopbackHost(rawURL string) bool {
	host := rawURL
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	host = strings.TrimRight(host, "/")
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	// net.SplitHostPort understands "[::1]:11434" and "127.0.0.1:11434".
	// When there's no port (e.g. "localhost"), it returns an error and we
	// fall through to use the original string as the host.
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// Strip defensively in case SplitHostPort returned an IPv6 form with
	// leading/trailing whitespace that the caller rebuilt.
	host = strings.TrimSpace(host)
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return len(s) / 4
}
