package askeval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/llm"
)

func TestMeasuredEmbedderCountsSingletonInputsAndFailures(t *testing.T) {
	e := &measuredEmbedder{inner: &evalEmbedder{}}
	if _, err := e.Embed(context.Background(), "unit", []string{"one chunk"}); err != nil {
		t.Fatal(err)
	}
	if e.stats.Calls != 1 || e.stats.Inputs != 1 || e.stats.Vectors != 1 || e.stats.Duration < 0 {
		t.Fatalf("singleton batch was not measured: %+v", e.stats)
	}
	e.inner = unavailableEmbedder{}
	if _, err := e.Embed(context.Background(), "unit", []string{"query"}); err == nil {
		t.Fatal("expected unavailable error")
	}
	if e.stats.Calls != 2 || e.stats.Inputs != 2 || e.stats.Vectors != 1 {
		t.Fatalf("attempts and successful vectors were conflated: %+v", e.stats)
	}
}

func TestIndexSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index")
	if got, err := indexSHA256(path); err != nil || got != "absent" {
		t.Fatalf("missing index: %q %v", got, err)
	}
	raw := []byte("exact index bytes\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if got, err := indexSHA256(path); err != nil || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("index digest: %q %v", got, err)
	}
	if _, err := indexSHA256(filepath.Dir(path)); err == nil {
		t.Fatal("directory read error was hidden")
	}
}

type queryFailureEmbedder struct {
	calls int
}

func (e *queryFailureEmbedder) Embed(ctx context.Context, model string, inputs []string) ([][]float64, error) {
	e.calls++
	if e.calls == 2 {
		return nil, llm.ErrUnavailable
	}
	return (&evalEmbedder{}).Embed(ctx, model, inputs)
}

func TestMeasurementDistinguishesQueryFallbackAndHybrid(t *testing.T) {
	dataset, err := readDataset(testDatasetPath(t, "golden-questions.json"))
	if err != nil {
		t.Fatal(err)
	}
	dataset.Sites = dataset.Sites[:1]
	raw, err := json.Marshal(dataset)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dataset.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), Options{
		DatasetPath: path, ProjectRoot: testProjectRoot(t), Embeddings: true,
		EmbeddingCacheDir: t.TempDir(), Embedder: &queryFailureEmbedder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	site := report.Measurements[0]
	if site.ChunkEmbedding.Vectors != site.Chunks || site.ChunkEmbedding.Calls != 1 ||
		site.IndexSHA256Before != "absent" || len(site.IndexSHA256After) != 64 {
		t.Fatalf("construction measurement: %+v", site)
	}
	failed, next := site.Queries[0], site.Queries[1]
	if !strings.Contains(failed.Ranker, "lexical fallback") || failed.FallbackReason == "" ||
		failed.Embedding.Inputs != 1 || failed.Embedding.Vectors != 0 {
		t.Fatalf("query fallback misreported: %+v", failed)
	}
	if next.Ranker != "BM25-lite + Ollama embeddings (RRF)" || next.Embedding.Vectors != 1 ||
		next.FallbackReason != "" || report.Ranker != "mixed (see per-question measurement modes)" {
		t.Fatalf("hybrid after fallback misreported: %+v, aggregate=%s", next, report.Ranker)
	}
	var output bytes.Buffer
	if err := WriteReport(&output, report); err != nil {
		t.Fatal(err)
	}
	const marker = "Measurement JSON (durations in ns; excludes completion): "
	_, encoded, ok := strings.Cut(output.String(), marker)
	var decoded []SiteMeasurement
	if !ok || json.Unmarshal([]byte(encoded), &decoded) != nil || len(decoded) != 1 ||
		decoded[0].Queries[0].Ranker != failed.Ranker {
		t.Fatalf("missing machine-readable measurements: %s", output.String())
	}
}
