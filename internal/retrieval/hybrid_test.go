package retrieval

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type testEmbedder struct {
	calls  int
	inputs int
	delay  time.Duration
}

func (e *testEmbedder) Embed(_ context.Context, _ string, texts []string) ([][]float64, error) {
	e.calls++
	e.inputs += len(texts)
	if len(texts) > 1 {
		time.Sleep(e.delay)
	}
	out := make([][]float64, len(texts))
	for i, text := range texts {
		switch text {
		case "semantic question", "Warm room with no power":
			out[i] = []float64{1, 0}
		default:
			out[i] = []float64{0, 1}
		}
	}
	return out, nil
}

func testHybridCorpus() Corpus {
	return Corpus{Chunks: []Chunk{
		{ID: "a#h0", PageID: "a", Text: "Warm room with no power"},
		{ID: "b#h0", PageID: "b", Text: "Unrelated words"},
	}}
}

func TestHybridIndexReuseAndInvalidation(t *testing.T) {
	c := testHybridCorpus()
	cache := filepath.Join(t.TempDir(), ".la-famille-vectors.json")
	embedder := &testEmbedder{delay: 150 * time.Millisecond}
	start := time.Now()
	first, err := NewHybridRanker(context.Background(), c, embedder, "model", cache, "build-a")
	firstDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if embedder.inputs != len(c.Chunks) {
		t.Fatalf("first build embedded %d chunks, want %d", embedder.inputs, len(c.Chunks))
	}
	before, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cache)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("vector index is not private: info=%v err=%v", info, err)
	}
	result, err := first.RankContext(context.Background(), "semantic question", 2)
	if err != nil || len(result) != 1 || result[0].Chunk.ID != "a#h0" {
		t.Fatalf("semantic ranking = %v, %v", result, err)
	}
	built := embedder.inputs
	start = time.Now()
	second, err := NewHybridRanker(context.Background(), c, embedder, "model", cache, "build-a")
	secondDuration := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if embedder.inputs != built {
		t.Fatal("cache hit still embedded chunks")
	}
	if secondDuration >= firstDuration {
		t.Fatalf("second run did not skip embedding work: first=%v second=%v", firstDuration, secondDuration)
	}
	t.Logf("vector index first=%v reused=%v, bytes unchanged", firstDuration, secondDuration)
	after, err := os.ReadFile(cache)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("cache changed on second run: %v", err)
	}
	secondResult, err := second.RankContext(context.Background(), "semantic question", 2)
	if err != nil || !reflect.DeepEqual(result, secondResult) {
		t.Fatalf("cache hit changed ranking: %v, %v", secondResult, err)
	}
	built = embedder.inputs
	_, err = NewHybridRanker(context.Background(), c, embedder, "model", cache, "build-b")
	if err != nil || embedder.inputs != built+len(c.Chunks) {
		t.Fatalf("build fingerprint did not invalidate cache: inputs=%d err=%v", embedder.inputs, err)
	}
	_, err = NewHybridRanker(context.Background(), c, embedder, "another-model", cache, "build-b")
	if err != nil || embedder.inputs != built+2*len(c.Chunks) {
		t.Fatalf("model did not invalidate cache: inputs=%d err=%v", embedder.inputs, err)
	}
	c.Chunks[0].Text += " edited"
	_, err = NewHybridRanker(context.Background(), c, embedder, "another-model", cache, "build-b")
	if err != nil || embedder.inputs != built+3*len(c.Chunks) {
		t.Fatalf("corpus change did not invalidate cache: inputs=%d err=%v", embedder.inputs, err)
	}
}

func TestReciprocalRankFusionAndEmptyLexical(t *testing.T) {
	c := testHybridCorpus()
	embedder := &testEmbedder{}
	h, err := NewHybridRanker(context.Background(), c, embedder, "model", filepath.Join(t.TempDir(), "vectors.json"), "fp")
	if err != nil {
		t.Fatal(err)
	}
	got, err := h.RankContext(context.Background(), "semantic question", 5)
	if err != nil || len(got) != 1 || got[0].Chunk.ID != "a#h0" {
		t.Fatalf("dense-only ranking = %v, %v", got, err)
	}
	lex := NewRanker(c).Rank("Unrelated words", 5)
	fused := reciprocalRankFusion(lex, []Scored{{Chunk: c.Chunks[0], Score: 0.99}, {Chunk: c.Chunks[1], Score: 0.01}}, 2)
	if len(fused) != 2 || fused[0].Chunk.ID != "b#h0" {
		t.Fatalf("lexical + dense overlap should rank first: %v", fused)
	}
	tie := reciprocalRankFusion(
		[]Scored{{Chunk: c.Chunks[0]}, {Chunk: c.Chunks[1]}},
		[]Scored{{Chunk: c.Chunks[1]}, {Chunk: c.Chunks[0]}}, 2)
	if len(tie) != 2 || tie[0].Chunk.ID != "a#h0" {
		t.Fatalf("RRF tie must break by stable chunk ID: %v", tie)
	}
	if got, err := h.RankContext(context.Background(), "", 5); err != nil || len(got) != 0 {
		t.Fatalf("empty query must remain empty: %v, %v", got, err)
	}
}

func TestVectorIndexRejectsCorruptionAndUsesBuildFingerprint(t *testing.T) {
	c := testHybridCorpus()
	cache := filepath.Join(t.TempDir(), VectorFileName)
	embedder := &testEmbedder{}
	if _, err := NewHybridRanker(context.Background(), c, embedder, "model", cache, "build-a"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, []byte(`{"version":1,"fingerprint":"build-a","corpus_digest":"wrong"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewHybridRanker(context.Background(), c, embedder, "model", cache, "build-a"); err != nil || embedder.inputs != 2*len(c.Chunks) {
		t.Fatalf("corrupt cache reused: inputs=%d err=%v", embedder.inputs, err)
	}
	buildCache := filepath.Join(t.TempDir(), ".la-famille-cache.json")
	if got := BuildFingerprint(buildCache, c); got != CorpusDigest(c) {
		t.Fatalf("missing build cache should use corpus digest: %s", got)
	}
	if err := os.WriteFile(buildCache, []byte(`{"version":3,"fingerprint":"the-generator-fingerprint"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := BuildFingerprint(buildCache, c); got != "the-generator-fingerprint" {
		t.Fatalf("ignored generator fingerprint: %s", got)
	}
}
