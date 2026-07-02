package taxonomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/render"
)

func TestGenerateTags(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "public")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	_ = os.WriteFile(tmplPath, []byte("{{.Content}}"), 0600)

	cfg := config.Config{
		OutputDir: outDir,
		Template:  tmplPath,
	}

	renderTrue := true
	fileMap := map[string]*content.FileMeta{
		"post1.md": {Title: "Post 1", Tags: []string{"go", "web"}, Render: &renderTrue},
		"post2.md": {Title: "Post 2", Tags: []string{"go"}, Render: &renderTrue},
	}

	renderer := render.New(tmplDir)
	p := bluemonday.UGCPolicy()

	err := GenerateTags(cfg, fileMap, renderer, p)
	if err != nil {
		t.Fatalf("GenerateTags failed: %v", err)
	}

	// Check if go tag page was created
	goTagPath := filepath.Join(outDir, "tags", "go", "index.html")
	b, err := os.ReadFile(goTagPath)
	if err != nil {
		t.Fatalf("expected tags/go/index.html to exist: %v", err)
	}
	html := string(b)
	if !strings.Contains(html, "<h2>Tag: go</h2>") {
		t.Errorf("expected tag title, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post1/"`) {
		t.Errorf("expected link to post1, got: %s", html)
	}
	if !strings.Contains(html, `href="../../post2/"`) {
		t.Errorf("expected link to post2, got: %s", html)
	}
}
