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
	_ = os.MkdirAll(tmplDir, 0755)
	tmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(tmplPath, []byte("Hello {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: tmplPath}
	p := page.Page{Title: "World", Content: template.HTML("")}

	renderer := New(filepath.Dir(cfg.Template))
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
	_ = os.MkdirAll(tmplDir, 0755)

	defaultTmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(defaultTmplPath, []byte("Default: {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	customTmplPath := filepath.Join(tmplDir, "custom.html")
	err = os.WriteFile(customTmplPath, []byte("Custom: {{.Title}}"), 0600)
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
	defer func() { _ = os.Chdir(origWd) }()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no layout specified uses default", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "out_default.html")
		p := page.Page{Title: "Page One"}

		renderer := New(filepath.Dir(cfg.Template))
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

		renderer := New(filepath.Dir(cfg.Template))
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
		renderer := New(filepath.Dir(cfg.Template))

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

func TestDiscoverLayouts(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "layout1.html"), []byte("<html></html>"), 0600); err != nil {
		t.Fatalf("Failed to write layout1.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "not-a-layout.txt"), []byte("text"), 0600); err != nil {
		t.Fatalf("Failed to write not-a-layout.txt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "partials"), 0755); err != nil {
		t.Fatalf("Failed to create partials dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "partials", "partial.html"), []byte("<div></div>"), 0600); err != nil {
		t.Fatalf("Failed to write partial.html: %v", err)
	}

	allowlist, err := DiscoverLayouts(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverLayouts failed: %v", err)
	}

	if len(allowlist) != 1 {
		t.Fatalf("Expected 1 layout, got %d", len(allowlist))
	}
	if !allowlist["layout1"] {
		t.Errorf("Expected layout1 to be in allowlist")
	}
}

func TestHTMLWithPartial(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.html")

	// Setup fake templates directory structure
	tmplDir := filepath.Join(tmpDir, "templates")
	_ = os.MkdirAll(tmplDir, 0755)

	// Layout using a partial
	tmplPath := filepath.Join(tmplDir, "layout.html")
	err := os.WriteFile(tmplPath, []byte("Layout: {{template \"partials/footer.html\" .}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Partial file
	partialsDir := filepath.Join(tmplDir, "partials")
	_ = os.MkdirAll(partialsDir, 0755)
	partialPath := filepath.Join(partialsDir, "footer.html")
	err = os.WriteFile(partialPath, []byte("Footer - {{.Title}}"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// We need to change the working directory so findPartials can locate "templates"
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Template: tmplPath}
	p := page.Page{Title: "World", Content: template.HTML("")}

	renderer := New(filepath.Dir(cfg.Template))
	err = renderer.HTML(cfg, p, "", outPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	expected := "Layout: Footer - World"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}
