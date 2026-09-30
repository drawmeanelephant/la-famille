package ask

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

type askEmbedder struct{ calls int }

func (e *askEmbedder) Embed(_ context.Context, _ string, texts []string) ([][]float64, error) {
	e.calls++
	out := make([][]float64, len(texts))
	for i := range texts {
		out[i] = []float64{1, 0}
	}
	return out, nil
}

func TestHybridAnswerRespectsCancellation(t *testing.T) {
	root := t.TempDir()
	rag := filepath.Join(root, "rag-archive")
	fixtureCorpus(t, rag)
	srv, err := NewServer(Config{
		ProviderName: "fake", RagDir: rag, CacheDir: root,
		LoopbackOnly: true, Embeddings: true, embedder: &cancelEmbedder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := srv.Answer(ctx, AnswerRequest{Question: "Where is the RAG archive?"}); !errors.Is(err, llm.ErrCancelled) {
		t.Fatalf("canceled query fell through to lexical answer: %v", err)
	}
}

type cancelEmbedder struct{}

func (*cancelEmbedder) Embed(ctx context.Context, _ string, texts []string) ([][]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([][]float64, len(texts))
	for i := range out {
		out[i] = []float64{1, 0}
	}
	return out, nil
}

func TestServerEmbeddingsOptInAndLexicalFallback(t *testing.T) {
	root := t.TempDir()
	rag := filepath.Join(root, "rag-archive")
	fixtureCorpus(t, rag)
	cfg := Config{ProviderName: "fake", RagDir: rag, CacheDir: root, LoopbackOnly: true}
	plain, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := plain.ranker.(*retrieval.Ranker); !ok {
		t.Fatalf("default changed lexical scorer: %T", plain.ranker)
	}
	if _, err := os.Stat(filepath.Join(root, retrieval.VectorFileName)); !os.IsNotExist(err) {
		t.Fatal("embedding index created without opt-in")
	}
	offline := cfg
	offline.Embeddings = true
	offline.EmbeddingModel = "missing-model"
	offline.embedder = failingEmbedder{}
	fallback, err := NewServer(offline)
	if err != nil {
		t.Fatalf("offline embedding must not prevent startup: %v", err)
	}
	got, err := fallback.Answer(context.Background(), AnswerRequest{Question: "Where is the RAG archive?"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := plain.Answer(context.Background(), AnswerRequest{Question: "Where is the RAG archive?"})
	if err != nil || !reflect.DeepEqual(got.Sources, want.Sources) || got.NoAnswer != want.NoAnswer {
		t.Fatalf("fallback changed lexical answer: got=%+v want=%+v err=%v", got, want, err)
	}
	online := cfg
	online.Embeddings = true
	online.EmbeddingModel = "test-model"
	e := &askEmbedder{}
	online.embedder = e
	hybrid, err := NewServer(online)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hybrid.ranker.(*retrieval.HybridRanker); !ok || e.calls == 0 {
		t.Fatalf("opt-in did not build index: %T calls=%d", hybrid.ranker, e.calls)
	}
}

type failingEmbedder struct{}

func (failingEmbedder) Embed(context.Context, string, []string) ([][]float64, error) {
	return nil, context.DeadlineExceeded
}
