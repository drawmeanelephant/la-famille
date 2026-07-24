// Package llm defines the Provider interface for the Ask This Site assistant
// and shared request/response types. It is intentionally pure-Go with zero
// external dependencies, so it can be embedded by both the local HTTP server
// (internal/ask) and the CLI without surprising compile-time bloat.
//
// Implementations MUST be safe for concurrent use; the Ask server may dispatch
// several requests in flight on a single Provider instance.
package llm

import (
	"context"
	"errors"
	"strings"
)

// Provider is implemented by anything that can answer a grounded question.
// The interface is intentionally minimal so additional local providers (llama.cpp,
// ollama-compatible daemons, etc.) can plug in without touching the orchestrator.
type Provider interface {
	// Name returns the stable identifier of the provider, e.g. "ollama", "fake".
	Name() string

	// Available reports whether the backing service is reachable.
	// Implementations must respect ctx quickly; an unavailable provider should
	// return a non-nil error rather than block.
	Available(ctx context.Context) error

	// Complete issues a single non-streaming completion. Implementations MUST
	// respect ctx cancellation, return a context.Canceled / DeadlineExceeded
	// appropriate error on cancellation, and never panic on empty input.
	Complete(ctx context.Context, req Request) (Response, error)
}

// Request is the structured prompt we send to any provider.
// All fields are optional except Question. Citations map numeric IDs in the
// model's answer (e.g. "[1]") to chunk IDs that we will use to emit verifiable
// source cards in the UI.
type Request struct {
	Question  string
	System    string
	Context   string
	Citations []CitationHint
	MaxTokens int
	Model     string
}

// CitationHint tells the model which integer key in its answer corresponds
// to which chunk identifier. The verifier uses these IDs to build the source
// cards.
type CitationHint struct {
	Key     string `json:"key"`
	ChunkID string `json:"chunk_id"`
	Title   string `json:"title,omitempty"`
	Heading string `json:"heading,omitempty"`
	URL     string `json:"url,omitempty"`
	Excerpt string `json:"excerpt,omitempty"`
}

// Response is the provider's structured answer. The Answer is plain prose
// (markdown source links must be in Markdown field, not Answer). Citations
// contains the *normalized* numeric keys that survived server-side validation.
type Response struct {
	Answer     string
	Markdown   string
	Citations  []string
	TokensUsed int
}

// ErrUnavailable indicates a provider cannot serve requests right now.
// The HTTP layer turns this into a "provider unavailable" UI state.
var ErrUnavailable = errors.New("llm provider unavailable")

// ErrCancelled indicates the user navigated away or the context expired
// before a response was produced.
var ErrCancelled = errors.New("llm request cancelled")

// SafeResponse returns a defensive copy of r whose text fields have been
// length-capped to maxLen runes. The slice copies preserve citation order.
func SafeResponse(r Response, maxLen int) Response {
	out := r
	out.Answer = Truncate(r.Answer, maxLen)
	out.Markdown = Truncate(r.Markdown, maxLen)
	if len(r.Citations) > 0 {
		out.Citations = append([]string(nil), r.Citations...)
	}
	return out
}

// Truncate caps a string to at most maxRunes, adding an ellipsis when
// shortened. Inputs shorter than or equal to the cap (by rune count) are
// returned unchanged.
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes == 1 {
		return "…"
	}
	return string(runes[:maxRunes-1]) + "…"
}

// NormalizeWhitespace collapses runs of whitespace into single spaces. We use
// it in a handful of places so that test fixtures and provider outputs do not
// need to match exact newline counts.
func NormalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
