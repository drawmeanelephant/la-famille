package retrieval

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tbuddy/la-famille/internal/graph"
)

func graphCorpus() Corpus {
	return Corpus{
		Chunks: []Chunk{
			{ID: "a", PageID: "sensor", Title: "Field Sensor", SourceKind: "rag-content", Text: "Salinity sensor detected a tidal anomaly.", URL: "/project/field/"},
			{ID: "b", PageID: "assay", Title: "Bench Assay", SourceKind: "rag-content", Text: "Chloride examination found elevated concentration.", URL: "/project/lab/"},
			{ID: "c", PageID: "registry", Title: "Dispatch Register", SourceKind: "rag-content", Text: "Custody evidence. Courier Mara delivered the sealed amber vial to [sheet](destination.md).", URL: "/project/dispatch/"},
			{ID: "d", PageID: "decoy", Title: "Sensor Manual", SourceKind: "rag-content", Text: "Salinity sensor chloride instruction sheet."},
		},
		Graph: graph.Graph{
			Nodes: map[string]graph.Node{
				"sensor": {Type: "page", Render: true}, "registry": {Type: "page", Render: true},
				"assay": {Type: "page", Render: true}, "decoy": {Type: "page", Render: true},
			},
			Edges: [][2]string{{"sensor", "registry"}, {"registry", "assay"}},
		},
	}
}

func TestGraphConnectingPathAndStableCitations(t *testing.T) {
	c := graphCorpus()
	g := NewGraphRanker(c)
	query := "How are Field Sensor and Bench Assay connected?"
	off := NewRanker(c).Rank(query, 3)
	on := g.Retrieve(query, 3)
	if slices.ContainsFunc(off, func(s Scored) bool { return s.Chunk.PageID == "registry" }) {
		t.Fatal("baseline already retrieved the bridge")
	}
	if len(on.Paths) != 1 || len(on.Scored) != 3 {
		t.Fatalf("missing bounded route: %+v", on)
	}
	var ids []string
	for _, node := range on.Paths[0].Nodes {
		ids = append(ids, node.PageID)
	}
	if !reflect.DeepEqual(ids, []string{"sensor", "registry", "assay"}) &&
		!reflect.DeepEqual(ids, []string{"assay", "registry", "sensor"}) {
		t.Fatalf("invented route: %v", ids)
	}
	for _, edge := range on.Paths[0].Edges {
		if !slices.Contains(c.Graph.Edges, edge) {
			t.Fatalf("invented edge direction: %v", edge)
		}
	}
	for range 20 {
		if !reflect.DeepEqual(on, g.Retrieve(query, 3)) {
			t.Fatal("nondeterministic retrieval")
		}
	}
	prompt, hints, paths := BuildGraphAnswerPrompt(query, on, DefaultPromptBudget())
	if len(hints) != 3 || len(paths) != 1 || !strings.Contains(prompt, "Courier Mara") {
		t.Fatalf("missing grounded evidence: %s %+v", prompt, paths)
	}
	cites := NewCitations(scoredChunksOf(on.Scored))
	if len(CitedPaths(paths, cites.Verify("[1] [2] [3] [99]").VerifiedKeys)) != 1 ||
		len(CitedPaths(paths, cites.Verify("[1] [3]").VerifiedKeys)) != 0 {
		t.Fatal("path citation verification did not require the bridge")
	}
}

func TestGraphSinglePagePreservesLexicalAndNeverInventsAnswers(t *testing.T) {
	c := graphCorpus()
	c.Graph.Edges = append(c.Graph.Edges, [2]string{"registry", "sensor"})
	g := NewGraphRanker(c)
	for _, query := range []string{"tidal anomaly", "sealed amber vial", "quuxivorzz", ""} {
		want := NewRanker(c).Rank(query, 5)
		got := g.Retrieve(query, 5)
		if !reflect.DeepEqual(got.Scored, want) || len(got.Paths) != 0 {
			t.Fatalf("single-page/no-answer changed for %q: %+v vs %+v", query, got, want)
		}
	}
}

func TestGraphGroundingSkipsHeadingOnlyChunks(t *testing.T) {
	c := graphCorpus()
	c.Chunks = append(c.Chunks,
		Chunk{ID: "title-a", PageID: "sensor", Title: "Field Sensor", SourceKind: "rag-content", Text: "# Field Sensor"},
		Chunk{ID: "title-b", PageID: "assay", Title: "Bench Assay", SourceKind: "rag-content", Text: "# Bench Assay"},
		Chunk{ID: "title-c", PageID: "registry", Title: "Dispatch Register", SourceKind: "rag-content", Text: "# Dispatch Register"})
	result := NewGraphRanker(c).Retrieve("Field Sensor Bench Assay", 3)
	if len(result.Paths) != 1 {
		t.Fatal("missing graph route")
	}
	for _, node := range result.Paths[0].Nodes {
		ch, _ := c.ChunkByID(node.ChunkID)
		if !hasSourceEvidence(ch) || strings.HasPrefix(ch.ID, "title-") {
			t.Fatalf("route grounded on a title without evidence: %+v", ch)
		}
	}
	reverse := NewGraphRanker(c).Retrieve("Bench Assay Field Sensor", 3)
	if len(reverse.Paths) != 1 || reverse.Paths[0].Nodes[0].PageID != "assay" ||
		result.Paths[0].Nodes[0].PageID != "sensor" {
		t.Fatal("explicit endpoints lost query traversal order")
	}
}

