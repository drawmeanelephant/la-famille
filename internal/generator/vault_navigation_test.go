package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestBuildStubAndBacklinkNavigation(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	templateDir := filepath.Join(root, "templates")
	outputDir := filepath.Join(root, "public")
	for _, dir := range []string{contentDir, templateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	template := `<!doctype html><html><head><title>{{.Title}}</title></head><body>
<nav>{{range .Site.SiteLinks}}<a href="{{.URL}}">{{.Label}}</a>{{end}}</nav>
<main>{{.Content}}</main></body></html>`
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte(template), 0600); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("index.md", "---\ntitle: Home\n---\n# Home\n\n[[First Note]] [[Second Note]]\n")
	write("source.md", "---\ntitle: Source Note\n---\n# Source\n\n[[First Note]]\n")

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = filepath.Join(templateDir, "layout.html")

	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build() error = %v", err)
	}

	index := readOutput(t, cfg, unresolvedNotesOutput)
	for _, want := range []string{
		`href="../first-note/" rel="nofollow">First Note</a>`,
		`href="../second-note/" rel="nofollow">Second Note</a>`,
		`href="/unresolved-notes/">Unresolved Notes</a>`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("unresolved-notes index missing %q:\n%s", want, index)
		}
	}

	stub := readOutput(t, cfg, "first-note/index.html")
	for _, want := range []string{
		`<h3 id="backlinks-heading">Linked from</h3>`,
		`href="../" rel="nofollow">Home</a>`,
		`href="../source/" rel="nofollow">Source Note</a>`,
	} {
		if !strings.Contains(stub, want) {
			t.Errorf("generated note stub missing backlink %q:\n%s", want, stub)
		}
	}

	write("first-note.md", "---\ntitle: First Note\n---\n# First Note\n\nReal note content.\n")
	if _, err := Build(cfg); err != nil {
		t.Fatalf("second Build() error = %v", err)
	}

	index = readOutput(t, cfg, unresolvedNotesOutput)
	if strings.Contains(index, `href="../first-note/" rel="nofollow">First Note</a>`) {
		t.Errorf("resolved note remains in unresolved-notes index:\n%s", index)
	}
	if !strings.Contains(index, `href="../second-note/" rel="nofollow">Second Note</a>`) {
		t.Errorf("remaining stub is missing from unresolved-notes index:\n%s", index)
	}

	realNote := readOutput(t, cfg, "first-note/index.html")
	if !strings.Contains(realNote, "Real note content.") {
		t.Fatalf("filled stub did not render as a normal page:\n%s", realNote)
	}
	for _, want := range []string{
		`<section class="backlinks-panel"`,
		`href="../">Home</a>`,
		`href="../source/">Source Note</a>`,
	} {
		if !strings.Contains(realNote, want) {
			t.Errorf("normal note missing inbound link %q:\n%s", want, realNote)
		}
	}

	var graphData struct {
		Nodes map[string]struct {
			Type    string `json:"type"`
			Missing bool   `json:"missing"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &graphData); err != nil {
		t.Fatalf("parse graph.json: %v", err)
	}
	if node := graphData.Nodes["first-note"]; node.Type != "page" || node.Missing {
		t.Errorf("filled note graph node = %+v, want normal page", node)
	}

	var explorerData struct {
		Nodes []struct {
			ID   string `json:"id"`
			Stub bool   `json:"stub"`
		} `json:"nodes"`
	}
	explorerBytes, err := os.ReadFile(filepath.Join(outputDir, "graph", "data.json"))
	if err != nil {
		t.Fatalf("read graph explorer data: %v", err)
	}
	if err := json.Unmarshal(explorerBytes, &explorerData); err != nil {
		t.Fatalf("parse graph explorer data: %v", err)
	}
	for _, node := range explorerData.Nodes {
		if node.ID == "first-note" {
			if node.Stub {
				t.Errorf("filled note still classified as a stub: %+v", node)
			}
			return
		}
	}
	t.Fatal("graph explorer has no node for the filled note")
}
