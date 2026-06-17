package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSite_MkdirAllError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}

	// Try to use a subdirectory of the read-only directory as outputDir
	outputDir := filepath.Join(readOnlyDir, "public")

	err := buildSite("content", "templates/layout.html", outputDir)
	if err == nil {
		t.Errorf("expected error when output directory cannot be created, got nil")
	}
}
