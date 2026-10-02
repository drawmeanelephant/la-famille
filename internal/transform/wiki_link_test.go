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

func TestExcludedLinksDoNotStopTargetTraversal(t *testing.T) {
	for _, first := range []string{"[[Private|first label]]", "[first label](private.md)"} {
		t.Run(first, func(t *testing.T) {
			excluded := false
			missing := make(map[string][]string)
			backlinks := make(map[string][]string)
			g := &graph.Graph{Nodes: make(map[string]graph.Node)}
			transformer := &LinkTransformer{
				CurrentFile: "index.md",
				FileMap: map[string]*content.FileMeta{
					"index.md":   {Title: "Home"},
					"private.md": {Title: "Private", Publish: &excluded},
					"public.md":  {Title: "Public"},
				},
				MissingFiles:  missing,
				MissingTitles: make(map[string]string),
				Backlinks:     backlinks,
				Graph:         g,
			}
			engine := goldmark.New(goldmark.WithParserOptions(
				parser.WithASTTransformers(util.Prioritized(transformer, 100)),
				parser.WithInlineParsers(util.Prioritized(&WikiLinkParser{}, 150)),
			))
			var output bytes.Buffer
			source := first + " [[Private|wiki label]] [markdown label](private.md) " +
				"[[Public|public wiki]] [public markdown](public.md) [[Missing|future note]]"
			if err := engine.Convert([]byte(source), &output); err != nil {
				t.Fatal(err)
			}
			for _, label := range []string{"first label", "wiki label", "markdown label"} {
				if !strings.Contains(output.String(), label) {
					t.Errorf("excluded link label %q was lost: %s", label, output.String())
				}
			}
			if strings.Contains(output.String(), `href="private`) {
				t.Errorf("excluded target still linked: %s", output.String())
			}
			for _, want := range []string{
				`href="public/">public wiki</a>`,
				`href="public/">public markdown</a>`,
				`href="missing/">future note</a>`,
			} {
				if !strings.Contains(output.String(), want) {
					t.Errorf("link after exclusion missing %q: %s", want, output.String())
				}
			}
			if len(g.Edges) != 3 || !containsEdge(g.Edges, [2]string{"index", "missing"}) {
				t.Errorf("graph edges = %v, want only two public edges and one missing edge", g.Edges)
			}
			if _, exists := backlinks["private"]; exists {
				t.Error("excluded target has backlinks")
			}
			if len(missing) != 1 || len(missing["missing.md"]) != 1 {
				t.Errorf("missing targets = %v, want only missing.md", missing)
			}
		})
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
