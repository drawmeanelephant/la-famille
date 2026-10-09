package generator

import (
	"encoding/json"
	"slices"
	"testing"
)

// A page that links to the same target several times contributes one edge
// and one backlink, not one per occurrence: graph.json, backlinks.json and
// the manifest's inbound_link_count are all sets of distinct pages (#650).
func TestBuild_DedupesRepeatedLinkTargets(t *testing.T) {
	cfg := setupCollisionSite(t, map[string]string{
		"index.md": "---\ntitle: Index\n---\nINDEX_BODY\n",
		"a.md":     "---\ntitle: A\n---\n[b](b.md) [b](b.md)\n\n[[b]] [[b]]\n",
		"b.md":     "---\ntitle: B\n---\nB_BODY\n",
	})

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var parsedGraph struct {
		Edges [][2]string `json:"edges"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &parsedGraph); err != nil {
		t.Fatalf("invalid graph.json: %v", err)
	}
	var count int
	for _, edge := range parsedGraph.Edges {
		if edge == [2]string{"a", "b"} {
			count++
		}
	}
	if count != 1 {
		t.Errorf("graph.json edges = %v, want exactly one [a b] edge", parsedGraph.Edges)
	}

	var backlinks map[string][]string
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "backlinks.json")), &backlinks); err != nil {
		t.Fatalf("invalid backlinks.json: %v", err)
	}
	if got := backlinks["b"]; !slices.Equal(got, []string{"a"}) {
		t.Errorf("backlinks[\"b\"] = %v, want [a]", got)
	}

	var manifest struct {
		Pages []struct {
			Identity         string `json:"identity"`
			InboundLinkCount int    `json:"inbound_link_count"`
		} `json:"pages"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "site-manifest.json")), &manifest); err != nil {
		t.Fatalf("invalid site-manifest.json: %v", err)
	}
	for _, page := range manifest.Pages {
		if page.Identity == "b" && page.InboundLinkCount != 1 {
			t.Errorf("b inbound_link_count = %d, want 1 (distinct inbound pages)", page.InboundLinkCount)
		}
	}
}
