package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// Issue #506: `check` advertises internal-link validation but silently ignored
// links that do not end in .md, even though a build ships them verbatim.
func TestValidateFlagsBrokenExtensionlessLink(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "blog"), 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Home
description: home
date: 2026-08-25
---
[Missing page](/does-not-exist)
[Missing nested](/missing-page/subpath)
[Home ok](/)
[Self ok](./)
`
	blogPost := `---
title: Blog post
description: post
date: 2026-08-25
tags:
  - welcome
---
[Root relative missing](/nope)
[Relative sibling](../also-missing)
[Taxonomy ok](/tags/welcome/)
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "blog", "post.md"), []byte(blogPost), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	var broken []string
	for _, f := range res.Findings {
		if f.Category == CategoryBrokenLink && strings.Contains(f.Message, "broken internal link") {
			broken = append(broken, f.Message)
		}
	}

	wantTargets := []string{"/does-not-exist", "/missing-page/subpath", "/nope", "../also-missing"}
	if len(broken) != len(wantTargets) {
		t.Fatalf("expected %d broken-link findings, got %d: %v", len(wantTargets), len(broken), broken)
	}
	for _, want := range wantTargets {
		found := false
		for _, msg := range broken {
			if strings.Contains(msg, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected broken link %q in findings, got: %v", want, broken)
		}
	}
	if containsMessage(broken, `"/"`) || containsMessage(broken, "tags/welcome") || containsMessage(broken, `"./"`) {
		t.Errorf("safe links flagged as broken: %v", broken)
	}
}

func containsMessage(messages []string, substr string) bool {
	for _, m := range messages {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

// Issue #506: .html links resolve against expected output paths, including the
// foo.html -> foo/index.html fallback the publisher already honours.
func TestValidateAcceptsHtmlAliasesForRenderedPages(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	indexDoc := `---
title: Home
description: home
date: 2026-08-25
---
[About via html](/about-page.html)
[About via dir](/about-page/)
`
	aboutDoc := `---
title: About
description: about
slug: about-page
---
[Back](/index.html)
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(indexDoc), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "about.md"), []byte(aboutDoc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	for _, f := range res.Findings {
		if f.Category != CategoryBrokenLink {
			continue
		}
		if strings.Contains(f.Message, `"/about-page.html"`) || strings.Contains(f.Message, `"/about-page/"`) {
			t.Errorf("slug output dir must resolve via both the .html alias and dir form, flagged anyway: %s", f.Message)
		}
		if strings.Contains(f.Message, `"-> "index.html"`) || strings.Contains(f.Message, `" -> "/index.html"`) {
			t.Errorf("/index.html resolves to the homepage and must not be flagged: %s", f.Message)
		}
	}
}

// Issue #509: filenames with spaces/uppercase become URL directories verbatim;
// check must warn with a rename suggestion.
func TestValidateWarnsOnUnsafeSlugFilename(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	docs := map[string]string{
		"My Post With Spaces.md": "---\ntitle: Space Test\ndescription: d\ndate: 2026-08-25\n---\nbody\n",
		"clean-name.md":          "---\ntitle: Clean\ndescription: d\ndate: 2026-08-25\n---\nbody\n",
	}
	for name, doc := range docs {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(doc), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	foundSuggestion := false
	for _, f := range res.Findings {
		if f.File == "clean-name.md" && strings.Contains(f.Message, "unsafe for URLs") {
			t.Errorf("clean filename must not be flagged: %s", f.Message)
		}
		if f.File == "My Post With Spaces.md" && strings.Contains(f.Message, "unsafe for URLs") {
			foundSuggestion = true
			if !strings.Contains(f.Message, "my-post-with-spaces.md") {
				t.Errorf("warning should suggest the normalized name my-post-with-spaces.md, got: %s", f.Message)
			}
		}
	}
	if !foundSuggestion {
		t.Errorf("expected a URL-safety warning for unsafe filename, got none of: %v", res.Findings)
	}
}

// Issue #515: --asset-health must scan installed templates for local /assets/
// references; a layout pointing at an image no build deploys has to surface
// here instead of only failing publish-check after the artifact exists.
func TestAssetHealthScansTemplates(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	templateDir := filepath.Join(tempDir, "templates")
	assetDir := filepath.Join(tempDir, "assets")
	for _, dir := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	doc := "---\ntitle: Home\ndescription: d\ndate: 2026-08-25\n---\nbody\n"
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	tmpl := "<html><body><img src=\"/assets/img/jules-logo.png\"><img src=\"/assets/img/totally-absent.png\"></body></html>"
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte(tmpl), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.AssetDir = assetDir
	cfg.Template = filepath.Join(templateDir, "layout.html")

	resOff, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	for _, f := range resOff.Findings {
		if f.Category == CategoryAssetHealth {
			t.Errorf("template scan must stay off without --asset-health, got: %s", f.Message)
		}
	}

	cfg.CheckAssetHealth = true
	resOn, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	flaggedAbsent, flaggedBundled := false, false
	for _, f := range resOn.Findings {
		if f.Category != CategoryAssetHealth {
			continue
		}
		if strings.Contains(f.Message, "totally-absent.png") {
			flaggedAbsent = true
		}
		if strings.Contains(f.Message, "jules-logo.png") {
			flaggedBundled = true
		}
	}
	if !flaggedAbsent {
		t.Errorf("expected template reference to missing asset to be flagged, got: %v", resOn.Findings)
	}
	if flaggedBundled {
		t.Errorf("embedded runtime assets deploy on every build and must not be flagged: %v", resOn.Findings)
	}
}
