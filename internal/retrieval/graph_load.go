package retrieval

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tbuddy/la-famille/internal/graph"
)

// loadLinkGraph reuses the explorer's directed edge contract. Backlinks are
// target -> sources, not additional reverse links. Missing optional artifacts
// leave lexical retrieval usable; malformed artifacts are surfaced to callers.
func loadLinkGraph(c *Corpus, outputDir string) []string {
	g := graph.Graph{Nodes: make(map[string]graph.Node)}
	var warnings []string
	read := func(name string, dst any) bool {
		raw, err := os.ReadFile(filepath.Join(outputDir, name))
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		if err == nil {
			err = json.Unmarshal(raw, dst)
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", name, err))
			return false
		}
		return true
	}
	var artifact graph.Graph
	hasGraph := read("graph.json", &artifact)
	if hasGraph {
		g = artifact
	}
	var backlinks map[string][]string
	if read("backlinks.json", &backlinks) {
		for target, sources := range backlinks {
			for _, source := range sources {
				g.Edges = append(g.Edges, [2]string{source, target})
			}
		}
	}
	pages := make(map[string]graph.Node)
	for _, ch := range c.Chunks {
		if ch.SourceKind != "rag-content" || ch.PageID == "" {
			continue
		}
		node, ok := g.Nodes[ch.PageID]
		if hasGraph && (!ok || !node.Render || node.Missing || node.Type != "page") {
			continue
		}
		pages[ch.PageID] = graph.Node{Type: "page", Render: true}
	}
	edges := make([][2]string, 0, len(g.Edges))
	for _, edge := range g.Edges {
		_, source := pages[edge[0]]
		_, target := pages[edge[1]]
		if source && target && edge[0] != edge[1] {
			edges = append(edges, edge)
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i][0] != edges[j][0] {
			return edges[i][0] < edges[j][0]
		}
		return edges[i][1] < edges[j][1]
	})
	c.Graph = graph.Graph{Nodes: pages, Edges: slices.Compact(edges)}
	return warnings
}
