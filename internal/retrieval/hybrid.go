package retrieval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Embedder is the local vector operation, separate from text completion.
type Embedder interface {
	Embed(context.Context, string, []string) ([][]float64, error)
}

// Scorer preserves the existing lexical ranking API for callers that do not
// opt into embeddings.
type Scorer interface {
	Rank(string, int) []Scored
}

const VectorFileName = ".la-famille-vectors.json"

type vectorEntry struct {
	ID     string    `json:"id"`
	Vector []float64 `json:"vector"`
}

type vectorIndex struct {
	Version     int           `json:"version"`
	Fingerprint string        `json:"fingerprint"`
	Digest      string        `json:"corpus_digest"`
	Model       string        `json:"model"`
	Entries     []vectorEntry `json:"entries"`
}

// CorpusDigest captures the exact chunks sent to the embedding model. Stable
// IDs alone cannot detect an edited chunk with the same heading and position.
func CorpusDigest(c Corpus) string {
	h := sha256.New()
	for _, ch := range c.Chunks {
		_, _ = io.WriteString(h, ch.ID+"\x00"+ch.Title+"\x00"+strings.Join(ch.HeadingPath, "\x00")+"\x00"+ch.Text+"\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
}

// BuildFingerprint reads the generator's existing cache fingerprint, which
// already covers content, config, templates, assets and generator identity.
// A missing/invalid build cache uses the corpus digest; RAG archives can be
// exported independently of `build`, and must still get safe invalidation.
func BuildFingerprint(cachePath string, c Corpus) string {
	var cache struct {
		Version     int    `json:"version"`
		Fingerprint string `json:"fingerprint"`
	}
	raw, err := os.ReadFile(cachePath)
	if err == nil && json.Unmarshal(raw, &cache) == nil && cache.Version == 3 && cache.Fingerprint != "" {
		return cache.Fingerprint
	}
	return CorpusDigest(c)
}

// HybridRanker fuses the complete BM25-lite and dense rankings with RRF.
// Its lexical arm includes the query-coverage guard; dense matches remain an
// independent, explicitly opted-in source of evidence.
type HybridRanker struct {
	lexical  *Ranker
	corpus   Corpus
	embedder Embedder
	model    string
	vectors  [][]float64
}

// NewHybridRanker loads a validated flat-file index or embeds every chunk in
// bounded batches. Cache writes are atomic and private; no vectors enter the
// generated site. The caller opts in explicitly.
func NewHybridRanker(ctx context.Context, c Corpus, embedder Embedder, model, cachePath, fingerprint string) (*HybridRanker, error) {
	if embedder == nil || strings.TrimSpace(model) == "" || cachePath == "" || fingerprint == "" {
		return nil, fmt.Errorf("retrieval: hybrid embeddings require a provider, model, cache and fingerprint")
	}
	digest := CorpusDigest(c)
	index, err := readVectorIndex(cachePath)
	if err != nil || !index.valid(c, model, fingerprint, digest) {
		index = vectorIndex{Version: 1, Fingerprint: fingerprint, Digest: digest, Model: model}
		for start := 0; start < len(c.Chunks); start += 16 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			end := min(start+16, len(c.Chunks))
			texts := make([]string, 0, end-start)
			for _, ch := range c.Chunks[start:end] {
				texts = append(texts, embeddingText(ch))
			}
			vectors, err := embedder.Embed(ctx, model, texts)
			if err != nil {
				return nil, err
			}
			if len(vectors) != len(texts) {
				return nil, fmt.Errorf("retrieval: embedding count mismatch")
			}
			for i, v := range vectors {
				if !validVector(v) || (len(index.Entries) > 0 && len(v) != len(index.Entries[0].Vector)) {
					return nil, fmt.Errorf("retrieval: invalid embedding vector")
				}
				index.Entries = append(index.Entries, vectorEntry{ID: c.Chunks[start+i].ID, Vector: v})
			}
		}
		if err := writeVectorIndex(cachePath, index); err != nil {
			return nil, fmt.Errorf("retrieval: save vector index: %w", err)
		}
	}
	h := &HybridRanker{lexical: NewRanker(c), corpus: c, embedder: embedder, model: model}
	for _, entry := range index.Entries {
		h.vectors = append(h.vectors, entry.Vector)
	}
	return h, nil
}

func embeddingText(ch Chunk) string {
	return strings.TrimSpace(ch.Title + "\n" + strings.Join(ch.HeadingPath, " > ") + "\n" + ch.Text)
}

func (idx vectorIndex) valid(c Corpus, model, fingerprint, digest string) bool {
	if idx.Version != 1 || idx.Model != model || idx.Fingerprint != fingerprint || idx.Digest != digest || len(idx.Entries) != len(c.Chunks) {
		return false
	}
	dim := 0
	for i, entry := range idx.Entries {
		if entry.ID != c.Chunks[i].ID || !validVector(entry.Vector) || (dim != 0 && dim != len(entry.Vector)) {
			return false
		}
		dim = len(entry.Vector)
	}
	return true
}

func validVector(v []float64) bool {
	if len(v) == 0 {
		return false
	}
	norm := 0.0
	for _, value := range v {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
		norm += value * value
	}
	return norm > 0 && !math.IsInf(norm, 0)
}

func readVectorIndex(path string) (vectorIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return vectorIndex{}, err
	}
	defer f.Close()
	var idx vectorIndex
	if err := json.NewDecoder(io.LimitReader(f, 128<<20)).Decode(&idx); err != nil {
		return vectorIndex{}, err
	}
	return idx, nil
}

