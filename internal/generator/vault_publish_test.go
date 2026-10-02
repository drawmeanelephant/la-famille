package generator

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/search"
	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestBuildExcludesUnpublishedNotesAndIncomingLinks(t *testing.T) {
	const canary = "VAULT582_UNPUBLISHED_BODY_CANARY_4d1f9a"
	root := t.TempDir()
	contentDir := filepath.Join(root, "vault")
	templateDir := filepath.Join(root, "templates")
	assetDir := filepath.Join(root, "assets")
	for _, dir := range []string{contentDir, templateDir, assetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	layout := `<!doctype html><html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`
	if err := os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte(layout), 0600); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"index.md": `---
title: Home
description: Home
date: 2026-09-01
tags: [shared]
---
[[Public Note]] and [[Private Note|private label]] and [private markdown](private.md).
[[Private Raw Note|raw label]] and [raw markdown](private-raw.md).
[public after exclusions](public.md).
`,
		"public.md": `---
title: Public Note
description: Public note
date: 2026-09-02
tags: [shared]
---
[[Home]]
`,
		"private.md": `---
title: Private Note
description: Private note
date: 2026-09-03
tags: [shared, unpublished-frontmatter-tag]
categories: [unpublished-category]
publish: false
---
` + canary + ` #unpublished-body-tag [[Public Note]].
`,
		"private-raw.md": `---
title: Private Raw Note
date: 2026-09-04
render: false
publish: false
---
` + canary + ` [[Home]].
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cfg.ContentDir = contentDir
	cfg.OutputDir = filepath.Join(root, "public")
	cfg.AssetDir = assetDir
	cfg.Template = filepath.Join(templateDir, "layout.html")
	cfg.SiteURL = "https://example.test/vault"
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build(): %v", err)
	}

	for _, path := range []string{
		"private/index.html", "private.md", "private-raw/index.html", "private-raw.md",
		"tags/unpublished-frontmatter-tag/index.html",
		"tags/unpublished-body-tag/index.html",
		"categories/unpublished-category/index.html",
	} {
		if _, err := os.Stat(filepath.Join(cfg.OutputDir, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("unpublished output %q exists: %v", path, err)
		}
	}
	page := readOutput(t, cfg, "index.html")
	if !strings.Contains(page, "private label") {
		t.Fatalf("unpublished wiki-link label is missing from the source page:\n%s", page)
	}
	if !strings.Contains(page, "private markdown") {
		t.Fatalf("unpublished Markdown link label is missing from the source page:\n%s", page)
	}
	for _, label := range []string{"raw label", "raw markdown"} {
		if !strings.Contains(page, label) {
			t.Errorf("unpublished raw-note label %q is missing from the source page", label)
		}
	}
	if !strings.Contains(page, `href="public/" rel="nofollow">public after exclusions</a>`) {
		t.Errorf("public link after exclusions was not transformed:\n%s", page)
	}
	for _, href := range []string{`href="private/`, `href="private.md`, `href="private-raw/`, `href="private-raw.md`} {
		if strings.Contains(page, href) {
			t.Errorf("published page links into an excluded note: %s", href)
		}
	}

	var graphData struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
		Edges [][2]string                `json:"edges"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &graphData); err != nil {
		t.Fatalf("parse graph.json: %v", err)
	}
	excludedID := func(id string) bool {
		return id == "private" || id == "private-raw" || id == "private-raw.md"
	}
	for id := range graphData.Nodes {
		if excludedID(id) {
			t.Errorf("excluded note is in graph nodes: %s", id)
		}
	}
	for _, edge := range graphData.Edges {
		if excludedID(edge[1]) || excludedID(edge[0]) {
			t.Errorf("excluded note appears in graph edge: %v", edge)
		}
	}

	var searchIndex []search.Item
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "search.json")), &searchIndex); err != nil {
		t.Fatalf("parse search.json: %v", err)
	}
	for _, item := range searchIndex {
		if strings.Contains(item.Title, "Private") || strings.Contains(item.Snippet, canary) ||
			item.URL == "/vault/private/" || item.URL == "/vault/private-raw.md" {
			t.Errorf("excluded note appears in search index: %+v", item)
		}
	}

	var backlinks map[string][]string
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "backlinks.json")), &backlinks); err != nil {
		t.Fatalf("parse backlinks.json: %v", err)
	}
	for target, sources := range backlinks {
		if excludedID(target) {
			t.Errorf("backlinks output contains excluded target: %s", target)
		}
		for _, source := range sources {
			if excludedID(source) {
				t.Errorf("backlinks output contains excluded source: %s", source)
			}
		}
	}

	var meta map[string]json.RawMessage
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "meta.json")), &meta); err != nil {
		t.Fatalf("parse meta.json: %v", err)
	}
	for id := range meta {
		if excludedID(id) {
			t.Errorf("metadata contains excluded note: %s", id)
		}
	}
	manifest, err := sitedata.ReadManifest(filepath.Join(cfg.OutputDir, sitedata.ManifestFileName))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	for _, note := range manifest.Pages {
		if excludedID(note.Identity) || note.SourcePath == "private.md" || note.SourcePath == "private-raw.md" {
			t.Errorf("manifest contains excluded note: %s", note.SourcePath)
		}
		for _, link := range note.Links {
			if excludedID(link.GraphTarget) || link.Target == "private.md" || link.Target == "private-raw.md" {
				t.Errorf("manifest contains link into excluded note: %+v", link)
			}
		}
	}

	archive := readOutput(t, cfg, "tags/shared/index.html")
	for _, title := range []string{"Home", "Public Note"} {
		if !strings.Contains(archive, title) {
			t.Errorf("shared taxonomy archive missing published note %q", title)
		}
	}
	if strings.Contains(archive, "Private") {
		t.Error("shared taxonomy archive contains an unpublished note")
	}
	for path, publicMarker := range map[string]string{
		"feed.xml":    "<link>https://example.test/vault/public/</link>",
		"sitemap.xml": "<loc>https://example.test/vault/public/</loc>",
	} {
		output := readOutput(t, cfg, path)
		if !strings.Contains(output, publicMarker) {
			t.Errorf("%s missing published URL", path)
		}
		if strings.Contains(output, "/vault/private") {
			t.Errorf("%s contains an unpublished URL", path)
		}
	}

	if err := filepath.WalkDir(cfg.OutputDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		output, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Public source labels are intentionally retained. Only content from
		// the excluded notes, not those labels, constitutes a leak here.
		for _, marker := range []string{canary, "unpublished-frontmatter-tag", "unpublished-body-tag", "unpublished-category"} {
			if strings.Contains(string(output), marker) {
				t.Errorf("excluded-note marker %q leaked into %s", marker, path)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("scan generated output: %v", err)
	}
}
