package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFakeProviderEcho(t *testing.T) {
	p := &FakeProvider{EchoMode: "echo"}
	resp, err := p.Complete(context.Background(), Request{Question: "What is La Famille?"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(resp.Answer, "What is La Famille?") {
		t.Fatalf("expected answer to echo question, got %q", resp.Answer)
	}
}

func TestFakeProviderCiteReturnsReferencedKey(t *testing.T) {
	p := &FakeProvider{EchoMode: "cite"}
	req := Request{
		Question: "Does it work?",
		Citations: []CitationHint{
			{Key: "3", ChunkID: "c3", Title: "Test"},
		},
	}
	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(resp.Answer, "[3]") {
		t.Fatalf("expected answer to reference key [3], got %q", resp.Answer)
	}
}

func TestFakeProviderMissReturnsBogusKey(t *testing.T) {
	p := &FakeProvider{EchoMode: "miss"}
	resp, err := p.Complete(context.Background(), Request{Question: "Anything"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(resp.Answer, "[99]") {
		t.Fatalf("expected bogus [99], got %q", resp.Answer)
	}
}

func TestFakeProviderForceError(t *testing.T) {
	want := errors.New("provider offline")
	p := &FakeProvider{ForceError: want}
	if _, err := p.Complete(context.Background(), Request{Question: "x"}); !errors.Is(err, want) {
		t.Fatalf("ForceError not propagated: %v", err)
	}
}

func TestFakeProviderAvailableNeverErrors(t *testing.T) {
	p := &FakeProvider{}
	if err := p.Available(context.Background()); err != nil {
		t.Fatalf("Available should never error: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello world", 5); got != "hell…" {
		t.Errorf("Truncate got %q", got)
	}
	if got := Truncate("hi", 5); got != "hi" {
		t.Errorf("Truncate should leave short strings alone, got %q", got)
	}
	if got := Truncate("anything", 0); got != "" {
		t.Errorf("Truncate(0) should return empty, got %q", got)
	}
}

func TestSafeResponseCapsText(t *testing.T) {
	r := Response{Answer: strings.Repeat("a", 1000), Markdown: "m", Citations: []string{"1", "2"}}
	out := SafeResponse(r, 10)
	if got := utf8.RuneCountInString(out.Answer); got > 10 {
		t.Errorf("SafeResponse.Answer not capped: %d runes", got)
	}
	if out.Markdown != "m" || len(out.Citations) != 2 {
		t.Errorf("SafeResponse other fields should be preserved/independent: %+v", out)
	}
}

func TestOllamaLoopbackOnly(t *testing.T) {
	tests := map[string]bool{
		"http://127.0.0.1:11434": true,
		"http://localhost:9999":  true,
		"http://localhost":       true,
		"http://192.168.1.5":     false,
		"http://example.com":     false,
		"http://[::1]:11434":     true,
	}
	for raw, want := range tests {
		if got := isLoopbackHost(raw); got != want {
			t.Errorf("isLoopbackHost(%q)=%v, want %v", raw, got, want)
		}
	}
}

func TestOllamaAvailableRejectsNonLoopback(t *testing.T) {
	o := NewOllama(OllamaConfig{Endpoint: "http://192.168.0.10:11434", Model: "llama3.2"})
	err := o.Available(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for non-loopback endpoint, got %v", err)
	}
}

func TestNewOllamaDefaults(t *testing.T) {
	o := NewOllama(OllamaConfig{})
	if o.cfg.Endpoint != "http://127.0.0.1:11434" {
		t.Errorf("default endpoint wrong: %s", o.cfg.Endpoint)
	}
	if o.cfg.Timeout != 60*1_000_000_000 {
		t.Errorf("default timeout wrong: %v", o.cfg.Timeout)
	}
	if o.Name() != "ollama" {
		t.Errorf("Name wrong: %s", o.Name())
	}
}
