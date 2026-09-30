package transform

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

func TestWikiLinksResolveAliasesHeadingsAndMissingNotes(t *testing.T) {
	rendered := true
	fileMap := map[string]*content.FileMeta{
		"index.md": {
			Title:  "Home",
			Render: &rendered,
		},
		"notes/wiki-links.md": {
			Title:  "Wiki Links",
			Slug:   "wiki",
			Render: &rendered,
		},
		"notes/target.md": {
			Title:  "Target",
			Render: &rendered,
		},
	}
	missingFiles := make(map[string][]string)
	missingTitles := make(map[string]string)
	backlinks := make(map[string][]string)
	g := &graph.Graph{Nodes: make(map[string]graph.Node)}
	headingTargets := map[string]map[string]bool{
		"notes/target.md": {"introduction": true},
	}
	mu := &sync.Mutex{}
	transformer := &LinkTransformer{
		CurrentFile:        "index.md",
		FileMap:            fileMap,
		MissingFiles:       missingFiles,
		MissingTitles:      missingTitles,
		Backlinks:          backlinks,
		WikiHeadingTargets: headingTargets,
		Graph:              g,
		Mu:                 mu,
	}
	engine := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithASTTransformers(util.Prioritized(transformer, 100)),
			parser.WithInlineParsers(util.Prioritized(&WikiLinkParser{}, 150)),
		),
	)

	source := []byte(`[[Wiki Links]] [[target|Target alias]] [[target#Introduction]] [[Future Note|Draft]]`)
	var output bytes.Buffer
	if err := engine.Convert(source, &output); err != nil {
		t.Fatalf("convert wiki links: %v", err)
	}
	for _, want := range []string{
		`<a href="notes/wiki/">Wiki Links</a>`,
		`<a href="notes/target/">Target alias</a>`,
		`<a href="notes/target/#introduction">target#Introduction</a>`,
		`<a href="future-note/">Draft</a>`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("rendered wiki links missing %q:\n%s", want, output.String())
		}
	}

	for _, edge := range [][2]string{
		{"index", "notes/wiki-links"},
		{"index", "notes/target"},
		{"index", "future-note"},
	} {
		if !containsEdge(g.Edges, edge) {
			t.Errorf("graph edges = %v, want %v", g.Edges, edge)
		}
	}
	if got := missingFiles["future-note.md"]; len(got) != 1 || got[0] != "index.md" {
		t.Errorf("missing parents = %v, want [index.md]", got)
	}
	if got := missingTitles["future-note.md"]; got != "Future Note" {
		t.Errorf("missing title = %q, want Future Note", got)
	}
	if got := backlinks["future-note"]; len(got) != 1 || got[0] != "index" {
		t.Errorf("backlinks = %v, want [index]", got)
	}

	transformer.CurrentFile = "notes/target.md"
	var targetOutput bytes.Buffer
	if err := engine.Convert([]byte("# Introduction\n\nTarget body."), &targetOutput); err != nil {
		t.Fatalf("convert target heading: %v", err)
	}
	if !strings.Contains(targetOutput.String(), `<h1 id="introduction">Introduction</h1>`) {
		t.Errorf("target heading has no wiki-link fragment id:\n%s", targetOutput.String())
	}
}

func TestResolveWikiTargetRejectsAmbiguousTitles(t *testing.T) {
	fileMap := map[string]*content.FileMeta{
		"one.md": {Title: "Same Name"},
		"two.md": {Title: "Same Name"},
	}
	if _, _, ok := ResolveWikiTarget("index.md", "Same Name", fileMap); ok {
		t.Fatal("ResolveWikiTarget() resolved an ambiguous title")
	}
}

func containsEdge(edges [][2]string, want [2]string) bool {
	for _, edge := range edges {
		if edge == want {
			return true
		}
	}
	return false
}
