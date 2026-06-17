package main

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainXSS(t *testing.T) {
	// Create temporary directories
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

	// Create test template
	templateFile := filepath.Join(templateDir, "layout.html")
	err = os.WriteFile(templateFile, []byte("<html><body>{{.Content}}</body></html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create test markdown with XSS payload
	testFile := filepath.Join(contentDir, "test.md")
	err = os.WriteFile(testFile, []byte("## Title\n\n<script>alert('xss')</script>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Change working directory to temp dir
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Run main
	main()

	// Check output
	outputFile := filepath.Join(outputDir, "test.md.html")
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

func TestProcessFile(t *testing.T) {
	// Setup mock input and output directories
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	outputDir := filepath.Join(tempDir, "public")
	err := os.MkdirAll(contentDir, 0755)
	if err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create a dummy template
	tmpl, err := template.New("layout").Parse("<html><body>{{.Content}}</body></html>")
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	// Create a dummy markdown file
	fileName := "test.md"
	content := []byte("# Hello World\nThis is a test.")
	err = os.WriteFile(filepath.Join(contentDir, fileName), content, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run processFile
	err = processFile(fileName, contentDir, outputDir, tmpl)
	if err != nil {
		t.Errorf("processFile failed: %v", err)
	}

	// Assert the output file exists
	outFileName := fileName + ".html"
	_, err = os.Stat(filepath.Join(outputDir, outFileName))
	if os.IsNotExist(err) {
		t.Errorf("expected output file %s does not exist", outFileName)
	} else if err != nil {
		t.Errorf("failed to check output file: %v", err)
	}
}
