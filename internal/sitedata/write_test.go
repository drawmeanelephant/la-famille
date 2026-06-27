package sitedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/graph"
)

func TestWrite(t *testing.T) {
	tempDir := t.TempDir()

	g := graph.Graph{
		Nodes: map[string]graph.Node{
			"index": {Type: "page", Render: true},
		},
		Edges: [][2]string{
			{"index", "about"},
		},
	}

	backlinks := map[string][]string{
		"about": {"index", "home", "a_test"}, // Intentionally unordered
	}

	metaData := map[string]map[string]string{
		"index": {
			"title": "Home Page",
		},
	}

	err := Write(tempDir, g, backlinks, metaData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 1. Check graph.json
	graphContent, err := os.ReadFile(filepath.Join(tempDir, "graph.json"))
	if err != nil {
		t.Fatalf("Failed to read graph.json: %v", err)
	}
	var readGraph graph.Graph
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

	// 3. Check meta.json
	metaContent, err := os.ReadFile(filepath.Join(tempDir, "meta.json"))
	if err != nil {
		t.Fatalf("Failed to read meta.json: %v", err)
	}
	var readMeta map[string]map[string]string
	if err := json.Unmarshal(metaContent, &readMeta); err != nil {
		t.Fatalf("Failed to parse meta.json: %v", err)
	}

	if readMeta["index"]["title"] != "Home Page" {
		t.Errorf("Unexpected meta content: %+v", readMeta)
	}
}
