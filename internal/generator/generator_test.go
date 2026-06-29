package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func TestGeneratorSEO(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "generator_seo_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	contentDir := filepath.Join(tempDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}

	outDir := filepath.Join(tempDir, "public")
	if err := os.Mkdir(outDir, 0755); err != nil {
		t.Fatalf("failed to create public dir: %v", err)
	}

	assetDir := filepath.Join(tempDir, "assets")
	if err := os.Mkdir(assetDir, 0755); err != nil {
		t.Fatalf("failed to create asset dir: %v", err)
	}

	ragDir := filepath.Join(tempDir, "rag-archive")
	if err := os.Mkdir(ragDir, 0755); err != nil {
		t.Fatalf("failed to create rag dir: %v", err)
	}

	tmplDir := filepath.Join(tempDir, "templates")
	if err := os.Mkdir(tmplDir, 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	tmplPath := filepath.Join(tmplDir, "layout.html")
	tmplContent := `<!DOCTYPE html><html><head><title>{{.Title}}</title><meta name="description" content="{{.Description}}"><meta property="og:image" content="{{.Image}}"></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	mdContent := `---
title: Test SEO
description: "Test SEO Description"
image: "/images/test-seo.png"
---
# Hello SEO`
	mdPath := filepath.Join(contentDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
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

	_, err = generator.Build(cfg)
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
