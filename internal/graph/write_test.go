package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGraphFiles(t *testing.T) {
	tempDir := t.TempDir()

	g := Graph{
		Nodes: map[string]Node{
			"index": {Type: "page", Render: true},
		},
		Edges: [][2]string{
			{"index", "about"},
		},
	}

	backlinks := map[string][]string{
		"about": {"index", "home", "a_test"}, // Intentionally unordered
	}

	err := WriteGraphFiles(tempDir, g, backlinks)
	if err != nil {
		t.Fatalf("WriteGraphFiles failed: %v", err)
	}

	// 1. Check graph.json
	graphContent, err := os.ReadFile(filepath.Join(tempDir, "graph.json"))
	if err != nil {
		t.Fatalf("Failed to read graph.json: %v", err)
	}
	var readGraph Graph
	if err := json.Unmarshal(graphContent, &readGraph); err != nil {
		t.Fatalf("Failed to parse graph.json: %v", err)
	}
	if len(readGraph.Nodes) != 1 || readGraph.Nodes["index"].Type != "page" {
		t.Errorf("Unexpected graph content: %+v", readGraph)
	}

	// 2. Check backlinks.json (and ensure sorting happened)
	backlinksContent, err := os.ReadFile(filepath.Join(tempDir, "backlinks.json"))
	if err != nil {
		t.Fatalf("Failed to read backlinks.json: %v", err)
	}
	var readBacklinks map[string][]string
	if err := json.Unmarshal(backlinksContent, &readBacklinks); err != nil {
		t.Fatalf("Failed to parse backlinks.json: %v", err)
	}

	aboutBacklinks := readBacklinks["about"]
	if len(aboutBacklinks) != 3 {
		t.Errorf("Expected 3 backlinks for 'about', got %d", len(aboutBacklinks))
	}
	if aboutBacklinks[0] != "a_test" || aboutBacklinks[1] != "home" || aboutBacklinks[2] != "index" {
		t.Errorf("Backlinks were not sorted correctly: %v", aboutBacklinks)
	}
}
