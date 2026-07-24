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
)

func stubTestConfig(t *testing.T) config.Config {
	t.Helper()
	tempDir := t.TempDir()
	templatePath := filepath.Join(tempDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte(`<html><body>{{.Content}}</body></html>`), 0600); err != nil {
		t.Fatal(err)
	}
	return config.Config{OutputDir: tempDir, Template: templatePath}
}

// Stubs are written last, with os.Create. A claimed path belongs to another
// writer, so the stub must be skipped rather than truncating what is there.
func TestGenerateStubs_SkipsClaimedPaths(t *testing.T) {
	cfg := stubTestConfig(t)

	taken := filepath.Join(cfg.OutputDir, "taken", "index.html")
	if err := os.MkdirAll(filepath.Dir(taken), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taken, []byte("REAL_GENERATED_PAGE"), 0600); err != nil {
		t.Fatal(err)
	}

	missingFiles := map[string][]string{
		"taken.md": {"parent.md"},
		"free.md":  {"parent.md"},
	}
	g := &graph.Graph{Nodes: make(map[string]graph.Node)}
	p := bluemonday.UGCPolicy()

	var asked []string
	claim := func(_, relOut string) (string, bool) {
		asked = append(asked, relOut)
		if relOut == "taken/index.html" {
			return "the generated taxonomy page", false
		}
		return "", true
	}

	if err := GenerateStubs(cfg, cfg, missingFiles, g, p, map[string]*content.FileMeta{}, claim); err != nil {
		t.Fatalf("GenerateStubs() error = %v", err)
	}

	if len(asked) != 2 {
		t.Errorf("claim was consulted for %v, want both stub paths", asked)
	}

	got, err := os.ReadFile(taken)
	if err != nil {
		t.Fatalf("read claimed path: %v", err)
	}
	if string(got) != "REAL_GENERATED_PAGE" {
		t.Errorf("claimed path = %q, want it untouched", got)
	}

	free, err := os.ReadFile(filepath.Join(cfg.OutputDir, "free", "index.html"))
	if err != nil {
		t.Fatalf("read unclaimed stub: %v", err)
	}
	if !strings.Contains(string(free), "Under Construction") {
		t.Errorf("free/index.html = %q, want a stub page", free)
	}

	// The link that produced the skipped stub is already an edge in the graph,
	// so its node still has to exist or that edge dangles.
	if node, ok := g.Nodes["taken"]; !ok || node.Type != "stub" {
		t.Errorf("g.Nodes[\"taken\"] = %v, ok=%v, want a stub node even when the page is skipped", node, ok)
	}
}

// A slug that the renderer discards must be discarded here too, or the stub
// links back to a page that was never written at that address.
func TestGenerateStubs_ParentLinkIgnoresUnusableSlug(t *testing.T) {
	cfg := stubTestConfig(t)

	fileMap := map[string]*content.FileMeta{
		"parent.md": {Slug: "my.page"},
	}
	missingFiles := map[string][]string{"ghost.md": {"parent.md"}}
	g := &graph.Graph{Nodes: make(map[string]graph.Node)}

	if err := GenerateStubs(cfg, cfg, missingFiles, g, bluemonday.UGCPolicy(), fileMap, nil); err != nil {
		t.Fatalf("GenerateStubs() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(cfg.OutputDir, "ghost", "index.html"))
	if err != nil {
		t.Fatalf("read stub: %v", err)
	}
	if strings.Contains(string(got), "my.page") {
		t.Errorf("stub = %q, want no link to the discarded slug my.page", got)
	}
	if !strings.Contains(string(got), "../parent/") {
		t.Errorf("stub = %q, want a link to the parent's real output path ../parent/", got)
	}
}
