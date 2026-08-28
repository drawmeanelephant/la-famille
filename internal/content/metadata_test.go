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
category: "Engineering"
categories: ["Tech", "Engineering"]
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

	if len(meta.Categories) != 2 {
		t.Fatalf("Expected 2 normalized unique categories, got %d: %v", len(meta.Categories), meta.Categories)
	}
	if meta.Categories[0] != "tech" || meta.Categories[1] != "engineering" {
		t.Errorf("Expected categories ['tech', 'engineering'], got %v", meta.Categories)
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

func TestGatherMetadata_FrontmatterParseWarning(t *testing.T) {
	tmpDir := t.TempDir()
	// Malformed frontmatter: unclosed sequence
	bad := "---\ntitle: Bad\ntags: [unclosed\n---\n# Body\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "bad.md"), []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	fileMap, err := GatherMetadata(tmpDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}
	meta, ok := fileMap["bad.md"]
	if !ok {
		t.Fatal("bad.md not in fileMap")
	}
	if len(meta.Warnings) == 0 {
		t.Fatalf("expected frontmatter warning, got none: %#v", meta.Warnings)
	}
	foundFallback := false
	for _, w := range meta.Warnings {
		if contains(w, "frontmatter parse warning in bad.md") && contains(w, "falling back to raw markdown") {
			foundFallback = true
			break
		}
	}
	if !foundFallback {
		t.Errorf("warnings = %v, want frontmatter parse fallback warning", meta.Warnings)
	}
	// Rest should be raw markdown fallback
	if string(meta.Rest) != bad {
		t.Errorf("Rest = %q, want raw markdown fallback", string(meta.Rest))
	}
}

// Issue #530: a  `---` opener with no matching closer used to be treated as
// "no frontmatter", so the whole document rendered as body text and the build
// summary stayed at warnings=0.
func TestGatherMetadataUnterminatedFrontmatterWarns(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{{
		name: "unterminated YAML opener",
		body: "---\ntitle: Broken\ncontent: hello\n",
	}, {
		name: "opener followed by blank then text",
		body: "---\n\nsome body\n",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			meta := writeContentFile(t, tc.body)
			if len(meta.Warnings) == 0 {
				t.Fatalf("expected an unterminated-frontmatter warning, got none: %#v", meta.Warnings)
			}
			found := false
			for _, w := range meta.Warnings {
				if contains(w, "unterminated frontmatter") && contains(w, "page.md") {
					found = true
				}
			}
			if !found {
				t.Errorf("warnings = %v, want an unterminated-frontmatter warning naming the file", meta.Warnings)
			}
		})
	}
}

// A properly closed frontmatter block must not trip the unterminated check,
// even when it is empty or has only comments.
func TestGatherMetadataClosedFrontmatterDoesNotWarnUnterminated(t *testing.T) {
	meta := writeContentFile(t, "---\n# only a comment\n---\nBody.\n")
	for _, w := range meta.Warnings {
		if contains(w, "unterminated frontmatter") {
			t.Errorf("closed frontmatter must not warn as unterminated, got %v", meta.Warnings)
		}
	}
}

// Issue #532: normalizing a value is lossy ("café ☕" → "caf") and dropping one
// that normalizes to empty ("☕") used to leave the build summary at warnings=0.
// Both must land in FileMeta.Warnings so the summary counts them.
func TestGatherMetadataCountsLossyTaxonomyWarnings(t *testing.T) {
	meta := writeContentFile(t, "---\nlayout: T\ntags: [café ☕, ☕, travel log, plain]\n---\nbody\n")

	// "plain" is already slug-safe and passes through; only the lossy pair is
	// mangled and the empty-normalized one dropped.
	if len(meta.Tags) != 3 || meta.Tags[0] != "caf" || meta.Tags[1] != "travellog" || meta.Tags[2] != "plain" {
		t.Errorf("tags = %v, want [caf travellog plain]", meta.Tags)
	}

	var sawMangled, sawEmpty bool
	for _, w := range meta.Warnings {
		switch {
		case contains(w, "normalized") && contains(w, "café ☕") && contains(w, "page.md"):
			sawMangled = true
		case contains(w, "empty value") && contains(w, "☕") && contains(w, "page.md"):
			sawEmpty = true
		}
	}
	if !sawMangled {
		t.Errorf("warnings = %v, want a counted warning for the lossy tag normalization", meta.Warnings)
	}
	if !sawEmpty {
		t.Errorf("warnings = %v, want a counted warning for the empty-normalized tag", meta.Warnings)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
