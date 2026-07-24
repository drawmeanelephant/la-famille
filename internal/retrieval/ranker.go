package retrieval

import (
	"math"
	"sort"
	"strings"
)

// Ranker scores chunks against a query with a tiny BM25-lite scorer and
// returns the top-K matches ordered by relevance. The implementation is
// deliberately small (in-memory inverted index, standard tokenization) so
// the corpus can be rebuilt after every `la-famille rag` without a separate
// indexing step.
type Ranker struct {
	corpus Corpus
	index  map[string]map[string]int // term -> chunkID -> frequency
	idf    map[string]float64
	avgDL  float64
	docLen map[string]int
}

// NewRanker precomputes an in-memory inverted index over the corpus. It is
// safe to call multiple times; each call returns an independent Ranker.
func NewRanker(c Corpus) *Ranker {
	r := &Ranker{
		corpus: c,
		index:  make(map[string]map[string]int),
		idf:    make(map[string]float64),
		docLen: make(map[string]int),
	}
	totalLen := 0
	df := make(map[string]int)
	for _, ch := range c.Chunks {
		tokens := tokenize(ch.Text + " " + strings.Join(ch.HeadingPath, " ") + " " + ch.Title)
		r.docLen[ch.ID] = len(tokens)
		totalLen += len(tokens)
		seen := make(map[string]bool)
		for _, t := range tokens {
			if r.index[t] == nil {
				r.index[t] = make(map[string]int)
			}
			r.index[t][ch.ID]++
			if !seen[t] {
				df[t]++
				seen[t] = true
			}
		}
	}
	if len(c.Chunks) > 0 {
		r.avgDL = float64(totalLen) / float64(len(c.Chunks))
	}
	for term, freq := range df {
		// IDF(t) = ln( 1 + (N - df + 0.5) / (df + 0.5) ). Saturated +1
		// stops the score going negative for very common terms.
		r.idf[term] = math.Log(1 + (float64(len(c.Chunks))-float64(freq)+0.5)/(float64(freq)+0.5))
	}
	return r
}

// Rank returns up to topK chunks ordered by relevance. If topK <= 0 or the
// corpus has fewer chunks than topK, fewer results are returned. An empty
// query yields no results.
func (r *Ranker) Rank(query string, topK int) []Scored {
	qTokens := tokenizeQuery(query)
	if len(qTokens) == 0 {
		return nil
	}
	if topK <= 0 {
		topK = 5
	}
	scores := make(map[string]float64, len(r.corpus.Chunks))

	// k1 and b are BM25 hyperparameters. The values mirror the original
	// BM25 paper defaults (k1=1.5, b=0.75). We don't tune them because the
	// ranking is a coarse filter; the LLM does the real selection work.
	const (
		k1 = 1.5
		b  = 0.75
	)
	for _, term := range qTokens {
		idf, ok := r.idf[term]
		if !ok {
			continue
		}
		postings := r.index[term]
		for id, tf := range postings {
			dl := float64(r.docLen[id])
			denom := float64(tf) + k1*(1-b+b*(dl/r.avgDL))
			numer := float64(tf) * (k1 + 1)
			scores[id] += idf * numer / denom
		}
	}
	if len(scores) == 0 {
		return nil
	}

	scored := make([]Scored, 0, len(scores))
	for id, s := range scores {
		ch, ok := r.corpus.ChunkByID(id)
		if !ok {
			continue
		}
		scored = append(scored, Scored{Chunk: ch, Score: s})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Chunk.ID < scored[j].Chunk.ID
	})

	if len(scored) > topK {
		scored = scored[:topK]
	}
	return scored
}

// Scored pairs a chunk with its relevance score. The score is unbounded;
// callers should treat it as an opaque ordering signal.
type Scored struct {
	Chunk Chunk
	Score float64
}
