package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/search"
)

func TestBuildExcludesUnpublishedNotesAndIncomingLinks(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "vault")
	templateDir := filepath.Join(root, "templates")
	for _, dir := range []string{contentDir, templateDir} {
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
---
[[Public Note]] and [[Private Note|private label]] and [private markdown](private.md).
`,
		"public.md": `---
title: Public Note
description: Public note
---
[[Home]]
`,
		"private.md": `---
title: Private Note
description: Private note
publish: false
---
Secret vault material.
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
	cfg.Template = filepath.Join(templateDir, "layout.html")
	if _, err := Build(cfg); err != nil {
		t.Fatalf("Build(): %v", err)
	}

	if _, err := os.Stat(filepath.Join(cfg.OutputDir, "private", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("unpublished note output exists: %v", err)
	}
	page := readOutput(t, cfg, "index.html")
	if !strings.Contains(page, "private label") {
		t.Fatalf("unpublished wiki-link label is missing from the source page:\n%s", page)
	}
	if !strings.Contains(page, "private markdown") {
		t.Fatalf("unpublished Markdown link label is missing from the source page:\n%s", page)
	}
	if strings.Contains(page, `href="private/`) {
		t.Fatalf("published page links into an excluded note:\n%s", page)
	}

	var graphData struct {
		Nodes map[string]json.RawMessage `json:"nodes"`
		Edges [][2]string                `json:"edges"`
	}
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "graph.json")), &graphData); err != nil {
		t.Fatalf("parse graph.json: %v", err)
	}
	if _, exists := graphData.Nodes["private"]; exists {
		t.Errorf("excluded note is in graph nodes: %s", graphData.Nodes["private"])
	}
	for _, edge := range graphData.Edges {
		if edge[1] == "private" || edge[0] == "private" {
			t.Errorf("excluded note appears in graph edge: %v", edge)
		}
	}

	var searchIndex []search.Item
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "search.json")), &searchIndex); err != nil {
		t.Fatalf("parse search.json: %v", err)
	}
	for _, item := range searchIndex {
		if strings.Contains(item.Title, "Private") || strings.Contains(item.Snippet, "Secret vault material") {
			t.Errorf("excluded note appears in search index: %+v", item)
		}
	}

	var backlinks map[string][]string
	if err := json.Unmarshal([]byte(readOutput(t, cfg, "backlinks.json")), &backlinks); err != nil {
		t.Fatalf("parse backlinks.json: %v", err)
	}
	if _, exists := backlinks["private"]; exists {
		t.Errorf("backlinks output contains excluded target: %v", backlinks["private"])
	}
}
