package graphexplorer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/graph"
)

func writeMinimumSite(t *testing.T, dir string) config.Config {
	t.Helper()
	content := filepath.Join(dir, "content")
	templates := filepath.Join(dir, "templates")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templates, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "index.md"),
		[]byte("---\ntitle: Home\n---\n# Welcome\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "about.md"),
		[]byte("---\ntitle: About\n---\n# About\nLink to [home](index.md).\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templates, "layout.html"),
		[]byte("{{.Title}} {{.Content}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return config.Config{
		SiteName:      "Test Site",
		Template:      filepath.Join(templates, "layout.html"),
		ContentDir:    content,
		OutputDir:     filepath.Join(dir, "public"),
		AssetDir:      filepath.Join(dir, "assets"),
		RagDir:        filepath.Join(dir, "rag-archive"),
		Theme:         "retro",
		Port:          8080,
		GraphExplorer: true,
	}
}

// sampleInput builds a small but non-symmetric graph: index links to about and
// to a missing page. The asymmetry matters — a symmetric fixture cannot tell a
// correct implementation apart from one that reads edges backwards.
func sampleInput(cfg config.Config) Input {
	g := graph.Graph{
		Nodes: map[string]graph.Node{
			"index":   {Type: "page", Render: true},
			"about":   {Type: "page", Render: true},
			"notes":   {Type: "page", Render: false},
			"missing": {Type: "stub", Render: false, Missing: true},
		},
		Edges: [][2]string{
			{"index", "about"},
			{"index", "about"}, // duplicate link to the same target
			{"index", "missing"},
		},
	}
	meta := map[string]map[string]interface{}{
		"index": {"title": "Home", "word_count": 12, "render": true},
		"about": {"title": "About", "author": "ada", "date": "2026-01-02",
			"tags": []string{"docs"}, "categories": []string{"meta"}, "word_count": 34, "render": true},
		"notes": {"title": "Notes", "word_count": 5, "render": false},
	}
	outputs := map[string]string{
		"index": "index.html",
		"about": "about/index.html",
	}
	return Input{Config: cfg, Graph: g, Meta: meta, PageOutputs: outputs}
}

func nodeByID(t *testing.T, data Data, id string) NodeData {
	t.Helper()
	for _, n := range data.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("node %q not present in payload", id)
	return NodeData{}
}

// repoRoot walks up from the test's working directory to the module root. It
// fails the test rather than skipping: a test that quietly skips when it cannot
// find its fixture reports success while checking nothing.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate module root (no go.mod above %s)", dir)
		}
		dir = parent
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

/* -- Write: enable / disable -- */

func TestWriteExplorerDisabled(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	cfg.GraphExplorer = false

	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if !res.Disabled {
		t.Errorf("expected Result.Disabled=true, got %+v", res)
	}
	if res.IndexPath != "" {
		t.Errorf("expected empty IndexPath when disabled, got %q", res.IndexPath)
	}
	for _, name := range []string{"index.html", "data.json"} {
		full := filepath.Join(cfg.OutputDir, "graph", name)
		if _, err := os.Stat(full); err == nil {
			t.Errorf("expected %s not to exist when explorer disabled", full)
		}
	}
}

