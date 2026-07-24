package retrieval

import (
	"regexp"
	"strings"
)

// citationPattern matches bracketed numeric keys that the model emits to
// cite a source: "[1]", "[42]", "[ 3 ]". We accept the loose spacing form
// because small local models are not always tidy.
var citationPattern = regexp.MustCompile(`\[\s*([0-9]+)\s*\]`)

// Citations maps a stable chunk ID to a deterministic numeric key string
// ("1", "2", …) so the model can echo them in its answer. The mapping is
// fixed for a given ordered set of chunks: reordering or growing the slice
// shifts the keys, but rerunning without changes produces identical keys.
type Citations struct {
	// order is the canonical ordering used to assign numeric keys.
	order  []Chunk
	keyOf  map[string]string
	chunks map[string]Chunk
}

// NewCitations builds a Citations index. Keys ("1", "2", …) are assigned in
// the order chunks arrive so the top-ranked chunk from `*Ranker.Rank`
// always receives key "1". This is the assignment the moonshot spec calls
// for: the model's most-likely-to-cite chunk should be index 1. Callers
// MUST pass chunks in a stable, score-relevant order. The order used by
// the BM25-lite ranker is score-descending with deterministic tie-breaks
// by chunk ID, so reruns always produce identical keys.
func NewCitations(chunks []Chunk) *Citations {
	c := &Citations{
		order:  append([]Chunk(nil), chunks...),
		keyOf:  make(map[string]string, len(chunks)),
		chunks: make(map[string]Chunk, len(chunks)),
	}
	for i, ch := range c.order {
		key := intToKey(i + 1)
		c.keyOf[ch.ID] = key
		c.chunks[ch.ID] = ch
	}
	return c
}

// Hints returns the full CitationHint slice that should be sent to the
// model so it knows which integer to emit for which chunk.
func (c *Citations) Hints() []CitationHint {
	hints := make([]CitationHint, 0, len(c.order))
	for _, ch := range c.order {
		hints = append(hints, CitationHint{
			Key:     c.keyOf[ch.ID],
			ChunkID: ch.ID,
			Title:   ch.Title,
			Heading: ch.HeadingText,
			URL:     ch.URL,
			Excerpt: ch.Excerpt(120),
		})
	}
	return hints
}

// KeyFor returns the "[N]" key (without brackets) that the model should
// emit to cite chunkID. Returns empty string when chunkID is unknown.
func (c *Citations) KeyFor(chunkID string) string { return c.keyOf[chunkID] }

// ChunkFor looks up the chunk that corresponds to key "[N]". Empty key or
// unknown numbers return ok=false. The key may include or exclude brackets.
func (c *Citations) ChunkFor(key string) (Chunk, bool) {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "[")
	key = strings.TrimSuffix(key, "]")
	idx, err := parseKey(key)
	if err != nil {
		return Chunk{}, false
	}
	if idx < 1 || idx > len(c.order) {
		return Chunk{}, false
	}
	return c.order[idx-1], true
}

// CitationResult is the outcome of validating model output. ValidatedKeys
// are the keys that survived verification; DroppedKeys lists ones the
// model invented. If the entire answer had no citations, ValidatedKeys is
// empty (the caller decides whether that's acceptable).
type CitationResult struct {
	VerifiedKeys []string
	DroppedKeys  []string
}

// Verify scans the model's answer for "[N]" patterns and validates each
// against the Citations index. Keys outside the index are recorded as
// DroppedKeys and removed from the returned VerifiedKeys.
func (c *Citations) Verify(answer string) CitationResult {
	seen := make(map[string]bool)
	var verified, dropped []string
	for _, m := range citationPattern.FindAllStringSubmatch(answer, -1) {
		key := strings.TrimSpace(m[1])
		if _, ok := c.ChunkFor(key); ok {
			if !seen[key] {
				verified = append(verified, key)
				seen[key] = true
			}
			continue
		}
		if !seen[key] {
			dropped = append(dropped, key)
			seen[key] = true
		}
	}
	return CitationResult{VerifiedKeys: verified, DroppedKeys: dropped}
}

// ResolveSourceCards turns verified keys into SourceCard structs ready
// for the UI. Unknown keys are silently skipped (the verifier already
// trimmed them). The returned slice preserves the order of VerifiedKeys.
func (c *Citations) ResolveSourceCards(keys []string) []SourceCard {
	out := make([]SourceCard, 0, len(keys))
	for _, k := range keys {
		ch, ok := c.ChunkFor(k)
		if !ok {
			continue
		}
		out = append(out, SourceCard{
			Key:         "[" + k + "]",
			ChunkID:     ch.ID,
			Title:       ch.Title,
			Heading:     ch.HeadingLabel(),
			HeadingOnly: ch.HeadingText,
			URL:         ch.URL,
			Excerpt:     ch.Excerpt(200),
		})
	}
	return out
}

// SourceCard is the UI-facing view of a citation. We emit ChunkID so the
// Ask server can rehydrate the chunk for copy/paste and developer details.
type SourceCard struct {
	Key         string `json:"key"`
	ChunkID     string `json:"chunk_id"`
	Title       string `json:"title"`
	Heading     string `json:"heading"`
	HeadingOnly string `json:"heading_only"`
	URL         string `json:"url"`
	Excerpt     string `json:"excerpt"`
}

// CitationHint is duplicated here so the retrieval package can produce
// llm.CitationHint values without importing internal/llm inline. The
// internal/ask layer converts this slice to []llm.CitationHint.
//
// We use a plain struct (not a type alias) so the retrieval package stays
// independent of the llm package's import path, leaving minor refactors
// easier when we add vector embeddings.
type CitationHint struct {
	Key     string
	ChunkID string
	Title   string
	Heading string
	URL     string
	Excerpt string
	Score   float64
}

func intToKey(n int) string {
	if n <= 0 {
		return "0"
	}
	// Plain ASCII digits; we cap at "999" because the UI only displays
	// three-character labels.
	if n > 999 {
		return "999"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func parseKey(s string) (int, error) {
	if s == "" {
		return 0, errInvalidKey
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errInvalidKey
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

var errInvalidKey = &keyError{msg: "invalid citation key"}

type keyError struct{ msg string }

func (e *keyError) Error() string { return e.msg }
