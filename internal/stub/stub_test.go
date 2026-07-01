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

func TestRelPathFromTo(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		target   string
		expected string
	}{
		{
			name:     "same directory",
			base:     "dir/missing.md",
			target:   "dir/parent.html",
			expected: "parent.html",
		},
		{
			name:     "target in parent directory",
			base:     "dir/subdir/missing.md",
			target:   "dir/parent.html",
			expected: "../parent.html",
		},
		{
			name:     "target in child directory",
			base:     "dir/missing.md",
			target:   "dir/subdir/parent.html",
			expected: "subdir/parent.html",
		},
		{
			name:     "different branch",
			base:     "dir1/missing.md",
			target:   "dir2/parent.html",
			expected: "../dir2/parent.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := RelPathFromTo(tt.base, tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rel != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rel)
			}
		})
	}
}

func TestGenerateStubs(t *testing.T) {
	// Setup a temporary directory for output
	tempDir, err := os.MkdirTemp("", "stub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy template file since GenerateStubs requires it
	templatePath := filepath.Join(tempDir, "layout.html")
	templateContent := `<html><body>{{.Content}}</body></html>`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write dummy template: %v", err)
	}

	cfg := config.Config{
		OutputDir: tempDir,
		Template:  templatePath,
	}

	missingFiles := map[string][]string{
		"missing.md":      {"parent1.md"},
		"dir/missing2.md": {"parent2.md", "dir/parent3.md"},
	}

	g := &graph.Graph{
		Nodes: make(map[string]graph.Node),
	}

	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Globally()

	// Execute GenerateStubs

	fileMap := make(map[string]*content.FileMeta)
	if err := GenerateStubs(cfg, missingFiles, g, p, fileMap); err != nil {
		t.Fatalf("unexpected error from GenerateStubs: %v", err)
	}

	// Verify graph node updates
	if node, ok := g.Nodes["missing"]; !ok || node.Type != "stub" {
		t.Errorf("expected missing node in graph with type 'stub', got %v", node)
	}
	if node, ok := g.Nodes["dir/missing2"]; !ok || node.Type != "stub" {
		t.Errorf("expected dir/missing2 node in graph with type 'stub', got %v", node)
	}

	// Verify output files are created with expected content
	checkFile := func(relPath string, expectedSubstrings []string) {
		fullPath := filepath.Join(tempDir, relPath)
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read expected stub file %q: %v", fullPath, err)
		}
		contentStr := string(contentBytes)
		for _, substr := range expectedSubstrings {
			if !strings.Contains(contentStr, substr) {
				t.Errorf("file %q did not contain expected substring %q. Content:\n%s", relPath, substr, contentStr)
			}
		}
	}

	checkFile("missing/index.html", []string{
		"🌱 This page is a stub",
		"alert alert-warning",
		"menu bg-base-100",
		`<a href="../parent1/" rel="nofollow">parent1.md</a>`,
	})

	checkFile("dir/missing2/index.html", []string{
		"🌱 This page is a stub",
		"alert alert-warning",
		"menu bg-base-100",
		`<a href="../../parent2/" rel="nofollow">parent2.md</a>`,
		`<a href="../parent3/" rel="nofollow">dir/parent3.md</a>`,
	})
}
