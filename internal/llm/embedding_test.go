package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOllamaEmbedBatchAndLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "test-model" || !reflect.DeepEqual(body.Input, []string{"one", "two"}) {
			t.Errorf("unexpected payload: %+v", body)
		}
		_, _ = w.Write([]byte(`{"embeddings":[[1,0],[0,1]]}`))
	}))
	defer server.Close()
	o := NewOllama(OllamaConfig{Endpoint: server.URL})
	got, err := o.Embed(context.Background(), "test-model", []string{"one", "two"})
	if err != nil || !reflect.DeepEqual(got, [][]float64{{1, 0}, {0, 1}}) {
		t.Fatalf("Embed = %v, %v", got, err)
	}

	remote := NewOllama(OllamaConfig{Endpoint: "http://example.com"})
	if _, err := remote.Embed(context.Background(), "model", []string{"secret"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("non-loopback embedding accepted: %v", err)
	}
	if _, err := remote.Complete(context.Background(), Request{Model: "model", Question: "secret"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("non-loopback completion accepted: %v", err)
	}
}

func TestOllamaEmbedRejectsBadResponsesAndRedirects(t *testing.T) {
	for _, body := range []string{
		`{"embeddings":[]}`, `{"embeddings":[[0,0]]}`, `{"embeddings":[[1,2],[3,4]]}`,
		`{"embeddings":[[1,"bad"]]}`, `not json`,
	} {
		t.Run(body, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			defer s.Close()
			if _, err := NewOllama(OllamaConfig{Endpoint: s.URL}).Embed(context.Background(), "model", []string{"one"}); err == nil {
				t.Fatal("malformed embedding accepted")
			}
		})
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/api/embed", http.StatusTemporaryRedirect)
	}))
	defer s.Close()
	if _, err := NewOllama(OllamaConfig{Endpoint: s.URL}).Embed(context.Background(), "model", []string{"one"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("redirect outside loopback accepted: %v", err)
	}
	if _, err := NewOllama(OllamaConfig{Endpoint: s.URL, Model: "model"}).Complete(context.Background(), Request{Question: "secret"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("completion redirect outside loopback accepted: %v", err)
	}
}
