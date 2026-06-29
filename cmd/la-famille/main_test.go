package main

import (
	"bytes"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/stub"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIOverrides(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Write a config.yaml to the temp dir
	configFile := filepath.Join(tmpDir, "config.yaml")
	yamlContent := []byte(`
site_name: "Test Site From Config"
output_dir: "default_output_from_config"
content_dir: "default_content_from_config"
theme: "dark"
`)
	if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Create content dir
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	// Write a test markdown file
	mdContent := []byte(`---
title: Test Page
---
# Hello World
<script>alert('xss')</script>
`)
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), mdContent, 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Create templates dir and layout
	templateDir := filepath.Join(tmpDir, "templates")
	if err := os.Mkdir(templateDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}
	htmlContent := []byte(`<!DOCTYPE html>
<html lang="en" data-theme="{{.Site.Theme}}">
<body>
<h1>{{.Title}} - {{.Site.SiteName}}</h1>
{{.Content}}
</body>
</html>`)
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), htmlContent, 0644); err != nil {
		t.Fatalf("Failed to write layout.html: %v", err)
	}

	// Build la-famille executable first
	exePath := filepath.Join(tmpDir, "la-famille.bin")
	cmdBuild := exec.Command("go", "build", "-o", exePath, "../../cmd/la-famille")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to build la-famille: %v", err)
	}

	cmdRun := exec.Command(exePath, "build",
		"--content", contentDir,
		"--output", filepath.Join(tmpDir, "cli_output"),
		"--template", filepath.Join(templateDir, "layout.html"))

	// Run from tmpDir so it picks up config.yaml
	cmdRun.Dir = tmpDir

	var stderr bytes.Buffer
	cmdRun.Stderr = &stderr
	if err := cmdRun.Run(); err != nil {
		t.Fatalf("la-famille run failed: %v, stderr: %s", err, stderr.String())
	}

	// Check if output went to `cli_output` instead of `default_output_from_config`
	outputFile := filepath.Join(tmpDir, "cli_output", "index.html")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file was not created in the CLI-specified directory. Did CLI flag override fail?")
	}

	// Read output to ensure config vars (like Theme and SiteName) were still loaded
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated html: %v", err)
	}
	htmlStr := string(content)

	if !strings.Contains(htmlStr, `data-theme="dark"`) {
		t.Errorf("Expected config data-theme='dark' to be present, but it wasn't")
	}
	if !strings.Contains(htmlStr, `Test Page - Test Site From Config`) {
		t.Errorf("Expected SiteName from config to be present, but it wasn't")
	}
	if strings.Contains(htmlStr, "<script>") {
		t.Errorf("XSS payload was not sanitized: %s", htmlStr)
	}
}

func TestInitCommand(t *testing.T) {
	tmpDir := t.TempDir()

	exePath := filepath.Join(tmpDir, "la-famille.bin")
	cmdBuild := exec.Command("go", "build", "-o", exePath, "../../cmd/la-famille")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to build la-famille: %v", err)
	}

	cmdRun := exec.Command(exePath, "init")
	cmdRun.Dir = tmpDir

	if err := cmdRun.Run(); err != nil {
		t.Fatalf("la-famille init failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("la-famille init did not create config.yaml")
	}
}

func TestStubRelPathFromToFallback(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		target   string
		expected string
		wantErr  bool
	}{
		{
			name:     "same directory",
			base:     "a.md",
			wantErr:  false,
			target:   "b.html",
			expected: "b.html",
		},
		{
			name:     "target in subdirectory",
			base:     "a.md",
			wantErr:  false,
			target:   "dir/b.html",
			expected: "dir/b.html",
		},
		{
			name:     "base in subdirectory",
			base:     "dir/a.md",
			wantErr:  false,
			target:   "b.html",
			expected: "../b.html",
		},
		{
			name:     "absolute and relative paths (error)",
			base:     "/absolute/path/base.md",
			target:   "relative/target.html",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stub.RelPathFromTo(tt.base, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("stub.RelPathFromTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("stub.RelPathFromTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessFile_PathTraversalPrevented(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock config
	cfg := config.Config{
		ContentDir: filepath.Join(tempDir, "content"),
		OutputDir:  filepath.Join(tempDir, "public"),
		Template:   filepath.Join(tempDir, "layout.html"),
	}

	os.MkdirAll(cfg.ContentDir, 0755)
	os.MkdirAll(cfg.OutputDir, 0755)
	os.WriteFile(cfg.Template, []byte("<html><body>{{.Content}}</body></html>"), 0644)

	fileName := "index.md"
	// Path traverses out of the content directory to a theoretical /tmp directory
	content := []byte("# Home\n[Malicious](../../../../../tmp/hack.md)")
	os.WriteFile(filepath.Join(cfg.ContentDir, fileName), content, 0644)

	_, err := generator.Build(cfg)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	// Make sure the index file is generated but doesn't rewrite to .html (stays as original destination because traversal was blocked)
	indexFile := filepath.Join(cfg.OutputDir, "index.html")
	indexContent, _ := os.ReadFile(indexFile)
	if strings.Contains(string(indexContent), `href="../../../../../tmp/hack.html"`) {
		t.Errorf("Malicious link was incorrectly rewritten to .html: %s", string(indexContent))
	}

	// Verify that the malicious file stub is not created anywhere
	maliciousFile := filepath.Join(tempDir, "tmp", "hack.html")
	if _, err := os.Stat(maliciousFile); !os.IsNotExist(err) {
		t.Errorf("Malicious stub was incorrectly generated outside the output directory at: %s", maliciousFile)
	}
}

func TestRun_WalkError(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock config
	cfg := config.Config{
		ContentDir: filepath.Join(tempDir, "does-not-exist"),
		OutputDir:  filepath.Join(tempDir, "public"),
		Template:   filepath.Join(tempDir, "layout.html"),
	}

	// Create valid output dir and template file so it only fails on content dir
	os.MkdirAll(cfg.OutputDir, 0755)
	os.WriteFile(cfg.Template, []byte("<html><body>{{.Content}}</body></html>"), 0644)

	_, err := generator.Build(cfg)
	if err == nil {
		t.Fatalf("expected an error when walking a non-existent directory, but got nil")
	}

	if !strings.Contains(err.Error(), "failed to walk content directory") {
		t.Errorf("expected error message to contain 'failed to walk content directory', got: %v", err)
	}
}
