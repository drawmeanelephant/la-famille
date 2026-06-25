package render

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/page"
)

func TestHTML(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.html")

	// Setup a fake template
	tmplDir := filepath.Join(tmpDir, "templates")
	os.MkdirAll(tmplDir, 0755)
	tmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(tmplPath, []byte("Hello {{.Title}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: tmplPath}
	p := page.Page{Title: "World", Content: template.HTML("")}

	err = HTML(cfg, p, "", outPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", string(content))
	}
}
