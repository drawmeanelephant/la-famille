package generator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestBuildFailurePreservesExistingOutput(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")
	templatePath := filepath.Join(templateDir, "layout.html")

	for _, dir := range []string{contentDir, outputDir, templateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "new.md"), []byte("# New page"), 0600); err != nil {
		t.Fatal(err)
	}
	oldOutput := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(oldOutput, []byte("known good output"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	originalConvert := getConvertMarkdown()
	t.Cleanup(func() { setConvertMarkdown(originalConvert) })
	setConvertMarkdown(func(_ goldmark.Markdown, _ []byte, _ *bytes.Buffer) error {
		return errors.New("simulated conversion failure")
	})

	if _, err := Build(cfg); err == nil {
		t.Fatal("Build() error = nil, want conversion failure")
	}

	got, err := os.ReadFile(oldOutput)
	if err != nil {
		t.Fatalf("read preserved output: %v", err)
	}
	if string(got) != "known good output" {
		t.Fatalf("existing output changed after failed build: %q", got)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "new", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("failed build wrote new page to existing output: %v", err)
	}
}

func TestBuildRemovesOutputForDeletedSource(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")
	templatePath := filepath.Join(templateDir, "layout.html")

	for _, dir := range []string{contentDir, templateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"keep.md": "# Keep",
		"gone.md": "# Gone",
	} {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath
	if _, err := Build(cfg); err != nil {
		t.Fatalf("first Build() error: %v", err)
	}

	if err := os.Remove(filepath.Join(contentDir, "gone.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("second Build() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outputDir, "keep", "index.html")); err != nil {
		t.Fatalf("kept page missing after rebuild: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "gone", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("deleted source output remains after rebuild: %v", err)
	}
}

func TestBuild_UsesAndInvalidatesCache(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "public")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	contentPath := filepath.Join(contentDir, "page.md")
	if err := os.WriteFile(contentPath, []byte("# first"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ContentDir, cfg.Template, cfg.OutputDir = contentDir, templatePath, outputDir
	cfg.ProjectRoot = tempDir
	if err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("ignored.tmp\n"), 0600); err != nil {
		t.Fatal(err)
	}

	var conversions atomic.Int32
	original := getConvertMarkdown()
	defer func() { setConvertMarkdown(original) }()
	setConvertMarkdown(func(md goldmark.Markdown, source []byte, w *bytes.Buffer) error {
		conversions.Add(1)
		return md.Convert(source, w)
	})
	resFirst, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resFirst.CacheHit {
		t.Error("expected initial build to be a cache miss")
	}
	first := conversions.Load()
	resSecond, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !resSecond.CacheHit {
		t.Error("expected repeat build to be a cache hit")
	}
	if got := conversions.Load(); got != first {
		t.Fatalf("cache miss rebuilt unchanged content: conversions %d -> %d", first, got)
	}
	staging, err := filepath.Glob(filepath.Join(tempDir, ".public.staging-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(staging) != 0 {
		t.Fatalf("cache hit leaked staging directories: %v", staging)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("changed.tmp\n"), 0600); err != nil {
		t.Fatal(err)
	}
	resGitignore, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resGitignore.CacheHit {
		t.Error("expected .gitignore change to cause a cache miss")
	}
	if conversions.Load() <= first {
		t.Fatal(".gitignore change did not invalidate cache")
	}
	cfg.Theme = "dark"
	resCfg, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resCfg.CacheHit {
		t.Error("expected config change to cause a cache miss")
	}
	if conversions.Load() <= first {
		t.Fatal("output-affecting config change did not invalidate cache")
	}
	if err := os.WriteFile(contentPath, []byte("# changed"), 0600); err != nil {
		t.Fatal(err)
	}
	resContent, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resContent.CacheHit {
		t.Error("expected content change to cause a cache miss")
	}
	if conversions.Load() <= first {
		t.Fatal("content change did not invalidate cache")
	}
	if err := os.Remove(filepath.Join(outputDir, "page", "index.html")); err != nil {
		t.Fatal(err)
	}
	resMissing, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resMissing.CacheHit {
		t.Error("expected missing output file to cause a cache miss")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "page", "index.html")); err != nil {
		t.Fatalf("missing output was not rebuilt: %v", err)
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
	tmplContent := `<!DOCTYPE html><html><head><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><meta property="og:image" content="{{.Image}}">{{if .CanonicalURL}}<link rel="canonical" href="{{.CanonicalURL}}"><meta property="og:url" content="{{.CanonicalURL}}">{{end}}</head><body>{{.Content}}</body></html>`
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
		SiteURL:            "https://example.com",
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

	expectedCanonical := `<link rel="canonical" href="https://example.com/test/">`
	if !strings.Contains(outHTML, expectedCanonical) {
		t.Errorf("output HTML missing expected canonical link.\nGot: %s", outHTML)
	}

	expectedOGURL := `<meta property="og:url" content="https://example.com/test/">`
	if !strings.Contains(outHTML, expectedOGURL) {
		t.Errorf("output HTML missing expected og:url tag.\nGot: %s", outHTML)
	}
}

func TestBuildDiscoveryUsesRenderedPagesOnly(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("# Home"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "guide.md"), []byte("# Guide"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "source.md"), []byte("---\nrender: false\n---\n# Source"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = filepath.Join(templateDir, "layout.html")
	cfg.SiteURL = "https://example.com"

	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	sitemap, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(sitemap)
	for _, want := range []string{"https://example.com/", "https://example.com/guide/"} {
		if !strings.Contains(got, want) {
			t.Errorf("sitemap missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "source") {
		t.Errorf("sitemap must exclude render:false pages: %s", got)
	}
}

func TestGeneratorSEOEmptySiteURLUsesRelativeSitemap(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	output := filepath.Join(root, "public")
	if err := os.MkdirAll(content, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "index.md"), []byte("# Home"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(content, "draft.md"), []byte("---\nrender: false\n---\n# Draft"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ContentDir = content
	cfg.OutputDir = output
	cfg.Template = "../../templates/layout.html"
	cfg.SiteURL = ""
	if _, err := Build(cfg); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "rel=\"canonical\"") || strings.Contains(string(html), "property=\"og:url\"") {
		t.Fatalf("SEO URL tags should be omitted without siteurl: %s", html)
	}
	sitemap, err := os.ReadFile(filepath.Join(output, "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sitemap), "<loc>/</loc>") || strings.Contains(string(sitemap), "draft") {
		t.Fatalf("unexpected local sitemap: %s", sitemap)
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

func TestBuild_CacheHitMissStats(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "public")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(templateDir, "layout.html")
	if err := os.WriteFile(templatePath, []byte("{{.Content}}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "page1.md"), []byte("# Page 1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "page2.md"), []byte("# Page 2"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.Template = templatePath
	cfg.OutputDir = outputDir
	cfg.ProjectRoot = tempDir

	// 1. Initial build: cache miss
	res1, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build 1 failed: %v", err)
	}
	if res1.CacheHit {
		t.Errorf("Build 1 CacheHit = true, want false (cache miss)")
	}
	if res1.PageCount != 2 {
		t.Errorf("Build 1 PageCount = %d, want 2", res1.PageCount)
	}

	// 2. Repeat build with no changes: cache hit
	res2, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build 2 failed: %v", err)
	}
	if !res2.CacheHit {
		t.Errorf("Build 2 CacheHit = false, want true (cache hit)")
	}
	if res2.PageCount != 2 {
		t.Errorf("Build 2 PageCount = %d, want 2", res2.PageCount)
	}

	// 3. Modify source file: cache miss (invalidation)
	if err := os.WriteFile(filepath.Join(contentDir, "page1.md"), []byte("# Page 1 updated"), 0600); err != nil {
		t.Fatal(err)
	}
	res3, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build 3 failed: %v", err)
	}
	if res3.CacheHit {
		t.Errorf("Build 3 CacheHit = true, want false after content invalidation")
	}

	// 4. Repeat build after invalidation rebuild: cache hit
	res4, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build 4 failed: %v", err)
	}
	if !res4.CacheHit {
		t.Errorf("Build 4 CacheHit = false, want true (cache hit)")
	}
}

func TestBuild_SearchIndexHeadings(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	mdContent := `---
title: "Search Test Page"
tags: ["search", "test"]
---
# Main Header

Some introductory text.

## Feature Section

More text.
`
	_ = os.WriteFile(filepath.Join(contentDir, "test.md"), []byte(mdContent), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	searchJSONPath := filepath.Join(outputDir, "search.json")
	data, err := os.ReadFile(searchJSONPath)
	if err != nil {
		t.Fatalf("failed to read search.json: %v", err)
	}

	searchJSON := string(data)
	if !strings.Contains(searchJSON, `"t":"Search Test Page"`) {
		t.Errorf("search.json missing title: %s", searchJSON)
	}
	if !strings.Contains(searchJSON, `"h":["Main Header","Feature Section"]`) {
		t.Errorf("search.json missing expected headings: %s", searchJSON)
	}
}

func TestBuild_TaxonomyPagesIntegration(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateDir := filepath.Join(tempDir, "templates")

	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(templateDir, 0755)

	templatePath := filepath.Join(templateDir, "layout.html")
	_ = os.WriteFile(templatePath, []byte("{{.Content}}"), 0600)

	page1 := `---
title: "First Post"
tags: ["go", "web"]
category: "tech"
---
# First Post Content
`
	page2 := `---
title: "Second Post"
tags: ["go"]
categories: ["tech", "news"]
---
# Second Post Content
`
	hiddenPage := `---
title: "Hidden Draft"
tags: ["secret"]
category: "internal"
render: false
---
# Hidden
`
	_ = os.WriteFile(filepath.Join(contentDir, "p1.md"), []byte(page1), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "p2.md"), []byte(page2), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "hidden.md"), []byte(hiddenPage), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.OutputDir = outputDir
	cfg.Template = templatePath
	cfg.ProjectRoot = tempDir

	res, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// 2 rendered pages + 3 tag pages (tags/index, tags/go, tags/web) + 3 category pages (categories/index, categories/news, categories/tech) = 8 pages total
	if res.PageCount != 8 {
		t.Errorf("expected PageCount = 8, got %d", res.PageCount)
	}

	// Verify tag index page exists
	tagIndexBytes, err := os.ReadFile(filepath.Join(outputDir, "tags", "index.html"))
	if err != nil {
		t.Fatalf("tags/index.html missing: %v", err)
	}
	if !strings.Contains(string(tagIndexBytes), `href="go/"`) || !strings.Contains(string(tagIndexBytes), `href="web/"`) {
		t.Errorf("tags/index.html content unexpected: %s", string(tagIndexBytes))
	}

	// Verify secret tag (from render: false) is NOT created
	if _, err := os.Stat(filepath.Join(outputDir, "tags", "secret", "index.html")); !os.IsNotExist(err) {
		t.Errorf("tags/secret/index.html should not exist for render: false page")
	}

	// Verify sitemap.xml includes taxonomy pages
	sitemapBytes, err := os.ReadFile(filepath.Join(outputDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("sitemap.xml missing: %v", err)
	}
	sitemap := string(sitemapBytes)
	if !strings.Contains(sitemap, "/tags/") || !strings.Contains(sitemap, "/categories/") {
		t.Errorf("sitemap.xml missing taxonomy URLs: %s", sitemap)
	}
}
