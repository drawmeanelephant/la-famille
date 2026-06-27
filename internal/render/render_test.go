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

	renderer := New()
	err = renderer.HTML(cfg, p, "", outPath)
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

func TestHTMLLayoutSelection(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup fake templates
	tmplDir := filepath.Join(tmpDir, "templates")
	os.MkdirAll(tmplDir, 0755)

	defaultTmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(defaultTmplPath, []byte("Default: {{.Title}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	customTmplPath := filepath.Join(tmplDir, "custom.html")
	err = os.WriteFile(customTmplPath, []byte("Custom: {{.Title}}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: defaultTmplPath}

	// Temporarily change directory to tmpDir so that filepath.Join("templates", layout+".html")
	// resolves to our mocked templates directory.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no layout specified uses default", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "out_default.html")
		p := page.Page{Title: "Page One"}

		renderer := New()
		err = renderer.HTML(cfg, p, "", outPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "Default: Page One" {
			t.Errorf("expected 'Default: Page One', got '%s'", string(content))
		}
	})

	t.Run("layout specified uses custom", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "out_custom.html")
		p := page.Page{Title: "Page Two"}

		renderer := New()
		err = renderer.HTML(cfg, p, "custom", outPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "Custom: Page Two" {
			t.Errorf("expected 'Custom: Page Two', got '%s'", string(content))
		}
	})

	t.Run("back-to-back renders using different layouts", func(t *testing.T) {
		renderer := New()

		// First render
		outPath1 := filepath.Join(tmpDir, "out_bb_1.html")
		p1 := page.Page{Title: "First"}
		err = renderer.HTML(cfg, p1, "", outPath1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Second render
		outPath2 := filepath.Join(tmpDir, "out_bb_2.html")
		p2 := page.Page{Title: "Second"}
		err = renderer.HTML(cfg, p2, "custom", outPath2)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		content1, err := os.ReadFile(outPath1)
		if err != nil {
			t.Fatal(err)
		}
		if string(content1) != "Default: First" {
			t.Errorf("expected 'Default: First', got '%s'", string(content1))
		}

		content2, err := os.ReadFile(outPath2)
		if err != nil {
			t.Fatal(err)
		}
		if string(content2) != "Custom: Second" {
			t.Errorf("expected 'Custom: Second', got '%s'", string(content2))
		}
	})
}
