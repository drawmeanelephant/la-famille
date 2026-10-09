package stub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/render"
)

// TestGenerateStubsAppliesBasePath guards #636: stubs rendered through a
// hand-rolled template skipped applyBasePath, so under a subpath siteurl a
// stub emitted href="/assets/..." links that 404 on the real deploy, carried
// no la-famille-base-path meta, and never received the watch-mode livereload
// script. Stubs must go through the same renderer as normal pages.
func TestGenerateStubsAppliesBasePath(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "layout.html")
	templateContent := `<html><head><link href="/assets/site.css" rel="stylesheet"></head><body>{{.Content}}<a href="/tags/">tags</a></body></html>`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{OutputDir: tempDir, Template: templatePath, SiteURL: "https://example.com/blog"}
	siteCfg := cfg

	missingFiles := map[string][]string{"missing.md": {"parent.md"}}
	g := &graph.Graph{Nodes: make(map[string]graph.Node)}
	r := render.New(filepath.Dir(templatePath))

	if err := GenerateStubs(cfg, siteCfg, missingFiles, nil, g, bluemonday.UGCPolicy(), map[string]*content.FileMeta{}, r, nil); err != nil {
		t.Fatalf("GenerateStubs() error = %v", err)
	}

	out, err := os.ReadFile(filepath.Join(tempDir, "missing", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(out)
	if !strings.Contains(page, `<meta name="la-famille-base-path" content="/blog">`) {
		t.Errorf("stub lacks the base-path meta:\n%s", page)
	}
	if !strings.Contains(page, `href="/blog/assets/site.css"`) {
		t.Errorf("stub asset link was not rebased under /blog:\n%s", page)
	}
	if !strings.Contains(page, `href="/blog/tags/"`) {
		t.Errorf("stub tag link was not rebased under /blog:\n%s", page)
	}
}

// In watch mode the stub page must receive the same livereload injection as
// rendered pages.
func TestGenerateStubsInjectsLiveReloadInWatchMode(t *testing.T) {
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte(`<html><body>{{.Content}}</body></html>`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{OutputDir: tempDir, Template: templatePath, WatchMode: true}
	g := &graph.Graph{Nodes: make(map[string]graph.Node)}
	r := render.New(filepath.Dir(templatePath))

	if err := GenerateStubs(cfg, cfg, map[string][]string{"gone.md": {"parent.md"}}, nil, g, bluemonday.UGCPolicy(), nil, r, nil); err != nil {
		t.Fatalf("GenerateStubs() error = %v", err)
	}
	out, err := os.ReadFile(filepath.Join(tempDir, "gone", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "EventSource") {
		t.Errorf("watch-mode stub lacks the livereload script:\n%s", out)
	}
}
