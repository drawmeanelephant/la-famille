package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/checker"
	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
)

func TestNewCommand_Defaults(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"new", "my-first-post", "--content", contentDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected new command to succeed, got error: %v (stderr: %s)", err, errBuf.String())
	}

	expectedPath := filepath.Join(contentDir, "my-first-post.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created, but it was not", expectedPath)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "Created content file:") || !strings.Contains(outStr, "Next steps:") {
		t.Errorf("expected stdout to contain path and next steps, got: %s", outStr)
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	contentStr := string(data)
	if !strings.Contains(contentStr, `title: My First Post`) {
		t.Errorf("expected frontmatter title 'My First Post', got content:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `date: "`+today+`"`) && !strings.Contains(contentStr, `date: `+today) {
		t.Errorf("expected frontmatter date %q, got content:\n%s", today, contentStr)
	}

	// Validate frontmatter compatibility with internal/content and internal/checker
	metaMap, err := content.GatherMetadata(contentDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}
	meta, ok := metaMap["my-first-post.md"]
	if !ok {
		t.Fatalf("expected my-first-post.md in GatherMetadata map")
	}
	if meta.Title != "My First Post" {
		t.Errorf("expected title 'My First Post', got %q", meta.Title)
	}

	checkRes, err := checker.Validate(cfg)
	if err != nil {
		t.Fatalf("checker.Validate failed: %v", err)
	}
	if checkRes.ErrorCount() > 0 {
		t.Errorf("expected 0 checker errors for default scaffolded file, got %d", checkRes.ErrorCount())
	}
}

func TestNewCommand_CustomFlags(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{
		"new", "custom-post.md",
		"--content", contentDir,
		"--title", "Custom Post Title",
		"--tags", "go,scaffold",
		"--layout", "custom-layout",
		"--date", "2026-05-20",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected command to succeed, got: %v (stderr: %s)", err, errBuf.String())
	}

	targetPath := filepath.Join(contentDir, "custom-post.md")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(data)
	if !strings.Contains(contentStr, "Custom Post Title") {
		t.Errorf("expected title flag in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "2026-05-20") {
		t.Errorf("expected custom date in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "custom-layout") {
		t.Errorf("expected custom layout in frontmatter, got:\n%s", contentStr)
	}

	metaMap, err := content.GatherMetadata(contentDir)
	if err != nil {
		t.Fatalf("GatherMetadata failed: %v", err)
	}
	meta, ok := metaMap["custom-post.md"]
	if !ok {
		t.Fatalf("expected custom-post.md in GatherMetadata map")
	}
	if meta.Title != "Custom Post Title" || meta.Date != "2026-05-20" || meta.Layout != "custom-layout" {
		t.Errorf("metadata mismatch: %+v", meta)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "go" || meta.Tags[1] != "scaffold" {
		t.Errorf("expected tags [go scaffold], got %v", meta.Tags)
	}

	checkRes, err := checker.Validate(cfg)
	if err != nil {
		t.Fatalf("checker.Validate failed: %v", err)
	}
	if checkRes.ErrorCount() > 0 {
		t.Errorf("expected 0 checker errors for custom scaffolded file, got %d", checkRes.ErrorCount())
	}
}

func TestNewCommand_NestedPath(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"new", "blog/tech/deep-dive", "--content", contentDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected command to succeed, got: %v (stderr: %s)", err, errBuf.String())
	}

	nestedPath := filepath.Join(contentDir, "blog", "tech", "deep-dive.md")
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Fatalf("expected nested file %s to exist", nestedPath)
	}
}

func TestNewCommand_OverwriteRefusalAndForce(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	targetPath := filepath.Join(contentDir, "existing.md")
	if err := os.WriteFile(targetPath, []byte("original content"), 0600); err != nil {
		t.Fatal(err)
	}

	// 1. Attempt overwrite without --force -> should fail
	rootCmd1 := setupRootCmd(cfg)
	var outBuf1, errBuf1 bytes.Buffer
	rootCmd1.SetOut(&outBuf1)
	rootCmd1.SetErr(&errBuf1)
	rootCmd1.SetArgs([]string{"new", "existing.md", "--content", contentDir})

	err := rootCmd1.Execute()
	if err == nil {
		t.Fatal("expected overwrite without --force to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error message to contain 'already exists', got: %v", err)
	}

	data1, _ := os.ReadFile(targetPath)
	if string(data1) != "original content" {
		t.Errorf("file content was altered without force: %s", string(data1))
	}

	// 2. Attempt overwrite with --force -> should succeed
	rootCmd2 := setupRootCmd(cfg)
	var outBuf2, errBuf2 bytes.Buffer
	rootCmd2.SetOut(&outBuf2)
	rootCmd2.SetErr(&errBuf2)
	rootCmd2.SetArgs([]string{"new", "existing.md", "--content", contentDir, "--force", "--title", "Forced Overwrite"})

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("expected force overwrite to succeed, got error: %v", err)
	}

	data2, _ := os.ReadFile(targetPath)
	if !strings.Contains(string(data2), "Forced Overwrite") {
		t.Errorf("expected file to be updated with new content, got:\n%s", string(data2))
	}
}

func TestNewCommand_UnsafePath(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"new", "../outside.md", "--content", contentDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected unsafe path to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "escapes content directory") {
		t.Errorf("expected error to contain 'escapes content directory', got: %v", err)
	}
}