func TestGraphStrongNeighborUsesComplementaryLexicalEvidence(t *testing.T) {
	c := graphCorpus()
	c.Chunks[0].Text = "salinity sensor monitor"
	c.Chunks[1].Text = "chloride concentration chloride record"
	c.Chunks[3].Text = "salinity sensor monitor"
	c.Graph.Edges = [][2]string{{"sensor", "assay"}, {"assay", "sensor"}, {"sensor", "registry"}}
	g := NewGraphRanker(c)
	query := "salinity sensor monitor chloride"
	off := NewRanker(c).Rank(query, 2)
	if slices.ContainsFunc(off, func(s Scored) bool { return s.Chunk.PageID == "assay" }) {
		t.Fatalf("neighbor promotion control is already satisfied by lexical retrieval: %+v", off)
	}
	got := g.Retrieve(query, 2)
	if len(got.Scored) != 2 || len(got.Paths) != 0 ||
		!slices.ContainsFunc(got.Scored, func(s Scored) bool { return s.Chunk.PageID == "assay" }) {
		t.Fatalf("strong neighbor not selected: %+v", got)
	}
	if slices.ContainsFunc(got.Scored, func(s Scored) bool { return s.Chunk.PageID == "registry" }) {
		t.Fatal("zero-relevance neighbor diluted retrieval")
	}
}

func TestGraphPathsAreBoundedAndExcludeUnpublishedNodes(t *testing.T) {
	for _, mutate := range []func(*Corpus){
		func(c *Corpus) { c.Graph.Nodes["registry"] = graph.Node{Type: "page", Render: false} },
		func(c *Corpus) { c.Graph.Nodes["registry"] = graph.Node{Type: "stub", Render: true, Missing: true} },
		func(c *Corpus) { c.Chunks[2].SourceKind = "system" },
		func(c *Corpus) { c.Graph.Edges = nil },
	} {
		c := graphCorpus()
		mutate(&c)
		if got := NewGraphRanker(c).Retrieve("Field Sensor Bench Assay", 5); len(got.Paths) != 0 {
			t.Fatalf("traversed unavailable bridge: %+v", got)
		}
	}
	g := NewGraphRanker(graphCorpus())
	if got := g.Retrieve("Field Sensor Bench Assay", 2); len(got.Paths) != 0 || len(got.Scored) > 2 {
		t.Fatalf("partial route escaped chunk budget: %+v", got)
	}
	c := graphCorpus()
	c.Graph.Edges = nil
	last := "sensor"
	for i := range 5 {
		id := intToKey(i + 10)
		c.Chunks = append(c.Chunks, Chunk{ID: id, PageID: id, SourceKind: "rag-content", Text: "routing"})
		c.Graph.Nodes[id] = graph.Node{Type: "page", Render: true}
		c.Graph.Edges = append(c.Graph.Edges, [2]string{last, id})
		last = id
	}
	c.Graph.Edges = append(c.Graph.Edges, [2]string{last, "assay"})
	if route := NewGraphRanker(c).shortestPath("sensor", "assay"); route != nil {
		t.Fatalf("hop limit ignored: %v", route)
	}
}

func TestGraphPromptBudgetDropsIncompleteRouteAndKeepsUTF8(t *testing.T) {
	c := graphCorpus()
	c.Chunks[2].Text = strings.Repeat("前書き ", 100) + "Courier Mara delivered the vial."
	result := NewGraphRanker(c).Retrieve("Field Sensor Bench Assay", 3)
	prompt, hints, paths := BuildGraphAnswerPrompt("connection", result, PromptBudget{MaxChunks: 3, MaxContextChars: 3000})
	if len(paths) != 1 || len(hints) != 3 || !strings.Contains(prompt, "Courier Mara") {
		t.Fatal("prompt still uses a 160-rune preview instead of supporting text")
	}
	for _, budget := range []PromptBudget{
		{MaxChunks: 2, MaxContextChars: 3000},
		{MaxChunks: 3, MaxContextChars: 7},
	} {
		prompt, hints, paths = BuildGraphAnswerPrompt("connection", result, budget)
		if len(paths) != 0 || !utf8.ValidString(prompt) {
			t.Fatalf("budget advertised incomplete route: %+v", paths)
		}
		for _, hint := range hints {
			if hint.Excerpt == "" {
				t.Fatal("citation without source evidence")
			}
		}
	}
}

func TestLoadGraphUnionsBacklinksAndOutboundAndReportsMalformed(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("graph.json", `{"nodes":{"sensor":{"type":"page","render":true},"registry":{"type":"page","render":true},"assay":{"type":"page","render":true},"decoy":{"type":"page","render":false}},"edges":[["sensor","registry"],["sensor","registry"],["registry","decoy"],["registry","missing"]]}`)
	write("backlinks.json", `{"registry":["sensor"],"assay":["registry"]}`)
	c := graphCorpus()
	if warnings := loadLinkGraph(&c, dir); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	want := [][2]string{{"registry", "assay"}, {"sensor", "registry"}}
	if !reflect.DeepEqual(c.Graph.Edges, want) {
		t.Fatalf("wrong edge union: %v", c.Graph.Edges)
	}
	write("graph.json", "{")
	warnings := loadLinkGraph(&c, dir)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "graph.json") || len(c.Graph.Edges) != 2 {
		t.Fatalf("malformed graph did not fall back to backlinks: %v %v", warnings, c.Graph.Edges)
	}
	c = graphCorpus()
	if warnings := loadLinkGraph(&c, t.TempDir()); len(warnings) != 0 || len(c.Graph.Edges) != 0 {
		t.Fatalf("missing optional graph changed lexical readiness: %v", warnings)
	}
}
