package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainXSS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "la-famille-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "public")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(templateDir, 0755)
	os.MkdirAll(outputDir, 0755)

	templateFile := filepath.Join(templateDir, "layout.html")
	err = os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	testFile := filepath.Join(contentDir, "test.md")
	err = os.WriteFile(testFile, []byte("## Title\n\n<script>alert('xss')</script>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	main()

	outputFile := filepath.Join(outputDir, "test.html")
	outputContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(outputContent)
	if strings.Contains(outputStr, "<script>") {
		t.Errorf("XSS payload was not sanitized: %s", outputStr)
	}

	if !strings.Contains(outputStr, "<h2>Title</h2>") {
		t.Errorf("Markdown was not rendered correctly: %s", outputStr)
	}
}

func TestRun_TemplateParsingError(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	outputDir := filepath.Join(tempDir, "public")
	templateFile := "non_existent_template.html"

	err := run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Fatalf("Expected error for non-existent template file, got nil")
	}
}

func TestRun_MkdirAllError(t *testing.T) {
	tempDir := t.TempDir()

	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}

	outputDir := filepath.Join(readOnlyDir, "public")

	// Create mock content and template so the failure happens at output creation
	contentDir := filepath.Join(tempDir, "content")
	os.MkdirAll(contentDir, 0755)
	os.WriteFile(filepath.Join(contentDir, "a.md"), []byte("a"), 0644)
	templateFile := filepath.Join(tempDir, "layout.html")
	os.WriteFile(templateFile, []byte("<html></html>"), 0644)

	err := run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Errorf("expected error when output directory cannot be created, got nil")
	}
}

func TestRun_ProcessFileErrorLog(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0555); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	if err := os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	fileName := "test.md"
	if err := os.WriteFile(filepath.Join(contentDir, fileName), []byte("# Hello"), 0644); err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	err := run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Fatalf("Expected run to return error when process file fails to write output, got %v", err)
	}
}

func TestMain_ErrorPath(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		tempDir := t.TempDir()
		os.Chdir(tempDir)
		os.Mkdir("content", 0755)
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ErrorPath")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestProcessFile_WithFrontmatter(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(outputDir, 0755)
	os.WriteFile(templateFile, []byte("<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1><h2>{{.Author}}</h2><h3>{{.Date}}</h3>{{.Content}}</body></html>"), 0644)

	fileName := "test_fm.md"
	content := []byte("---\ntitle: \"The Great Fart of 1922\"\nauthor: \"Don Corleone\"\ndate: \"2026-06-17\"\n---\n# Hello World\nThis is a test.")
	os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)

	err := run(contentDir, templateFile, outputDir)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	outFile := filepath.Join(outputDir, "test_fm.html")
	outContent, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(outContent)
	if !strings.Contains(outputStr, "<title>The Great Fart of 1922</title>") {
		t.Errorf("Output HTML does not contain expected title, got: %s", outputStr)
	}
}

func TestProcessFile_RenderFalse(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(outputDir, 0755)
	os.WriteFile(templateFile, []byte("<html></html>"), 0644)

	fileName := "raw.md"
	content := []byte("---\nrender: false\n---\n# Raw Markdown\n[Link](other.md)")
	os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)

	err := run(contentDir, templateFile, outputDir)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	outFile := filepath.Join(outputDir, fileName) // Should be .md
	_, err = os.Stat(outFile)
	if os.IsNotExist(err) {
		t.Errorf("Expected raw output file %s does not exist", fileName)
	}

	htmlFile := filepath.Join(outputDir, "raw.html")
	if _, err := os.Stat(htmlFile); !os.IsNotExist(err) {
		t.Errorf("Expected html file %s to not exist", "raw.html")
	}
}

func TestProcessFile_MissingFileStub(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(outputDir, 0755)
	os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644)

	fileName := "index.md"
	content := []byte("# Home\n[Missing](missing.md)")
	os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)

	err := run(contentDir, templateFile, outputDir)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	// Index should be generated and have link to missing.html
	indexFile := filepath.Join(outputDir, "index.html")
	indexContent, _ := os.ReadFile(indexFile)
	if !strings.Contains(string(indexContent), `href="missing.html"`) {
		t.Errorf("Index should link to missing.html, got %s", string(indexContent))
	}

	// Missing file stub should be generated
	missingFile := filepath.Join(outputDir, "missing.html")
	missingContent, err := os.ReadFile(missingFile)
	if err != nil {
		t.Fatalf("Expected stub missing.html does not exist")
	}
	if !strings.Contains(string(missingContent), "This page doesn't exist yet") {
		t.Errorf("Missing stub doesn't contain expected text, got %s", string(missingContent))
	}
	if !strings.Contains(string(missingContent), `href="index.html"`) {
		t.Errorf("Missing stub doesn't contain link back to parent, got %s", string(missingContent))
	}
}

func TestProcessFile_PathTraversalPrevented(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	templateFile := filepath.Join(tempDir, "layout.html")

	os.MkdirAll(contentDir, 0755)
	os.MkdirAll(outputDir, 0755)
	os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644)

	fileName := "index.md"
	// Path traverses out of the content directory to a theoretical /tmp directory
	content := []byte("# Home\n[Malicious](../../../../../tmp/hack.md)")
	os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)

	err := run(contentDir, templateFile, outputDir)
	if err != nil {
		t.Errorf("run failed: %v", err)
	}

	// Make sure the index file is generated but doesn't rewrite to .html (stays as original destination because traversal was blocked)
	indexFile := filepath.Join(outputDir, "index.html")
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

func TestRelPathFromTo(t *testing.T) {
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
			target:   "b.html",
			expected: "b.html",
		},
		{
			name:     "target in subdirectory",
			base:     "a.md",
			target:   "dir/b.html",
			expected: "dir/b.html",
		},
		{
			name:     "base in subdirectory",
			base:     "dir/a.md",
			target:   "b.html",
			expected: "../b.html",
		},
		{
			name:     "deep nested",
			base:     "dir1/dir2/a.md",
			target:   "dir3/b.html",
			expected: "../../dir3/b.html",
		},
		{
			name:     "root files",
			base:     "index.md",
			target:   "index.html",
			expected: "index.html",
		},
		{
			name:     "same nested directory",
			base:     "dir/a.md",
			target:   "dir/b.html",
			expected: "b.html",
		},
		{
			name:     "target is base directory",
			base:     "dir/a.md",
			target:   "dir",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relPathFromTo(tt.base, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("relPathFromTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("relPathFromTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}
