// Package retrieval builds an in-memory corpus from the La Famille RAG
// archive and the generated site, ranks chunks for a query, and validates
// citations emitted by the language model. It depends only on the standard
// library plus internal/search for snippet/heading helpers.
//
// The corpus is constructed once at startup (and optionally rebuilt after a
// content change). It is intentionally read-only after Load — Ask This Site
// is a local-first tool and we never mutate user content from the assistant.
package retrieval

import (
	"sort"
	"strings"
)

// Chunk is a single retrievable unit. IDs are deterministic across loads so
// tests and follow-up reruns compare cleanly.
type Chunk struct {
	ID          string
	PageID      string
	Title       string
	HeadingText string
	URL         string
	SourcePath  string
	SourceKind  string
	Text        string
	HeadingPath []string
	TokenCount  int
	Position    int
}

// Excerpt returns a short, single-line preview of the chunk text suitable
// for display in source cards. It trims aggressively because the UI renders
// cards compactly.
func (c Chunk) Excerpt(maxRunes int) string {
	text := strings.Join(strings.Fields(c.Text), " ")
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes-1]) + "…"
}

// HeadingLabel produces a human-readable heading trail: "Page > Foo > Bar".
// Returns just the title if there are no headings.
func (c Chunk) HeadingLabel() string {
	parts := append([]string{}, c.HeadingPath...)
	if len(parts) == 0 {
		return c.Title
	}
	return strings.Join(parts, " > ")
}

// Corpus is a deterministic view over the chunks known to the assistant.
type Corpus struct {
	Version       string
	SourceDir     string
	Chunks        []Chunk
	DocumentCount int
	ChunkCount    int
}

// ChunkByID returns the chunk with the given ID or a zero Chunk and false
// if none is present.
func (c Corpus) ChunkByID(id string) (Chunk, bool) {
	for _, ch := range c.Chunks {
		if ch.ID == id {
			return ch, true
		}
	}
	return Chunk{}, false
}

// SortedChunks returns chunks by Position within their source page, then
// SourcePath for tiebreaker. Used to assign citation keys deterministically
// so that "[N]" indices line up across reruns.
func (c Corpus) SortedChunks() []Chunk {
	out := append([]Chunk(nil), c.Chunks...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourcePath == out[j].SourcePath {
			return out[i].Position < out[j].Position
		}
		return out[i].SourcePath < out[j].SourcePath
	})
	return out
}
