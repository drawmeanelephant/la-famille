package generator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/yuin/goldmark"
)

func TestBuild_MarkdownConversionError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	_ = os.WriteFile(filepath.Join(contentDir, "test1.md"), []byte("# Hello 1"), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "test2.md"), []byte("# Hello 2"), 0600)

	// Mock convertMarkdown to always fail
	originalConvert := getConvertMarkdown()
	defer func() { setConvertMarkdown(originalConvert) }()

	setConvertMarkdown(func(_ goldmark.Markdown, _ []byte, _ *bytes.Buffer) error {
		return errors.New("simulated conversion error")
	})

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

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	_ = os.WriteFile(filepath.Join(contentDir, "test.md"), []byte("# Hello"), 0600)

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeneratorSEO(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outDir := filepath.Join(tempDir, "public")
	assetDir := filepath.Join(tempDir, "assets")
	ragDir := filepath.Join(tempDir, "rag-archive")
	tmplDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(outDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)
	_ = os.MkdirAll(ragDir, 0755)
	_ = os.MkdirAll(tmplDir, 0755)

	tmplPath := filepath.Join(tmplDir, "layout.html")
	tmplContent := `<!DOCTYPE html><html><head><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><meta property="og:image" content="{{.Image}}"></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0600); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	mdContent := `---
title: Test SEO
description: "Test SEO Description"
image: "/images/test-seo.png"
---
# Hello SEO`
	mdPath := filepath.Join(contentDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0600); err != nil {
		t.Fatalf("failed to write markdown file: %v", err)
	}

	cfg := config.Config{
		SiteName:           "Test Site",
		Template:           tmplPath,
		ContentDir:         contentDir,
		OutputDir:          outDir,
		AssetDir:           assetDir,
		RagDir:             ragDir,
		Theme:              "retro",
		Port:               8080,
		DefaultDescription: "Default Desc",
		DefaultOGImage:     "/default.png",
	}

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	outHTMLPath := filepath.Join(outDir, "test", "index.html")
	outBytes, err := os.ReadFile(outHTMLPath)
	if err != nil {
		t.Fatalf("failed to read output HTML: %v", err)
	}

	outHTML := string(outBytes)

	expectedDesc := `<meta name="description" content="Test SEO Description">`
	if !strings.Contains(outHTML, expectedDesc) {
		t.Errorf("output HTML missing expected description meta tag.\nGot: %s", outHTML)
	}

	expectedImage := `<meta property="og:image" content="/images/test-seo.png">`
	if !strings.Contains(outHTML, expectedImage) {
		t.Errorf("output HTML missing expected image meta tag.\nGot: %s", outHTML)
	}
}

func BenchmarkBuild(b *testing.B) {
	tempDir := b.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	// Create 1000 dummy files
	for i := 0; i < 1000; i++ {
		content := []byte("# Hello Benchmark\nThis is a [link](test0.md) to another page.")
		_ = os.WriteFile(filepath.Join(contentDir, fmt.Sprintf("test%d.md", i)), content, 0600)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Build(cfg)
	}
}

func TestCollisionDeterminism(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "templates"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "templates", "layout.html"), []byte("{{.Content}}"), 0600)

	_ = os.WriteFile(filepath.Join(contentDir, "alpha.md"), []byte("---\nslug: shared\n---\nAlpha"), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "beta.md"), []byte("---\nslug: shared\n---\nBeta"), 0600)

	cfg := config.Config{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		Template:   filepath.Join(tempDir, "templates", "layout.html"),
	}

	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Expected build to fail due to output path collision, but it succeeded")
	}
	if !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("Expected collision error, got: %v", err)
	}
}

func TestRaceRegression(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "templates"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "templates", "layout.html"), []byte("{{.Content}}"), 0600)

	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("---\ntitle: Page %d\n---\n[Link to missing](missing.md)\n[Link to next](page%d.md)", i, (i+1)%50)
		_ = os.WriteFile(filepath.Join(contentDir, fmt.Sprintf("page%d.md", i)), []byte(content), 0600)
	}

	cfg := config.Config{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		Template:   filepath.Join(tempDir, "templates", "layout.html"),
	}

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	b1, err := os.ReadFile(filepath.Join(outputDir, "graph.json"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = Build(cfg)
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	b2, err := os.ReadFile(filepath.Join(outputDir, "graph.json"))
	if err != nil {
		t.Fatal(err)
	}

	if string(b1) != string(b2) {
		t.Errorf("Graph JSON changed between builds: %s != %s", string(b1), string(b2))
	}
}

func TestErrorOrdering(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")
	templatePath := filepath.Join(templateDir, "layout.html")

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0o600); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"z_fail.md": "Z_FAIL",
		"a_fail.md": "A_FAIL",
		"m_fail.md": "M_FAIL",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Config{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		Template:   templatePath,
	}

	originalConvert := getConvertMarkdown()
	t.Cleanup(func() { setConvertMarkdown(originalConvert) })

	setConvertMarkdown(func(_ goldmark.Markdown, source []byte, _ *bytes.Buffer) error {
		return fmt.Errorf("forced conversion failure: %s", source)
	})

	result, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want conversion failures")
	}
	if result.ErrorCount != 3 {
		t.Fatalf("ErrorCount = %d, want 3", result.ErrorCount)
	}

	got := err.Error()
	wantOrder := []string{
		"error converting a_fail.md: forced conversion failure: A_FAIL",
		"error converting m_fail.md: forced conversion failure: M_FAIL",
		"error converting z_fail.md: forced conversion failure: Z_FAIL",
	}

	previous := -1
	for _, want := range wantOrder {
		position := strings.Index(got, want)
		if position < 0 {
			t.Fatalf("combined error missing %q:\n%s", want, got)
		}
		if position <= previous {
			t.Fatalf("errors are not in deterministic source-path order:\n%s", got)
		}
		previous = position
	}
}
