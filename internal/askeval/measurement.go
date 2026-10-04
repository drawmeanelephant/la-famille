package askeval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"time"

	"github.com/tbuddy/la-famille/internal/retrieval"
)

// EmbeddingMeasurement counts attempted inputs separately from returned vectors.
// Duration is provider-call wall time, not model-only inference time.
type EmbeddingMeasurement struct {
	Calls    int
	Inputs   int
	Vectors  int
	Duration time.Duration
}

// QueryMeasurement separates embedding from the rest of the ranking call.
// RankDuration is elapsed wall time minus provider-call time, not CPU time.
type QueryMeasurement struct {
	QuestionID     string
	Ranker         string
	FallbackReason string
	Embedding      EmbeddingMeasurement
	RankDuration   time.Duration
}

// SiteMeasurement identifies the exact corpus and private vector index.
// IndexDuration covers NewHybridRanker only, excluding fixture build/load,
// queries and completion. Durations serialize as integer nanoseconds.
type SiteMeasurement struct {
	SiteID            string
	CorpusDigest      string
	Chunks            int
	EmbeddingModel    string
	CachePath         string
	IndexSHA256Before string
	IndexSHA256After  string
	IndexDuration     time.Duration
	ChunkEmbedding    EmbeddingMeasurement
	FallbackReason    string
	Queries           []QueryMeasurement
}

// measuredEmbedder is scoped to one sequential eval site. The caller resets
// its counters between construction and queries; batch size never determines
// whether an input is a chunk or a query (a chunk batch can contain one input).
type measuredEmbedder struct {
	inner retrieval.Embedder
	stats EmbeddingMeasurement
}

func (e *measuredEmbedder) Embed(ctx context.Context, model string, inputs []string) ([][]float64, error) {
	start := time.Now()
	vectors, err := e.inner.Embed(ctx, model, inputs)
	e.stats.Duration += time.Since(start)
	e.stats.Calls++
	e.stats.Inputs += len(inputs)
	if err == nil {
		e.stats.Vectors += len(vectors)
	}
	return vectors, err
}

func indexSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
