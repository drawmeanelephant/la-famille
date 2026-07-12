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
	if err := os.WriteFile(filepath.Join(tmpDir, "frontmatter.md"), []byte(mdWithFrontmatter), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 2. Create a markdown file without frontmatter
	mdWithoutFrontmatter := `# Just content`
	if err := os.WriteFile(filepath.Join(tmpDir, "no_frontmatter.md"), []byte(mdWithoutFrontmatter), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// 3. Create a non-markdown file
	txtFile := `Just a text file`
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte(txtFile), 0600); err != nil {
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
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte(nestedMd), 0600); err != nil {
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

	t.Run("Mixed case frontmatter", func(t *testing.T) {
		content := `---
Title: "Uppercase Title"
author: "lowercase author"
Render: false
---
Some body text.`
		fileName := "mixed.md"
		if err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		fileMap, err := GatherMetadata(tmpDir)
		if err != nil {
			t.Fatalf("GatherMetadata failed: %v", err)
		}

		meta, ok := fileMap["mixed.md"]
		if !ok {
			t.Fatalf("Expected 'mixed.md' in fileMap, got none")
		}

		if meta.Title != "Uppercase Title" {
			t.Errorf("Expected Title to be 'Uppercase Title', got '%s'", meta.Title)
		}
		if meta.Author != "lowercase author" {
			t.Errorf("Expected Author to be 'lowercase author', got '%s'", meta.Author)
		}
		if meta.Render == nil || *meta.Render != false {
			t.Errorf("Expected Render to be false, got %v", meta.Render)
		}
	})

	t.Run("All uppercase frontmatter", func(t *testing.T) {
		content := `---
TITLE: "All Uppercase Title"
AUTHOR: "UPPERCASE AUTHOR"
DATE: "2024-01-01"
RENDER: false
LAYOUT: "blog"
---
Uppercase body.`
		fileName := "uppercase.md"
		if err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		fileMap, err := GatherMetadata(tmpDir)
		if err != nil {
			t.Fatalf("GatherMetadata failed: %v", err)
		}

		meta, ok := fileMap["uppercase.md"]
		if !ok {
			t.Fatalf("Expected 'uppercase.md' in fileMap, got none")
		}

		if meta.Title != "All Uppercase Title" {
			t.Errorf("Expected Title to be 'All Uppercase Title', got '%s'", meta.Title)
		}
		if meta.Author != "UPPERCASE AUTHOR" {
			t.Errorf("Expected Author to be 'UPPERCASE AUTHOR', got '%s'", meta.Author)
		}
		if meta.Date != "2024-01-01" {
			t.Errorf("Expected Date to be '2024-01-01', got '%s'", meta.Date)
		}
		if meta.Render == nil || *meta.Render != false {
			t.Errorf("Expected Render to be false, got %v", meta.Render)
		}
		if meta.Layout != "blog" {
			t.Errorf("Expected Layout to be 'blog', got '%s'", meta.Layout)
		}
	})

}

func TestGatherMetadataValidation(t *testing.T) {
	tmpDir := t.TempDir()

	mdContent := `---
title: "Test Title"
tags: ["Valid-Tag", "Inv@lid_Tag"]
date: "invalid-date"
---
Content
`
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(mdContent), 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	fileMap, err := GatherMetadata(tmpDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}

	meta, ok := fileMap["test.md"]
	if !ok {
		t.Fatalf("Expected test.md in fileMap")
	}

	if meta.Date != "" {
		t.Errorf("Expected date to be cleared due to invalid format, got: %s", meta.Date)
	}

	if len(meta.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(meta.Tags))
	}
	if meta.Tags[0] != "valid-tag" {
		t.Errorf("Expected tag 0 to be 'valid-tag', got: %s", meta.Tags[0])
	}
	if meta.Tags[1] != "invlidtag" {
		t.Errorf("Expected tag 1 to be 'invlidtag', got: %s", meta.Tags[1])
	}
}

func TestGatherMetadata_SkipSymlink(t *testing.T) {
	tempDir := t.TempDir()

	targetFile := filepath.Join(tempDir, "target.md")
	_ = os.WriteFile(targetFile, []byte("# Target"), 0600)

	contentDir := filepath.Join(tempDir, "content")
	_ = os.MkdirAll(contentDir, 0755)

	symlinkPath := filepath.Join(contentDir, "symlink.md")
	err := os.Symlink(targetFile, symlinkPath)
	if err != nil {
		t.Skipf("Symlinks not supported on this platform: %v", err)
	}

	fileMap, err := GatherMetadata(contentDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}

	if _, ok := fileMap["symlink.md"]; ok {
		t.Errorf("Expected symlink to be skipped")
	}
}
