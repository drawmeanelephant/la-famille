package generator

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
    "strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/yuin/goldmark"
)

func TestBuild_MarkdownConversionError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	os.WriteFile(templatePath, []byte("{{.Content}}"), 0644)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	os.WriteFile(filepath.Join(contentDir, "test1.md"), []byte("# Hello 1"), 0644)
	os.WriteFile(filepath.Join(contentDir, "test2.md"), []byte("# Hello 2"), 0644)

	// Mock convertMarkdown to always fail
	originalConvert := convertMarkdown
	defer func() { convertMarkdown = originalConvert }()

	convertMarkdown = func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
		return errors.New("simulated conversion error")
	}

	res, err := Build(cfg)
	if err == nil {
		t.Fatalf("expected error from Build, got nil")
	}

	if !strings.Contains(err.Error(), "simulated conversion error") {
		t.Errorf("expected error string to contain 'simulated conversion error', got: %v", err)
	}

	if res.ErrorCount != 2 {
		t.Errorf("expected 2 errors, got %d", res.ErrorCount)
	}
}

func TestBuild_Success(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	os.WriteFile(templatePath, []byte("{{.Content}}"), 0644)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	os.WriteFile(filepath.Join(contentDir, "test.md"), []byte("# Hello"), 0644)

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
