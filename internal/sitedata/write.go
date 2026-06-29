package sitedata

import (
	"path/filepath"
	"sort"

	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/jsonutil"
)

// Write writes the site data (graph, backlinks, meta) to the output directory.
func Write(outputDir string, g graph.Graph, backlinks map[string][]string, metaData map[string]map[string]interface{}) error {
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
	if err := jsonutil.WriteJSON(filepath.Join(outputDir, "meta.json"), metaData); err != nil {
		return err
	}

	return nil
}
