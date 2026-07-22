package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestCheckCommand_ValidContent(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := `---
title: Home Page
date: 2026-05-10
---
# Welcome
See [About](about.md).
`
	doc2 := `---
title: About Page
date: 2026-05-11
---
# About
Back to [Home](index.md).
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "about.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected check command to succeed, got error: %v (stderr: %s)", err, errBuf.String())
	}

	if !strings.Contains(outBuf.String(), "All content validation checks passed.") {
		t.Errorf("expected success message in stdout, got: %s", outBuf.String())
	}

	// Verify no output directory or artifacts were created
	publicDir := filepath.Join(tempDir, "public")
	if _, err := os.Stat(publicDir); !os.IsNotExist(err) {
		t.Errorf("check command created public directory: %s", publicDir)
	}
}

func TestCheckCommand_InvalidContent(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Broken Page
date: 2026-99-99
---
# Broken Page
Link to [missing](missing.md).
`
	if err := os.WriteFile(filepath.Join(contentDir, "broken.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected check command to fail on invalid content, but it succeeded")
	}

	if !strings.Contains(errBuf.String(), "invalid date format") {
		t.Errorf("expected stderr to contain 'invalid date format', got: %s", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "broken internal link") {
		t.Errorf("expected stderr to contain 'broken internal link', got: %s", errBuf.String())
	}
}
