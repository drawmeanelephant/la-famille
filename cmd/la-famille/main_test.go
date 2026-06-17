package main

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

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
