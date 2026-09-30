package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestBuildWikiLinksResolveAndGenerateUnresolvedNote(t *testing.T) {
	repoRoot := repositoryRoot(t)
	fixtureRoot := filepath.Join(repoRoot, "assets", "testdata", "sites", "wiki-links")
	projectRoot := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = projectRoot
	cfg.ContentDir = filepath.Join(fixtureRoot, "content")
	cfg.OutputDir = filepath.Join(projectRoot, "public")
	cfg.AssetDir = filepath.Join(repoRoot, "assets")
	cfg.Template = filepath.Join(repoRoot, "templates", "layout.html")
	cfg.GraphExplorer = true

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	index := readOutput(t, cfg, "index.html")
	for _, want := range []string{
		`href="notes/wiki/" rel="nofollow">Wiki Links</a>`,
		`href="notes/target/" rel="nofollow">Target alias</a>`,
		`href="notes/target/#introduction" rel="nofollow">target#Introduction</a>`,
		`href="future-note/" rel="nofollow">Draft</a>`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index.html missing %q:\n%s", want, index)
		}
	}

	target := readOutput(t, cfg, "notes/target/index.html")
	if !strings.Contains(target, `<h2 id="introduction">Introduction</h2>`) {
		t.Errorf("target heading has no fragment id:\n%s", target)
	}
	stub := readOutput(t, cfg, "future-note/index.html")
	if !strings.Contains(stub, "Unresolved Note: Future Note") {
		t.Errorf("unresolved stub is not visibly titled:\n%s", stub)
	}

	manifest, err := sitedata.ReadManifest(filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatalf("read site manifest: %v", err)
	}
	var manifestLinks []sitedata.ManifestLink
	for _, page := range manifest.Pages {
		if page.Identity == "index" {
			manifestLinks = page.Links
			break
		}
	}
	if len(manifestLinks) != 4 {
		t.Fatalf("manifest wiki links = %+v, want 4", manifestLinks)
	}
	resolvedWikiLinks := 0
	var unresolvedWikiLink *sitedata.ManifestLink
	for i := range manifestLinks {
		link := manifestLinks[i]
		if link.Target == "future-note.md" {
			unresolvedWikiLink = &manifestLinks[i]
			continue
		}
		if link.Resolved {
			resolvedWikiLinks++
		} else {
			t.Errorf("known wiki link %+v is unresolved", link)
		}
	}
	if resolvedWikiLinks != 3 {
		t.Errorf("resolved wiki links = %d, want 3", resolvedWikiLinks)
	}
	if unresolvedWikiLink == nil || unresolvedWikiLink.Resolved {
		t.Errorf("unresolved wiki link = %+v, want future-note.md marked unresolved", unresolvedWikiLink)
	}

	var graphData struct {
		Nodes map[string]struct {
			Type string `json:"type"`
		} `json:"nodes"`
		Edges [][2]string `json:"edges"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &graphData); err != nil {
		t.Fatalf("parse graph.json: %v", err)
	}
	if graphData.Nodes["future-note"].Type != "stub" {
		t.Errorf("future-note graph node = %+v, want stub", graphData.Nodes["future-note"])
	}
	if !slices.Contains(graphData.Edges, [2]string{"index", "future-note"}) {
		t.Errorf("graph edges = %v, want index → future-note", graphData.Edges)
	}

	var backlinks map[string][]string
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "backlinks.json")), &backlinks); err != nil {
		t.Fatalf("parse backlinks.json: %v", err)
	}
	if !slices.Contains(backlinks["future-note"], "index") {
		t.Errorf("future-note backlinks = %v, want index", backlinks["future-note"])
	}

	var meta map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "meta.json")), &meta); err != nil {
		t.Fatalf("parse meta.json: %v", err)
	}
	if title, _ := meta["future-note"]["title"].(string); title != "Unresolved Note: Future Note" {
		t.Errorf("meta title = %q, want unresolved-note label", title)
	}

	var explorer struct {
		Nodes []struct {
			ID      string   `json:"id"`
			Title   string   `json:"title"`
			Inbound []string `json:"inbound"`
			Stub    bool     `json:"stub"`
		} `json:"nodes"`
	}
	explorerPath := filepath.Join(cfg.OutputDir, "graph", "data.json")
	data, err := os.ReadFile(explorerPath)
	if err != nil {
		t.Fatalf("read graph/data.json: %v", err)
	}
	if err := json.Unmarshal(data, &explorer); err != nil {
		t.Fatalf("parse graph/data.json: %v", err)
	}
	for _, node := range explorer.Nodes {
		if node.ID != "future-note" {
			continue
		}
		if node.Title != "Unresolved Note: Future Note" || !node.Stub || !slices.Contains(node.Inbound, "index") {
			t.Errorf("graph explorer node = %+v, want titled stub with inbound index edge", node)
		}
		return
	}
	t.Fatal("graph/data.json has no future-note node")
}
