package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "la-famille-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up content directory
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	// Create a mock markdown file
	mockMD := filepath.Join(contentDir, "test.md")
	if err := os.WriteFile(mockMD, []byte("# Hello Test\nThis is a test."), 0644); err != nil {
		t.Fatalf("Failed to write mock markdown file: %v", err)
	}

	// Set up templates directory
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a mock template file
	mockTemplate := filepath.Join(templatesDir, "layout.html")
	templateContent := `<html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(mockTemplate, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write mock template file: %v", err)
	}

	// Output directory
	outputDir := filepath.Join(tmpDir, "public")

	// Execute run function
	if err := run(contentDir, mockTemplate, outputDir); err != nil {
		t.Fatalf("run() returned an error: %v", err)
	}

	// Verify output
	outputFile := filepath.Join(outputDir, "test.md.html")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Expected output file %s was not created", outputFile)
	}

	// Verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(content)
	if !strings.Contains(outputStr, "<title>test.md</title>") {
		t.Errorf("Output HTML does not contain expected title, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "<h1>Hello Test</h1>") {
		t.Errorf("Output HTML does not contain expected heading, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "<p>This is a test.</p>") {
		t.Errorf("Output HTML does not contain expected paragraph, got: %s", outputStr)
	}
}