func TestWriteExplorerEnabledProducesPageAndData(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)

	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if res.Disabled {
		t.Errorf("expected Result.Disabled=false, got %+v", res)
	}
	if res.IndexPath == "" || res.DataPath == "" {
		t.Fatalf("expected IndexPath and DataPath to be populated, got %+v", res)
	}

	html := string(mustRead(t, res.IndexPath))
	if !strings.Contains(html, "Knowledge Graph") {
		t.Errorf("explorer HTML missing title; got prefix: %.120s", html)
	}
	if !strings.Contains(html, `id="kgx-search-input"`) {
		t.Errorf("explorer HTML missing search input id")
	}
	if !strings.Contains(html, `<link rel="stylesheet" href="../assets/graph/explorer.css">`) {
		t.Errorf("explorer HTML missing CSS link reference")
	}
	if !strings.Contains(html, `<script src="../assets/graph/explorer.js" defer>`) {
		t.Errorf("explorer HTML missing JS script reference")
	}
	if !strings.Contains(html, `data-graph-data="data.json"`) {
		t.Errorf("explorer HTML must stamp the payload location onto <body>")
	}
	if strings.Contains(html, "<style>") {
		t.Errorf("explorer HTML should not inline <style> blocks (CSS ships externally)")
	}
	for _, bad := range []string{"https://cdn.", "//cdn.", "<script src=\"http", "googleapis.com", "jsdelivr.net", "unpkg.com", "https://fonts."} {
		if strings.Contains(html, bad) {
			t.Errorf("explorer HTML must not reference external resource: %q", bad)
		}
	}

	// The payload must be valid JSON with the nodes actually present.
	var data Data
	if err := json.Unmarshal(mustRead(t, res.DataPath), &data); err != nil {
		t.Fatalf("data.json is not valid JSON: %v", err)
	}
	if len(data.Nodes) != 4 {
		t.Errorf("expected 4 nodes in payload, got %d", len(data.Nodes))
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

/* -- Payload correctness -- */

// TestBuildDataLinkDirection is the contract that the explorer's whole point
// depends on: outbound is what a page links TO, inbound is what links to it.
func TestBuildDataLinkDirection(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	data := BuildData(cfg, sampleInput(cfg).Graph, sampleInput(cfg).Meta, sampleInput(cfg).PageOutputs)

	index := nodeByID(t, data, "index")
	about := nodeByID(t, data, "about")

	if got, want := index.Outbound, []string{"about", "missing"}; !reflect.DeepEqual(got, want) {
		t.Errorf("index.Outbound = %v, want %v", got, want)
	}
	if len(index.Inbound) != 0 {
		t.Errorf("index.Inbound = %v, want empty", index.Inbound)
	}
	if got, want := about.Inbound, []string{"index"}; !reflect.DeepEqual(got, want) {
		t.Errorf("about.Inbound = %v, want %v", got, want)
	}
	if len(about.Outbound) != 0 {
		t.Errorf("about.Outbound = %v, want empty", about.Outbound)
	}
	if reflect.DeepEqual(index.Inbound, index.Outbound) {
		t.Error("index Inbound and Outbound are identical — edges are being read in one direction only")
	}
}

func TestBuildDataDeduplicatesEdges(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	if len(data.Edges) != 2 {
		t.Errorf("expected duplicate index->about edge to collapse, got %d edges: %v", len(data.Edges), data.Edges)
	}
	index := nodeByID(t, data, "index")
	seen := map[string]int{}
	for _, id := range index.Outbound {
		seen[id]++
	}
	if seen["about"] != 1 {
		t.Errorf("about appears %d times in index.Outbound, want 1", seen["about"])
	}
}

func TestBuildDataClassifiesNodes(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	cases := []struct {
		id     string
		render bool
		stub   bool
	}{
		{"index", true, false},
		{"about", true, false},
		{"notes", false, false},
		{"missing", false, true},
	}
	for _, c := range cases {
		n := nodeByID(t, data, c.id)
		if n.Render != c.render {
			t.Errorf("%s: Render = %v, want %v", c.id, n.Render, c.render)
		}
		if n.Stub != c.stub {
			t.Errorf("%s: Stub = %v, want %v", c.id, n.Stub, c.stub)
		}
	}
}

// TestBuildDataOrphanRule pins the homepage exemption: nothing links to the
// front page of a fresh site, and flagging it is noise.
func TestBuildDataOrphanRule(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	if nodeByID(t, data, "index").Orphan {
		t.Error("homepage must be exempt from the orphan rule")
	}
	if nodeByID(t, data, "about").Orphan {
		t.Error("about has an inbound link and must not be an orphan")
	}
	if !nodeByID(t, data, "notes").Orphan {
		t.Error("notes has no inbound links and should be flagged as an orphan")
	}
}

// TestBuildDataURLsAreSlugAware covers the case the client-side version could
// not: the public URL comes from the path the page was actually written to.
func TestBuildDataURLsAreSlugAware(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	in.PageOutputs["about"] = "custom-slug/index.html"
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	if got, want := nodeByID(t, data, "about").URL, "/custom-slug/"; got != want {
		t.Errorf("about.URL = %q, want %q (URL must follow the real output path, not the page id)", got, want)
	}
	if got, want := nodeByID(t, data, "index").URL, "/"; got != want {
		t.Errorf("index.URL = %q, want %q", got, want)
	}
	if url := nodeByID(t, data, "missing").URL; url != "" {
		t.Errorf("stub must not advertise a URL, got %q", url)
	}
	if url := nodeByID(t, data, "notes").URL; url != "" {
		t.Errorf("raw render:false page must not advertise a URL, got %q", url)
	}
}

// TestBuildDataURLsRespectSubpathDeploys covers GitHub Pages project sites,
// which config.md explicitly recommends. Root-relative links 404 there.
func TestBuildDataURLsRespectSubpathDeploys(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	cfg.SiteURL = "https://example.github.io/my-project"
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	if got, want := data.BasePath, "/my-project"; got != want {
		t.Errorf("BasePath = %q, want %q", got, want)
	}
	if got, want := nodeByID(t, data, "about").URL, "/my-project/about/"; got != want {
		t.Errorf("about.URL = %q, want %q", got, want)
	}
	if got, want := nodeByID(t, data, "index").URL, "/my-project/"; got != want {
		t.Errorf("index.URL = %q, want %q", got, want)
	}
}

func TestBuildDataCarriesFrontmatter(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	about := nodeByID(t, data, "about")
	if about.Author != "ada" || about.Date != "2026-01-02" || about.WordCount != 34 {
		t.Errorf("frontmatter not carried through: %+v", about)
	}
	if got, want := about.Tags, []string{"docs"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Tags = %v, want %v", got, want)
	}
	if got, want := about.Categories, []string{"meta"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Categories = %v, want %v", got, want)
	}
}

func TestBuildDataNodesAreSorted(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)

	for i := 1; i < len(data.Nodes); i++ {
		if data.Nodes[i-1].ID > data.Nodes[i].ID {
			t.Fatalf("nodes not sorted by id: %q before %q", data.Nodes[i-1].ID, data.Nodes[i].ID)
		}
	}
}

