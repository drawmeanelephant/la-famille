package graph

import (
	"path/filepath"
	"sort"

	"github.com/tbuddy/la-famille/internal/jsonutil"
)

// WriteGraphFiles writes the graph and backlinks data to the output directory.
func WriteGraphFiles(outputDir string, g Graph, backlinks map[string][]string) error {
	// Sort backlinks for deterministic output
	for _, parents := range backlinks {
		sort.Strings(parents)
	}

	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "graph.json"), g); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "backlinks.json"), backlinks); err != nil {
		return err
	}

	return nil
}
