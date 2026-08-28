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
description: Home description
---
# Welcome
See [About](about.md).
`
	doc2 := `---
title: About Page
date: 2026-05-11
description: About description
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
	// A clean pass needs a siteurl: without it, check (as the discoverability
	// guard for the sitemap) reports a site-wide warning (#535).
	cfg.SiteURL = "https://example.com"

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected check command to succeed, got error: %v (stderr: %s)", err, errBuf.String())
	}

	if !strings.Contains(outBuf.String(), "La Famille Diagnostics [") {
		t.Errorf("expected build version header in stdout, got: %s", outBuf.String())
	}

	if !strings.Contains(outBuf.String(), "✓ 0 errors, 0 warnings | 0 orphaned pages, 0 missing descriptions, 0 missing dates") {
		t.Errorf("expected summary footer in stdout for clean site, got: %s", outBuf.String())
	}

	if strings.Contains(outBuf.String(), "All content validation checks passed.") {
		t.Errorf("clean run with summary should not print 'All content validation checks passed.', got: %s", outBuf.String())
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

func TestCheckCommand_Summary_WarningsOnly(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// index provides inbound to about, but orphan has no inbound; also missing description cases
	indexDoc := `---
title: Home
date: 2026-05-10
description: home desc
---
# Home
Link to [About](about.md).
`
	aboutDoc := `---
title: About
date: 2026-05-11
description: about desc
---
# About
No links.
`
	orphanDoc := `---
title: Orphan
date: 2026-05-12
---
# Orphan
Missing description should warn.
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(indexDoc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "about.md"), []byte(aboutDoc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "orphan.md"), []byte(orphanDoc), 0600); err != nil {
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
		t.Fatalf("warnings-only should succeed (exit 0), got error: %v (stderr: %s, stdout: %s)", err, errBuf.String(), outBuf.String())
	}

	outStr := outBuf.String()
	// Warnings with no errors => ⚠ symbol (#512); ✓ is reserved for a clean run.
	if !strings.Contains(outStr, "⚠ 0 errors") {
		t.Errorf("expected warnings-only footer to start with '⚠ 0 errors', got: %s", outStr)
	}
	if strings.Contains(outStr, "✓ 0 errors") {
		t.Errorf("warnings-only footer must not use the clean-run ✓ symbol, got: %s", outStr)
	}
	if !strings.Contains(outStr, "1 orphaned page, 1 missing description, 0 missing dates") {
		t.Errorf("expected footer with '1 orphaned page, 1 missing description, 0 missing dates', got: %s", outStr)
	}
	if !strings.Contains(outStr, "[WARN]") {
		t.Errorf("expected warnings in stdout, got: %s", outStr)
	}
	// warnings should be on stdout, not stderr
	if strings.Contains(errBuf.String(), "orphaned page") {
		t.Errorf("orphan warnings should be on stdout, not stderr: %s", errBuf.String())
	}
}

func TestCheckCommand_Summary_Errors(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexDoc := `---
title: Home
date: 2026-05-10
description: home desc
---
# Home
Link to [Broken](page.md).
`
	doc := `---
title: Broken
date: 2026-05-10
description: desc
---
# Broken
Link to [missing](missing.md).
Link to [also-missing](also_missing.md).
`
	orphanDoc := `---
title: Orphan
description: orphan desc
date: 2026-05-12
---
# Orphan
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(indexDoc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "page.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "orphan.md"), []byte(orphanDoc), 0600); err != nil {
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
		t.Fatalf("expected error exit for broken links, got success (stdout: %s, stderr: %s)", outBuf.String(), errBuf.String())
	}

	outStr := outBuf.String()
	errStr := errBuf.String()

	// Footer should be on stdout even when errors exist, with ✗ symbol
	if !strings.Contains(outStr, "✗") {
		t.Errorf("expected error footer with '✗' in stdout, got: %s", outStr)
	}
	if !strings.Contains(outStr, "2 errors") {
		t.Errorf("expected footer with '2 errors' in stdout, got: %s", outStr)
	}
	if !strings.Contains(outStr, "1 orphaned page") {
		t.Errorf("expected footer orphan count in stdout, got: %s", outStr)
	}
	// Broken links are errors on stderr
	if !strings.Contains(errStr, "broken internal link") {
		t.Errorf("expected broken link errors on stderr, got: %s", errStr)
	}
	// Footer must not be on stderr
	if strings.Contains(errStr, "orphaned page") && strings.Contains(errStr, "errors,") {
		t.Errorf("footer should be on stdout, not stderr: %s", errStr)
	}
}

func TestCheckCommand_Summary_Suppressed(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := `---
title: Home Page
date: 2026-05-10
description: Home description
---
# Welcome
See [About](about.md).
`
	doc2 := `---
title: About Page
date: 2026-05-11
description: About description
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
	// A "checks passed" run needs siteurl so check is genuinely clean (#535).
	cfg.SiteURL = "https://example.com"

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir, "--summary=false"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected check with --summary=false to succeed, got error: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, "All content validation checks passed.") {
		t.Errorf("expected suppressed summary to print 'All content validation checks passed.', got: %s", outStr)
	}
	if strings.Contains(outStr, "✓") || strings.Contains(outStr, "✗") {
		t.Errorf("suppressed summary should not contain footer symbols, got: %s", outStr)
	}
	if strings.Contains(outStr, "orphaned page") {
		t.Errorf("suppressed summary should not contain orphan count, got: %s", outStr)
	}
}

func TestCheckCommand_Summary_Suppressed_WithWarnings(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Orphan
description: desc
---
# Orphan
`
	if err := os.WriteFile(filepath.Join(contentDir, "orphan.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	rootCmd := setupRootCmd(cfg)
	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"check", "--content", contentDir, "--summary=false"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("warnings-only with suppressed summary should succeed, got error: %v", err)
	}

	outStr := outBuf.String()
	if strings.Contains(outStr, "✓") || strings.Contains(outStr, "✗") {
		t.Errorf("suppressed summary should not contain footer, got: %s", outStr)
	}
	if !strings.Contains(outStr, "[WARN]") {
		t.Errorf("warnings should still appear without footer, got: %s", outStr)
	}
}