func TestTitleFromID(t *testing.T) {
	cases := []struct{ id, want string }{
		{"index", "Index"},
		{"docs/getting-started", "Getting Started"},
		{"docs/some_page.md", "Some Page"},
		{"a-b--c", "A B C"},
		{"", ""},
	}
	for _, c := range cases {
		if got := TitleFromID(c.id); got != c.want {
			t.Errorf("TitleFromID(%q) = %q, want %q", c.id, got, c.want)
		}
	}
}

func TestBuildDataFallsBackToDerivedTitle(t *testing.T) {
	cfg := writeMinimumSite(t, t.TempDir())
	in := sampleInput(cfg)
	// "missing" has no frontmatter entry at all.
	data := BuildData(cfg, in.Graph, in.Meta, in.PageOutputs)
	if got, want := nodeByID(t, data, "missing").Title, "Missing"; got != want {
		t.Errorf("fallback title = %q, want %q", got, want)
	}
}

/* -- Determinism -- */

func TestWriteIsByteDeterministicForSameInputs(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	cfgA := writeMinimumSite(t, dirA)
	cfgB := writeMinimumSite(t, dirB)

	inA := sampleInput(cfgA)
	inB := sampleInput(cfgB)
	resA, err := Write(inA)
	if err != nil {
		t.Fatal(err)
	}
	resB, err := Write(inB)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(mustRead(t, resA.IndexPath), mustRead(t, resB.IndexPath)) {
		t.Error("graph/index.html differs between two builds of identical input")
	}
	if !reflect.DeepEqual(mustRead(t, resA.DataPath), mustRead(t, resB.DataPath)) {
		t.Error("graph/data.json differs between two builds of identical input")
	}
}

/* -- Page metadata -- */

func TestExplorerFooterReflectsNodeCountAndThreshold(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatal(err)
	}
	html := string(mustRead(t, res.IndexPath))
	if !strings.Contains(html, "4 nodes.") {
		t.Errorf("footer should report the payload node count; got: %.400s", html)
	}
	if !strings.Contains(html, "500 nodes") {
		t.Errorf("footer should surface the large-site threshold")
	}
}

func TestExplorerWithSiteURLProducesCanonical(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	cfg.SiteURL = "https://example.com"
	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatal(err)
	}
	html := string(mustRead(t, res.IndexPath))
	if !strings.Contains(html, `<link rel="canonical" href="https://example.com/graph/">`) {
		t.Errorf("expected canonical link for configured siteurl; got: %.400s", html)
	}
}

