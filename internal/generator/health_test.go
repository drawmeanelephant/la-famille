package generator

import (
	"reflect"
	"testing"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

func TestComputeContentHealth(t *testing.T) {
	renderTrue := true
	renderFalse := false

	fileMap := map[string]*content.FileMeta{
		"index.md": {
			Title:       "Home",
			Date:        "2026-01-01",
			Description: "Home page description",
			Tags:        []string{"go", "web"},
			Rest:        []byte("Welcome to the website with five words"),
			Render:      &renderTrue,
		},
		"about.md": {
			Title:       "About",
			Date:        "",
			Description: "",
			Tags:        []string{"go", "tui"},
			Rest:        []byte("About page content here"),
			Render:      &renderTrue,
		},
		"unrendered.md": {
			Title:       "Raw Asset",
			Date:        "",
			Description: "",
			Tags:        []string{"ignored"},
			Rest:        []byte("Raw content"),
			Render:      &renderFalse,
		},
	}

	g := graph.Graph{
		Nodes: map[string]graph.Node{
			"index":         {Type: "page", Render: true},
			"about":         {Type: "page", Render: true},
			"unrendered.md": {Type: "page", Render: false},
		},
		Edges: [][2]string{
			{"about", "index"},
		},
	}

	backlinks := map[string][]string{
		"index": {"about"},
	}

	health := ComputeContentHealth(fileMap, g, backlinks)

	if health.TotalWordCount != 11 {
		t.Errorf("TotalWordCount = %d, want 11", health.TotalWordCount)
	}

	if health.AvgWordsPerPage != 5.5 {
		t.Errorf("AvgWordsPerPage = %f, want 5.5", health.AvgWordsPerPage)
	}

	wantTopTags := []TagCount{
		{Tag: "go", Count: 2},
		{Tag: "ignored", Count: 1},
		{Tag: "tui", Count: 1},
		{Tag: "web", Count: 1},
	}
	if !reflect.DeepEqual(health.TopTags, wantTopTags) {
		t.Errorf("TopTags = %#v, want %#v", health.TopTags, wantTopTags)
	}

	wantOrphans := []string{"about"}
	if !reflect.DeepEqual(health.OrphanedPages, wantOrphans) {
		t.Errorf("OrphanedPages = %#v, want %#v", health.OrphanedPages, wantOrphans)
	}

	if health.NodeCount != 3 {
		t.Errorf("NodeCount = %d, want 3", health.NodeCount)
	}

	if health.EdgeCount != 1 {
		t.Errorf("EdgeCount = %d, want 1", health.EdgeCount)
	}

	wantMissingDesc := []string{"about"}
	if !reflect.DeepEqual(health.MissingDescriptions, wantMissingDesc) {
		t.Errorf("MissingDescriptions = %#v, want %#v", health.MissingDescriptions, wantMissingDesc)
	}

	wantMissingDate := []string{"about"}
	if !reflect.DeepEqual(health.MissingDates, wantMissingDate) {
		t.Errorf("MissingDates = %#v, want %#v", health.MissingDates, wantMissingDate)
	}
}
