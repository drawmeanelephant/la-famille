package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestValidate_ValidContent(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := `---
title: Page One
date: 2026-05-10
tags:
  - go
  - web
---
# Page One
Link to [Page Two](page2.md).
`
	doc2 := `---
title: Page Two
date: 2026-05-11
tags:
  - go
---
# Page Two
Back to [Page One](page1.md).
`
	if err := os.WriteFile(filepath.Join(contentDir, "page1.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "page2.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if len(res.Findings) != 0 {
		t.Errorf("expected 0 findings for valid content, got %d: %v", len(res.Findings), res.Findings)
	}
	if res.ErrorCount() != 0 {
		t.Errorf("expected ErrorCount() = 0, got %d", res.ErrorCount())
	}
	if res.WarnCount() != 0 {
		t.Errorf("expected WarnCount() = 0, got %d", res.WarnCount())
	}
}

func TestValidate_InvalidFrontmatter(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Broken YAML
date: [invalid yaml sequence
---
# Content
`
	if err := os.WriteFile(filepath.Join(contentDir, "bad_yaml.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.ErrorCount() == 0 {
		t.Fatalf("expected error for invalid frontmatter, got 0 errors")
	}

	found := false
	for _, f := range res.Findings {
		if f.File == "bad_yaml.md" && f.Level == LevelError && strings.Contains(f.Message, "invalid frontmatter") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid frontmatter error finding, got: %v", res.Findings)
	}
}

func TestValidate_InvalidDateAndMalformedTag(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Test Page
date: 2026-13-45
tags:
  - Valid-Tag
  - Bad Tag!
---
# Content
`
	if err := os.WriteFile(filepath.Join(contentDir, "page.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if res.ErrorCount() == 0 {
		t.Errorf("expected date error finding, got 0 errors")
	}
	if res.WarnCount() == 0 {
		t.Errorf("expected tag warning finding, got 0 warnings")
	}

	hasDateError := false
	hasTagWarn := false
	for _, f := range res.Findings {
		if f.File == "page.md" && f.Level == LevelError && strings.Contains(f.Message, "invalid date format") {
			hasDateError = true
			if f.Line != 3 {
				t.Errorf("expected date error line to be 3, got %d", f.Line)
			}
		}
		if f.File == "page.md" && f.Level == LevelWarn && strings.Contains(f.Message, "malformed tag") {
			hasTagWarn = true
		}
	}
	if !hasDateError {
		t.Errorf("missing invalid date error in findings: %v", res.Findings)
	}
	if !hasTagWarn {
		t.Errorf("missing tag warning in findings: %v", res.Findings)
	}
}

func TestValidate_RenderFalseAndSlugCases(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. render: false with slug (invalid combination)
	doc1 := `---
title: Raw File
render: false
slug: my-slug
---
# Raw File
`
	// 2. Invalid slug format (contains slashes/dots)
	doc2 := `---
title: Bad Slug Page
slug: ../bad/slug
---
# Bad Slug
`
	// 3. Valid render: false without slug
	doc3 := `---
title: Valid Raw File
render: false
---
# Valid Raw File
`
	if err := os.WriteFile(filepath.Join(contentDir, "raw_with_slug.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "bad_slug.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "valid_raw.md"), []byte(doc3), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	hasRenderSlugErr := false
	hasBadSlugErr := false

	for _, f := range res.Findings {
		if f.File == "raw_with_slug.md" && f.Level == LevelError && strings.Contains(f.Message, "invalid render/slug combination") {
			hasRenderSlugErr = true
		}
		if f.File == "bad_slug.md" && f.Level == LevelError && strings.Contains(f.Message, "invalid slug") {
			hasBadSlugErr = true
		}
	}

	if !hasRenderSlugErr {
		t.Errorf("missing render:false + slug error in findings: %v", res.Findings)
	}
	if !hasBadSlugErr {
		t.Errorf("missing invalid slug error in findings: %v", res.Findings)
	}
}

func TestValidate_OutputPathCollision(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Two files mapping to custom-page/index.html
	doc1 := `---
title: Page One
slug: custom-page
---
# Page One
`
	doc2 := `---
title: Page Two
slug: custom-page
---
# Page Two
`
	if err := os.WriteFile(filepath.Join(contentDir, "page1.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "page2.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	hasCollision := false
	for _, f := range res.Findings {
		if f.Level == LevelError && strings.Contains(f.Message, "output path collision") {
			hasCollision = true
			break
		}
	}
	if !hasCollision {
		t.Errorf("expected output path collision error, got findings: %v", res.Findings)
	}
}

func TestValidate_BrokenInternalLinks(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	subDir := filepath.Join(contentDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := `---
title: Root Page
---
# Root Page

Line 6: Link to [existing](sub/page2.md).
Line 7: Link to [broken relative](nonexistent.md).
Line 8: Link to [broken root-relative](/sub/missing.md).
`
	doc2 := `---
title: Sub Page
---
# Sub Page
Link to [broken backlink](../missing_root.md).
`
	if err := os.WriteFile(filepath.Join(contentDir, "root.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "page2.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if res.ErrorCount() < 3 {
		t.Errorf("expected at least 3 broken link errors, got ErrorCount() = %d (%v)", res.ErrorCount(), res.Findings)
	}

	brokenLinksFound := 0
	for _, f := range res.Findings {
		if f.Level == LevelError && strings.Contains(f.Message, "broken internal link") {
			brokenLinksFound++
			if f.File == "root.md" && strings.Contains(f.Message, "nonexistent.md") {
				if f.Line != 7 {
					t.Errorf("expected broken link on line 7, got line %d", f.Line)
				}
			}
			if f.File == "root.md" && strings.Contains(f.Message, "/sub/missing.md") {
				if f.Line != 8 {
					t.Errorf("expected broken link on line 8, got line %d", f.Line)
				}
			}
		}
	}
	if brokenLinksFound != 3 {
		t.Errorf("expected 3 broken link findings, got %d", brokenLinksFound)
	}
}

func TestValidate_RenderFalseLinksValid(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc1 := `---
title: Rendered Page
---
# Rendered Page
Link to [raw file](raw.md).
`
	doc2 := `---
title: Raw File
render: false
---
# Raw File Content
Link to [rendered page](page1.md).
`
	if err := os.WriteFile(filepath.Join(contentDir, "page1.md"), []byte(doc1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "raw.md"), []byte(doc2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if len(res.Findings) != 0 {
		t.Errorf("expected 0 findings for valid render:false link scenario, got %d: %v", len(res.Findings), res.Findings)
	}
}

func TestValidate_AssetHealth_LargeRaster(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	doc := "---\ntitle: Home\n---\n# Home\n![hero](/assets/hero.png)\n"
	_ = os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600)

	// Create a large file (e.g. 200 bytes with custom low threshold)
	largeData := make([]byte, 200)
	_ = os.WriteFile(filepath.Join(assetDir, "hero.png"), largeData, 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true
	cfg.MaxAssetSizeBytes = 100 // low threshold for testing

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	foundLarge := false
	for _, f := range res.Findings {
		if f.Level == LevelWarn && strings.Contains(f.Message, "unusually large raster asset") {
			foundLarge = true
			break
		}
	}
	if !foundLarge {
		t.Errorf("expected large raster warning, got findings: %v", res.Findings)
	}
}

func TestValidate_AssetHealth_SuspiciousExtensions(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	_ = os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("---\ntitle: Home\n---\n"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "design.psd"), []byte("psd content"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	foundSuspicious := false
	for _, f := range res.Findings {
		if f.Level == LevelWarn && strings.Contains(f.Message, "unsupported or suspicious image extension \".psd\"") {
			foundSuspicious = true
			break
		}
	}
	if !foundSuspicious {
		t.Errorf("expected suspicious extension warning for .psd, got findings: %v", res.Findings)
	}
}

func TestValidate_AssetHealth_MissingReferences(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	doc := "---\ntitle: Home\n---\n# Home\n![missing](/assets/nonexistent.png)\n"
	_ = os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	foundMissing := false
	for _, f := range res.Findings {
		if f.Level == LevelWarn && strings.Contains(f.Message, "missing referenced asset \"/assets/nonexistent.png\"") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("expected missing reference warning, got findings: %v", res.Findings)
	}
}

func TestValidate_AssetHealth_CaseCollision(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	doc := "---\ntitle: Home\n---\n# Home\n![logo](/assets/logo.png)\n"
	_ = os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "Logo.png"), []byte("data"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	foundCollision := false
	for _, f := range res.Findings {
		if f.Level == LevelWarn && (strings.Contains(f.Message, "case mismatch") || strings.Contains(f.Message, "asset case-collision")) {
			foundCollision = true
			break
		}
	}
	if !foundCollision {
		t.Errorf("expected case collision warning, got findings: %v", res.Findings)
	}
}

func TestValidate_AssetHealth_IgnoredFiles(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	_ = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte("assets/ignored.psd\n"), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("---\ntitle: Home\n---\n"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "ignored.psd"), []byte("ignored psd"), 0600)

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = tempDir
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	for _, f := range res.Findings {
		if strings.Contains(f.File, "ignored.psd") {
			t.Errorf("ignored file should not produce findings, got: %v", f)
		}
	}
}

func TestValidate_AssetHealth_DeterministicFindingOrder(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	assetDir := filepath.Join(tempDir, "assets")
	_ = os.MkdirAll(contentDir, 0755)
	_ = os.MkdirAll(assetDir, 0755)

	_ = os.WriteFile(filepath.Join(contentDir, "a.md"), []byte("---\ntitle: A\n---\n![m](/assets/missing_z.png)\n"), 0600)
	_ = os.WriteFile(filepath.Join(contentDir, "b.md"), []byte("---\ntitle: B\n---\n![m](/assets/missing_a.png)\n"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "b_file.psd"), []byte("psd"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "a_file.psd"), []byte("psd"), 0600)

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.CheckAssetHealth = true

	res1, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	res2, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if len(res1.Findings) != len(res2.Findings) {
		t.Fatalf("finding count mismatch: %d vs %d", len(res1.Findings), len(res2.Findings))
	}

	for i := range res1.Findings {
		if res1.Findings[i].String() != res2.Findings[i].String() {
			t.Errorf("finding %d mismatch:\n  res1: %s\n  res2: %s", i, res1.Findings[i].String(), res2.Findings[i].String())
		}
	}
}