func TestExplorerWithoutSiteURLUsesRelativeLinks(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatal(err)
	}
	html := string(mustRead(t, res.IndexPath))
	if strings.Contains(html, "rel=\"canonical\"") {
		t.Errorf("no canonical link should be emitted when siteurl is unset")
	}
}

func TestExplorerPagePathIsSafe(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	res, err := Write(sampleInput(cfg))
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(filepath.Clean(cfg.OutputDir), res.IndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(rel, "..") {
		t.Errorf("explorer page escapes the output directory: %s", rel)
	}
}

func TestIndexPathAndRelAreStable(t *testing.T) {
	dir := t.TempDir()
	cfg := writeMinimumSite(t, dir)
	if got, want := IndexPath(cfg.OutputDir), filepath.Join(cfg.OutputDir, "graph", "index.html"); got != want {
		t.Errorf("IndexPath = %q, want %q", got, want)
	}
	if got, want := DataPath(cfg.OutputDir), filepath.Join(cfg.OutputDir, "graph", "data.json"); got != want {
		t.Errorf("DataPath = %q, want %q", got, want)
	}
	if IndexRel() != "graph/index.html" {
		t.Errorf("IndexRel = %q, want graph/index.html", IndexRel())
	}
	if DataRel() != "data.json" {
		t.Errorf("DataRel = %q, want data.json", DataRel())
	}
	if AssetRel() != "../assets/graph/explorer" {
		t.Errorf("AssetRel = %q, want ../assets/graph/explorer", AssetRel())
	}
}

/* -- Shipped asset bundle --
   These read the real files from the repo. repoRoot() fails rather than skips,
   so a path mistake surfaces as a red test instead of silent green. */

func TestShippedJSConsumesPayloadAndDoesNotRederiveIt(t *testing.T) {
	js := readRepoFile(t, "assets/graph/explorer.js")

	for _, needle := range []string{"data.json", "suppressed", "revealGraph", "applyFocusMode", "history.replaceState"} {
		if !strings.Contains(js, needle) {
			t.Errorf("explorer.js missing required anchor: %s", needle)
		}
	}
	// The client must not go back to joining the raw artifacts itself; that
	// duplication is what produced the inverted link direction.
	for _, banned := range []string{"backlinks.json", "meta.json", "../graph.json"} {
		if strings.Contains(js, banned) {
			t.Errorf("explorer.js must consume the prebuilt payload, not %s", banned)
		}
	}
	if strings.Contains(js, "outbound.push") {
		t.Error("explorer.js must not rebuild adjacency client-side; the generator owns link direction")
	}
}

func TestShippedAssetsHaveNoThirdPartyReferences(t *testing.T) {
	for _, rel := range []string{"assets/graph/explorer.js", "assets/graph/explorer.css"} {
		body := readRepoFile(t, rel)
		// Note: the SVG namespace literal http://www.w3.org/2000/svg is not a
		// network reference, so this checks fetchable markers specifically.
		for _, bad := range []string{"https://cdn.", "//cdn.", "googleapis.com", "jsdelivr.net", "unpkg.com", "https://fonts.", "src=\"http", "url(http"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s must not reference external resource: %q", rel, bad)
			}
		}
	}
}

func TestShippedCSSBacksItsAccessibilityClasses(t *testing.T) {
	css := readRepoFile(t, "assets/graph/explorer.css")
	tmpl := readRepoFile(t, "internal/graphexplorer/assets/template.html")

	if !strings.Contains(css, "prefers-reduced-motion") {
		t.Error("CSS missing reduced-motion media query")
	}
	if !strings.Contains(css, ".kgx-root") {
		t.Error("CSS missing root selector")
	}
	// Every kgx- class the template relies on must actually exist in the
	// stylesheet, or the markup is decorative only.
	for _, class := range []string{".kgx-sr-only", ".kgx-skip-link", ".kgx-skip-link:focus", ".kgx-detail-close"} {
		if !strings.Contains(css, class) {
			t.Errorf("CSS missing %s, so the template class does nothing", class)
		}
	}
	if strings.Contains(tmpl, "focus:not-sr-only") {
		t.Error("template still uses Tailwind-style class names with no backing stylesheet")
	}
	if strings.Contains(tmpl, `role="img"`) {
		t.Error(`SVG must not use role="img": it prunes the focusable nodes from the accessibility tree`)
	}
}
