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

func TestCheckCommand_AssetHealth_WarningsDoNotFailCommand(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Page
date: 2026-05-10
---
# Page
![missing](/assets/missing.png)
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "design.psd"), []byte("psd"), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir, "--asset", assetDir, "--asset-health"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected check command with asset warnings to succeed (warnings do not fail command), got error: %v (stderr: %s)", err, errBuf.String())
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "[WARN]") {
		t.Errorf("expected stdout to contain '[WARN]', got: %s", outStr)
	}
	if !strings.Contains(outStr, "unsupported or suspicious image extension \".psd\"") {
		t.Errorf("expected stdout to contain '.psd' warning, got: %s", outStr)
	}
	if !strings.Contains(outStr, "missing referenced asset \"/assets/missing.png\"") {
		t.Errorf("expected stdout to contain missing asset warning, got: %s", outStr)
	}
}
