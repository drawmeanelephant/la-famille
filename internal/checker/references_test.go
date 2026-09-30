package checker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tbuddy/la-famille/internal/content"
)

func TestExtractManifestReferencesCollectsLocalLinksAndAssets(t *testing.T) {
	contentDir := t.TempDir()
	files := map[string]string{
		"index.md": `---
title: Home
image: /assets/cover.png
---
![Relative image](images/hero.png)
[Resolved source](about.md?view=full#section)
[Resolved output](/about/)
[Missing source](missing.md)
[External asset](https://example.com/photo.png)
`,
		"about.md": "# About\n",
	}
	for name, body := range files {
		path := filepath.Join(contentDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fileMap, err := content.GatherMetadata(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	links, assets := ExtractManifestReferences(fileMap, false)
	if got, want := len(links["index.md"]), 3; got != want {
		t.Fatalf("len(links[index.md]) = %d, want %d: %+v", got, want, links["index.md"])
	}
	if got, want := []string{links["index.md"][0].Destination, links["index.md"][1].Destination, links["index.md"][2].Destination}, []string{"about.md?view=full#section", "/about/", "missing.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("link destinations = %v, want %v", got, want)
	}
	if !links["index.md"][0].Resolved || links["index.md"][0].GraphTarget != "about" {
		t.Errorf("resolved source link = %+v, want graph target about", links["index.md"][0])
	}
	if !links["index.md"][1].Resolved || links["index.md"][1].GraphTarget != "" {
		t.Errorf("resolved output link = %+v, want no graph target", links["index.md"][1])
	}
	if links["index.md"][2].Resolved || links["index.md"][2].GraphTarget != "missing" {
		t.Errorf("missing source link = %+v, want unresolved graph target missing", links["index.md"][2])
	}
	if got, want := assets["index.md"], []string{"cover.png", "images/hero.png"}; !reflect.DeepEqual(got, want) {
		t.Errorf("asset references = %v, want %v", got, want)
	}
}

func TestExtractManifestReferencesResolvesWikiTitlesAndMissingTargets(t *testing.T) {
	contentDir := t.TempDir()
	files := map[string]string{
		"index.md": "Known [[Wiki Links]] and unresolved [[Future Note]].\n",
		"wiki.md":  "---\ntitle: Wiki Links\n---\n# Wiki Links\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fileMap, err := content.GatherMetadata(contentDir)
	if err != nil {
		t.Fatal(err)
	}

	links, _ := ExtractManifestReferences(fileMap, false)
	if len(links["index.md"]) != 2 {
		t.Fatalf("wiki links = %+v, want 2", links["index.md"])
	}
	var resolved, unresolved bool
	for _, link := range links["index.md"] {
		switch link.Target {
		case "wiki.md":
			resolved = link.Resolved && link.GraphTarget == "wiki"
		case "future-note.md":
			unresolved = !link.Resolved && link.GraphTarget == "future-note"
		}
	}
	if !resolved {
		t.Errorf("known title wiki link missing or unresolved: %+v", links["index.md"])
	}
	if !unresolved {
		t.Errorf("unresolved wiki link missing or misclassified: %+v", links["index.md"])
	}
}
