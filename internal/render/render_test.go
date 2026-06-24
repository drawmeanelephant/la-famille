package render

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
)

func TestHTMLPage(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "render_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy template file
	tmplPath := filepath.Join(tmpDir, "layout.html")
	tmplContent := "Title: {{.Title}}, Content: {{.Content}}"
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}

	cfg := config.Config{
		Template: tmplPath,
	}

	meta := &content.FileMeta{
		Author: "Test Author",
	}

	outPath := filepath.Join(tmpDir, "output.html")
	sanitizedHTML := []byte("<p>Hello World</p>")
	title := "Test Title"

	if err := HTMLPage(cfg, meta, title, sanitizedHTML, outPath); err != nil {
		t.Fatalf("HTMLPage failed: %v", err)
	}

	outContent, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expectedContent := "Title: Test Title, Content: <p>Hello World</p>"
	if string(bytes.TrimSpace(outContent)) != expectedContent {
		t.Errorf("expected output %q, got %q", expectedContent, string(bytes.TrimSpace(outContent)))
	}
}