func writeVectorIndex(path string, idx vectorIndex) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".la-famille-vectors-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return err
	}
	err = json.NewEncoder(f).Encode(idx)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// RankContext embeds the query, then fuses complete lexical and dense lists.
// Only positive cosine matches enter the dense list, so orthogonal chunks
// never become context merely because the model returned a vector for them.
func (h *HybridRanker) RankContext(ctx context.Context, query string, topK int) ([]Scored, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = 5
	}
	lexical := h.lexical.Rank(query, len(h.corpus.Chunks))
	vectors, err := h.embedder.Embed(ctx, h.model, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 || !validVector(vectors[0]) || (len(h.vectors) > 0 && len(vectors[0]) != len(h.vectors[0])) {
		return nil, fmt.Errorf("retrieval: invalid query embedding")
	}
	dense := make([]Scored, 0, len(h.vectors))
	for i, vector := range h.vectors {
		sim := cosine(vectors[0], vector)
		if sim > 0 {
			dense = append(dense, Scored{Chunk: h.corpus.Chunks[i], Score: sim})
		}
	}
	sort.Slice(dense, func(i, j int) bool {
		if dense[i].Score != dense[j].Score {
			return dense[i].Score > dense[j].Score
		}
		return dense[i].Chunk.ID < dense[j].Chunk.ID
	})
	return reciprocalRankFusion(lexical, dense, topK), nil
}

func (h *HybridRanker) Rank(query string, topK int) []Scored {
	scored, err := h.RankContext(context.Background(), query, topK)
	if err != nil {
		return h.lexical.Rank(query, topK)
	}
	return scored
}

func cosine(a, b []float64) float64 {
	var dot, aNorm, bNorm float64
	for i := range a {
		dot += a[i] * b[i]
		aNorm += a[i] * a[i]
		bNorm += b[i] * b[i]
	}
	return dot / (math.Sqrt(aNorm) * math.Sqrt(bNorm))
}

func reciprocalRankFusion(lexical, dense []Scored, topK int) []Scored {
	const rrfK = 60
	combined := make(map[string]Scored, len(lexical)+len(dense))
	for _, ranking := range [][]Scored{lexical, dense} {
		for i, item := range ranking {
			old := combined[item.Chunk.ID]
			old.Chunk = item.Chunk
			old.Score += 1.0 / float64(rrfK+i+1)
			combined[item.Chunk.ID] = old
		}
	}
	out := make([]Scored, 0, len(combined))
	for _, item := range combined {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Chunk.ID < out[j].Chunk.ID
	})
	if len(out) > topK {
		out = out[:topK]
	}
	return out
}
