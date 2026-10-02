package pack

import (
	"fmt"
	"io"

	"github.com/tbuddy/la-famille/internal/retrieval"
)

// LoadCorpus verifies a local pack into #618's private opaque snapshot and
// consumes only that snapshot. No member is extracted, executed, or reopened
// by path. The snapshot is closed and removed before returning, even on error.
func LoadCorpus(name string) (retrieval.LoadResult, error) {
	s, err := loadPack(name)
	if err != nil {
		return retrieval.LoadResult{}, fmt.Errorf("pack: verify corpus: %w", err)
	}
	defer s.close()
	return loadSnapshotCorpus(s, name)
}

func loadSnapshotCorpus(s *snapshot, name string) (retrieval.LoadResult, error) {
	var content io.Reader
	artifacts := make(map[string][]byte)
	for _, member := range s.manifest.Members {
		switch member.Path {
		case "rag-content.md":
			content = s.member(member)
		case "meta.json", "search.json", "graph.json", "backlinks.json", "site-manifest.json":
			data, err := io.ReadAll(s.member(member))
			if err != nil {
				return retrieval.LoadResult{}, fmt.Errorf("pack: read %s: %w", member.Path, err)
			}
			artifacts[member.Path] = data
		}
	}
	if content == nil {
		return retrieval.LoadResult{}, fmt.Errorf("pack: corpus requires rag-content.md")
	}
	return retrieval.LoadPayload(name, content, artifacts)
}
