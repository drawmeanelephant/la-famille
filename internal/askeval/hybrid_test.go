package askeval

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tbuddy/la-famille/internal/llm"
	"github.com/tbuddy/la-famille/internal/retrieval"
)

type evalEmbedder struct{ chunks, queries int }

func (e *evalEmbedder) Embed(_ context.Context, _ string, texts []string) ([][]float64, error) {
	if len(texts) == 1 {
		e.queries++
	} else {
		e.chunks += len(texts)
	}
	out := make([][]float64, len(texts))
	for i := range out {
		out[i] = []float64{1, 0}
	}
	return out, nil
}

type unavailableEmbedder struct{}

func (unavailableEmbedder) Embed(context.Context, string, []string) ([][]float64, error) {
	return nil, llm.ErrUnavailable
}

func TestHybridEvalOfflineFallsBackToHardBaseline(t *testing.T) {
	opts := Options{
		DatasetPath: testDatasetPath(t, "golden-questions-hard.json"),
		ProjectRoot: testProjectRoot(t), Embeddings: true,
		Embedder: unavailableEmbedder{}, EmbeddingCacheDir: t.TempDir(),
	}
	offline, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Embeddings = false
	lexical, err := Run(context.Background(), opts)
	if err != nil || offline.RecallAtK != lexical.RecallAtK ||
		!reflect.DeepEqual(offline.Questions, lexical.Questions) ||
		offline.Ranker != "BM25-lite + Ollama embeddings (RRF; lexical fallback: unavailable)" {
		t.Fatalf("offline changed hard baseline: offline=%+v lexical=%+v err=%v", offline, lexical, err)
	}
	for _, site := range offline.Measurements {
		if site.ChunkEmbedding.Calls != 1 || site.ChunkEmbedding.Inputs == 0 ||
			site.ChunkEmbedding.Vectors != 0 || site.FallbackReason == "" ||
			site.IndexSHA256Before != "absent" || site.IndexSHA256After != "absent" {
			t.Fatalf("unavailable construction misreported: %+v", site)
		}
		for _, query := range site.Queries {
			if query.Ranker != offline.Ranker || query.Embedding.Calls != 0 {
				t.Fatalf("fallback query misreported: %+v", query)
			}
		}
	}
}

func TestHybridEvalReusesIndexAndOffKeepsBaseline(t *testing.T) {
	root := testProjectRoot(t)
	opts := Options{
		DatasetPath: testDatasetPath(t, "golden-questions.json"),
		ProjectRoot: root, Embeddings: true, EmbeddingModel: "unit",
		EmbeddingCacheDir: t.TempDir(),
	}
	e := &evalEmbedder{}
	opts.Embedder = e
	first, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if e.chunks == 0 || first.RecallAtK != 1 || first.Ranker == "BM25-lite" {
		t.Fatalf("hybrid eval not exercised: chunks=%d report=%+v", e.chunks, first)
	}
	indices, err := filepath.Glob(filepath.Join(opts.EmbeddingCacheDir, "*", retrieval.VectorFileName))
	if err != nil || len(indices) == 0 {
		t.Fatalf("no persisted per-site index: %v, %v", indices, err)
	}
	before := make([][]byte, len(indices))
	for i, path := range indices {
		before[i], err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	chunks := e.chunks
	second, err := Run(context.Background(), opts)
	if err != nil || e.chunks != chunks || !reflect.DeepEqual(first.Questions, second.Questions) {
		t.Fatalf("index reused? chunks=%d->%d err=%v", chunks, e.chunks, err)
	}
	for i, site := range second.Measurements {
		cold := first.Measurements[i]
		if cold.ChunkEmbedding.Inputs != cold.Chunks || cold.ChunkEmbedding.Vectors != cold.Chunks ||
			cold.IndexSHA256Before != "absent" || cold.CorpusDigest != site.CorpusDigest ||
			site.IndexSHA256Before != cold.IndexSHA256After || site.IndexSHA256After != cold.IndexSHA256After ||
			site.ChunkEmbedding != (EmbeddingMeasurement{}) {
			t.Fatalf("cold/warm index measurements: cold=%+v warm=%+v", cold, site)
		}
		for _, query := range site.Queries {
			if query.Embedding.Calls != 1 || query.Embedding.Inputs != 1 || query.Embedding.Vectors != 1 ||
				query.Ranker != "BM25-lite + Ollama embeddings (RRF)" || query.RankDuration < 0 {
				t.Fatalf("warm query measurement: %+v", query)
			}
		}
	}
	for i, path := range indices {
		after, err := os.ReadFile(path)
		if err != nil || !reflect.DeepEqual(before[i], after) {
			t.Fatalf("index %s changed on hit: %v", path, err)
		}
	}
	opts.Embeddings = false
	off, err := Run(context.Background(), opts)
	if err != nil || off.RecallAtK != 1 || off.Ranker != "BM25-lite" {
		t.Fatalf("disabled embeddings changed frozen baseline: %+v %v", off, err)
	}
	for _, site := range off.Measurements {
		if site.CachePath != "" || site.IndexDuration != 0 || site.ChunkEmbedding != (EmbeddingMeasurement{}) {
			t.Fatalf("off mode touched embeddings: %+v", site)
		}
	}
}
