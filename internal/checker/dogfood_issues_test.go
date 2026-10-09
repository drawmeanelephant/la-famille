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

// Issue #647: a link whose target the build publishes as a generated
// "Missing Page" stub ([missing](missing.md), [[Future Note]]) resolves in the
// artifact, so it is at most a warning — matching publish-check semantics —
// never a broken-link error. Output-style links also resolve once any page
// causes the stub to be written.
func TestValidateStubLinksWarnNotError(t *testing.T) {
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
[Missing source](missing.md)
[[Future Note]]
[Stub via output](/missing)
[Stub via output dir](/missing/)
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(indexDoc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.SiteURL = "https://example.com"

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.ErrorCount() != 0 {
		t.Fatalf("links resolving to stub outputs must not be errors, got: %v", res.Findings)
	}
	var stubWarnings int
	for _, f := range res.Findings {
		if f.Level == LevelWarn && strings.Contains(f.Message, "Missing Page") && f.File == "index.md" {
			stubWarnings++
		}
	}
	if stubWarnings != 2 {
		t.Errorf("expected 2 stub warnings (missing.md and Future Note), got %d: %v", stubWarnings, res.Findings)
	}
}

// Issue #647: an extensionless link to a page no stub covers still 404s —
// it must remain an error.
func TestValidateExtensionlessMissingLinkStillErrors(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}
	doc := "---\ntitle: Home\ndescription: d\n---\n[missing](/missing)\n"
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.SiteURL = "https://example.com"

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	found := false
	for _, f := range res.Findings {
		if f.Level == LevelError && f.Category == CategoryBrokenLink && strings.Contains(f.Message, `"/missing"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a broken-link error for /missing (no stub is generated for it), got: %v", res.Findings)
	}
}

// Issue #648: a link that escapes the content root ([escape](../../outside.md)
// or its %2e%2e-encoded form) ships verbatim into the artifact where it 404s —
// check must report it instead of skipping it silently.
func TestValidateFlagsRootEscapingLinks(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	doc := `---
title: Home
description: home
date: 2026-08-25
---
[escape](../../outside.md)
[enc](%2e%2e/outside.md)
`
	if err := os.WriteFile(filepath.Join(contentDir, "index.md"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.SiteURL = "https://example.com"

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	var escaped, encoded bool
	for _, f := range res.Findings {
		if f.File != "index.md" || f.Category != CategoryBrokenLink {
			continue
		}
		if strings.Contains(f.Message, "../../outside.md") {
			escaped = true
		}
		if strings.Contains(f.Message, "%2e%2e/outside.md") {
			encoded = true
		}
	}
	if !escaped {
		t.Errorf("root-escaping link ../../outside.md not reported: %v", res.Findings)
	}
	if !encoded {
		t.Errorf("percent-encoded escaping link %%2e%%2e/outside.md not reported: %v", res.Findings)
	}
}

// Issue #648: output-tree links that climb above the site root are clamped to
// the root by browsers — the checker resolves them the same way, so
// ../../real-page resolves while ../../nope is a broken-link finding.
func TestValidateOutputRefClampsAtOutputRoot(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "blog"), 0755); err != nil {
		t.Fatal(err)
	}

	blogPost := `---
title: Post
description: post
date: 2026-08-25
---
[clamped hit](../../about)
[clamped miss](../../nope)
`
	aboutDoc := `---
title: About
description: about
date: 2026-08-25
---
[Home](/)
`
	if err := os.WriteFile(filepath.Join(contentDir, "blog", "post.md"), []byte(blogPost), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "about.md"), []byte(aboutDoc), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.SiteURL = "https://example.com"

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	var miss bool
	for _, f := range res.Findings {
		if f.Category != CategoryBrokenLink {
			continue
		}
		if strings.Contains(f.Message, `"../../about"`) {
			t.Errorf("../../about resolves to /about via root clamping and must not be flagged: %s", f.Message)
		}
		if strings.Contains(f.Message, `"../../nope"`) {
			miss = true
		}
	}
	if !miss {
		t.Errorf("expected broken-link finding for ../../nope, got: %v", res.Findings)
	}
}

// Issue #652: a page linked only via its output URL ([c](/c), [c](c.html),
// or a slug-aliased path) is not orphaned — the build resolves and publishes
// the link, so it counts as an inbound reference.
func TestValidateOutputStyleLinksCountAsInbound(t *testing.T) {
	tempDir := t.TempDir()
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"index.md": `---
title: Home
description: home
date: 2026-08-25
---
[c](/c)
[aliased](/aliased)
[clamped](../c-deeper)
`,
		"c.md":        "---\ntitle: C\ndescription: d\n---\nx\n",
		"hidden.md":   "---\ntitle: Aliased\ndescription: d\nslug: aliased\n---\nx\n",
		"c-deeper.md": "---\ntitle: Deeper\ndescription: d\n---\nx\n",
		"sub/page.md": "---\ntitle: Sub\ndescription: d\n---\n[deeper](../../c-deeper)\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(contentDir, filepath.FromSlash(name)), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ContentDir = contentDir
	cfg.SiteURL = "https://example.com"

	res, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	for _, f := range res.Findings {
		if f.Category != CategoryOrphan {
			continue
		}
		switch f.File {
		case "c.md", "hidden.md", "c-deeper.md":
			t.Errorf("page reachable via output-style link reported as orphan: %s", f)
		}
	}
}
