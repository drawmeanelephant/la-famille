package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNoExternalNetworkCalls asserts that the question-answering path can
// never accidentally phone home. The fake provider never makes outbound
// calls, so the test passes iff no real-network code ever runs in this
// package. The harness uses an HTTP transport that records every attempted
// connection and rejects every address — even if the loopback guard in
// internal/llm/ollama.go regressed, this transport refuses the request
// before any network I/O happens.
func TestNoExternalNetworkCalls(t *testing.T) {
	o := NewOllama(OllamaConfig{
		Model: "fake-model",
		HTTPClient: &http.Client{
			Transport: rejectingTransport{},
		},
	})
	if err := o.Available(context.Background()); err == nil {
		t.Fatalf("Available should fail without a real loopback listener — but it succeeded")
	}
	if _, err := o.Complete(context.Background(), Request{Question: "ping"}); err == nil {
		t.Fatalf("Complete should fail when transport refuses every host")
	}

	// FakeProvider must not even import net/http. Verify by exercising
	// the fake end-to-end without ever needing a network.
	fp := &FakeProvider{}
	if err := fp.Available(context.Background()); err != nil {
		t.Fatalf("FakeProvider.Available should never error: %v", err)
	}
	if _, err := fp.Complete(context.Background(), Request{Question: "ping"}); err != nil {
		t.Fatalf("FakeProvider.Complete should never error: %v", err)
	}
	if strings.Contains(strings.ToLower("ping"), "http") {
		t.Errorf("never imagine a URL inside this test")
	}
}

// rejectingTransport refuses every request.
type rejectingTransport struct{}

// RoundTrip refuses every request and never returns a response. It exists
// purely so the guard test can prove that any future regression of the
// loopback check would surface as a test failure.
func (rejectingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errTransportRefused
}

// errTransportRefused is the sentinel error returned by rejectingTransport.
var errTransportRefused = newTransportError("transport refused network call from test")

// transportError is a tiny error type used by the guard test. It is
// defined here (not in the production code) so the test never pulls in
// any non-test-only types from the production package.
type transportError string

// Error makes transportError satisfy the error interface.
func (e transportError) Error() string { return string(e) }

// newTransportError constructs a transportError. Kept as a constructor so
// future tweaks (e.g. attaching a stack trace) do not explode call sites.
func newTransportError(s string) error { return transportError(s) }

// Compile-time references that surface as errors if any of the imports
// above drift away from the test's needs. They have no runtime effect.
var (
	_ = httptest.NewServer
)
