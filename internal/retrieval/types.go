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
	ID          string   // stable, e.g. "docs/rag.md#h-1-foo"
	PageID      string   // identifier derived from file path without extension
	Title       string   // YAML title or first H1
	HeadingPath []string // ordered list of headings surrounding this chunk (closest first)
	HeadingText string   // the closest enclosing heading text (empty if no heading)
	URL         string   // generated site URL for the page (may be empty)
	SourcePath  string   // absolute path the chunk came from (rag archive or content)
	SourceKind  string   // "rag-content", "rag-system", "rag-config", "site-meta"
	Text        string   // full chunk text (used for context prompts and snippets)
	// TokenCount is approximate (bytes / 4). Used for budget enforcement
	// before sending chunks to the provider.
	TokenCount int
	// Position is the chunk's index inside its source page; preserved across
	// reloads so citations like "[3]" map to whichever chunk was at position 3.
	Position int
}

// Excerpt returns a short, single-line preview of the chunk text suitable
// for display in source cards. It trims aggressively because the UI renders
// cards compactly.
func (c Chunk) Excerpt(max int) string {
	text := strings.Join(strings.Fields(c.Text), " ")
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max-1]) + "…"
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
	Version       string  // "v1" today; bumped on schema/format changes
	SourceDir     string  // "rag-archive" or whatever --rag-dir was
	DocumentCount int     // number of distinct source files
	ChunkCount    int     // total chunks
	Chunks        []Chunk // immutable slice; never mutated after Load
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
