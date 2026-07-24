package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// FakeProvider is a deterministic Provider used in tests and as a default
// when no real daemon is reachable. It echoes the user's question back in a
// structured form so the citation pipeline can be exercised end-to-end.
//
// The EchoMode setting controls the response style:
//
//   - "cite": produce a single paragraph that references the first citation
//     key (so the verifier must accept it).
//   - "miss": produce a paragraph that references a non-existent key [99],
//     so the verifier must reject it.
//   - "<empty>": produce empty prose (used to exercise no-answer paths).
//
// Concurrent calls are safe.
type FakeProvider struct {
	mu       sync.Mutex
	EchoMode string
	// ForceError, when non-nil, makes Complete return that error verbatim.
	ForceError error
}

// Name returns "fake".
func (f *FakeProvider) Name() string { return "fake" }

// Available always succeeds.
func (f *FakeProvider) Available(_ context.Context) error { return nil }

// Complete returns a deterministic response based on EchoMode.
func (f *FakeProvider) Complete(_ context.Context, req Request) (Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ForceError != nil {
		return Response{}, f.ForceError
	}

	switch f.EchoMode {
	case "miss":
		return Response{
			Answer:   fmt.Sprintf("I cannot verify this claim. See bogus source [99] (no such citation)."),
			Markdown: fmt.Sprintf("I cannot verify this claim. See bogus source [99] (no such citation)."),
		}, nil
	case "cite":
		key := "1"
		if len(req.Citations) > 0 {
			key = req.Citations[0].Key
		}
		out := fmt.Sprintf("According to [%s], the answer to %q is: yes.\n\nThis is a deterministic test response.", key, strings.TrimSpace(req.Question))
		return Response{Answer: out, Markdown: out}, nil
	case "empty":
		return Response{}, nil
	default:
		out := fmt.Sprintf("Echo: %s", strings.TrimSpace(req.Question))
		return Response{Answer: out, Markdown: out}, nil
	}
}
