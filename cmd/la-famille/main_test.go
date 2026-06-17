package main

import (
	"os"
	"strings"
	"testing"
)

func TestRun_TemplateParsingError(t *testing.T) {
	// Create a temporary directory for content to avoid os.ReadDir error
	contentDir, err := os.MkdirTemp("", "content")
	if err != nil {
		t.Fatalf("Failed to create temp content dir: %v", err)
	}
	defer os.RemoveAll(contentDir)

	outputDir, err := os.MkdirTemp("", "public")
	if err != nil {
		t.Fatalf("Failed to create temp output dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Provide a non-existent template file
	templateFile := "non_existent_template.html"

	err = run(contentDir, templateFile, outputDir)
	if err == nil {
		t.Fatalf("Expected error for non-existent template file, got nil")
	}

	// Verify the error mentions the template file
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("Expected error to mention 'no such file or directory', got: %v", err)
	}
}
