package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGatherMetadata(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// 1. Create a markdown file with frontmatter
	mdWithFrontmatter := `---
title: "Test Title"
author: "Test Author"
---
# Content here
`
	if err := os.WriteFile(filepath.Join(tmpDir, "frontmatter.md"), []byte(mdWithFrontmatter), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 2. Create a markdown file without frontmatter
	mdWithoutFrontmatter := `# Just content`
	if err := os.WriteFile(filepath.Join(tmpDir, "no_frontmatter.md"), []byte(mdWithoutFrontmatter), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 3. Create a non-markdown file
	txtFile := `Just a text file`
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte(txtFile), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 4. Create a nested directory with a markdown file
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	nestedMd := `---
title: "Nested File"
---
# Nested content
`
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte(nestedMd), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run GatherMetadata
	fileMap, err := GatherMetadata(tmpDir)
	if err != nil {
		t.Fatalf("GatherMetadata returned error: %v", err)
	}

	// Verify results
	if len(fileMap) != 3 {
		t.Errorf("expected 3 files in map, got %d", len(fileMap))
	}

	// Check frontmatter.md
	fmFile, ok := fileMap["frontmatter.md"]
	if !ok {
		t.Errorf("frontmatter.md missing from map")
	} else {
		if fmFile.Title != "Test Title" {
			t.Errorf("expected title 'Test Title', got '%s'", fmFile.Title)
		}
		if fmFile.Author != "Test Author" {
			t.Errorf("expected author 'Test Author', got '%s'", fmFile.Author)
		}
		if string(fmFile.Rest) != "# Content here\n" {
			t.Errorf("expected rest content '# Content here\\n', got '%s'", string(fmFile.Rest))
		}
	}

	// Check no_frontmatter.md
	noFmFile, ok := fileMap["no_frontmatter.md"]
	if !ok {
		t.Errorf("no_frontmatter.md missing from map")
	} else {
		if noFmFile.Title != "" {
			t.Errorf("expected empty title, got '%s'", noFmFile.Title)
		}
		if string(noFmFile.Rest) != "# Just content" {
			t.Errorf("expected rest content '# Just content', got '%s'", string(noFmFile.Rest))
		}
	}

	// Check nested.md
	nestedFile, ok := fileMap["nested/nested.md"]
	if !ok {
		t.Errorf("nested/nested.md missing from map")
	} else {
		if nestedFile.Title != "Nested File" {
			t.Errorf("expected title 'Nested File', got '%s'", nestedFile.Title)
		}
	}

	// Check that text file was ignored
	if _, ok := fileMap["ignore.txt"]; ok {
		t.Errorf("ignore.txt should not be in map")
	}
}
