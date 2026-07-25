package retrieval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCorpusFixture builds a minimal archive plus the generated meta.json a
// real build would sit beside it.
func writeCorpusFixture(t *testing.T, bundle, meta string) (ragDir, outDir string) {
	t.Helper()
	root := t.TempDir()
	ragDir = filepath.Join(root, "rag-archive")
	outDir = filepath.Join(root, "public")
	for _, d := range []string{ragDir, outDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(ragDir, "rag-content.md"), []byte(bundle), 0o600); err != nil {
		t.Fatal(err)
	}
	if meta != "" {
		if err := os.WriteFile(filepath.Join(outDir, "meta.json"), []byte(meta), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return ragDir, outDir
}

// TestCitationURLComesFromTheGenerator is the whole point of the enrichment.
// Chunk URLs used to be invented from the source filename, which cannot know
// about a frontmatter slug or a siteurl base path — so every citation of the
// homepage, and of any page with a slug, pointed somewhere that was never
// published.
//
// Three separate defects had to be fixed before this could pass: the
// enrichment ran before the chunks existed, it decoded a meta.json shape the
// generator never wrote, and the merge was a backfill that could not fire
// because chunkFile always sets a URL.
func TestCitationURLComesFromTheGenerator(t *testing.T) {
	bundle := `<file path="pages/index.md">
<content>
---
title: Acme Handbook
---
# Acme Handbook
Staff accrue twenty days a year.
</content>
</file>

<file path="pages/docs/onboarding.md">
<content>
---
title: Onboarding
slug: getting-started
---
# Onboarding
Day one checklist.
</content>
</file>
`
	meta := `{
  "index": {"title": "Acme Handbook", "url": "/handbook/", "render": true},
  "docs/onboarding": {"title": "Onboarding", "url": "/handbook/docs/getting-started/", "render": true}
}`

	ragDir, outDir := writeCorpusFixture(t, bundle, meta)

	res, err := Load(LoadOptions{RagDir: ragDir, OutputDir: outDir, ContentDir: "pages"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	want := map[string]string{
		"index":           "/handbook/",
		"docs/onboarding": "/handbook/docs/getting-started/",
	}
	seen := map[string]string{}
	for _, c := range res.Corpus.Chunks {
		seen[c.PageID] = c.URL
	}
	for id, url := range want {
		got, ok := seen[id]
		if !ok {
			t.Errorf("page %q produced no chunks; page ids must match meta.json keys", id)
			continue
		}
		if got != url {
			t.Errorf("page %q URL = %q, want the generator's %q", id, got, url)
		}
	}
}

// TestPageIDUsesTheConfiguredContentDirectory pins the join key. Archive paths
// are project-root relative, so a site with content_dir "pages" records
// "pages/index.md"; stripping a hardcoded "content/" left the page id as
// "pages/index", which matches nothing in meta.json and silently disabled the
// enrichment entirely.
func TestPageIDUsesTheConfiguredContentDirectory(t *testing.T) {
	cases := []struct {
		source, contentDir, want string
	}{
		{"content/index.md", "", "index"},
		{"content/index.md", "content", "index"},
		{"pages/index.md", "pages", "index"},
		{"pages/docs/onboarding.md", "pages", "docs/onboarding"},
		{"src/site/a.md", "src/site", "a"},
		{"pages/index.md", "", "pages/index"}, // wrong dir configured: no strip
	}
	for _, c := range cases {
		if got := derivePageID(c.source, c.contentDir); got != c.want {
			t.Errorf("derivePageID(%q, %q) = %q, want %q", c.source, c.contentDir, got, c.want)
		}
	}
}

// TestDirectoryListingsAreNotCorpusDocuments covers the archive's inventory
// blocks. ragexport records directories as <file path="assets/"> whose body is
// a listing of names and sizes; chunked as prose they answered questions with a
// file listing, ranked ahead of real pages, and carried a fabricated citation
// URL of "/assets//" with no title.
func TestDirectoryListingsAreNotCorpusDocuments(t *testing.T) {
	bundle := `<file path="assets/">
<content>
assets/css/
assets/css/site.css (size: 42 bytes)
assets/img/logo.png (size: 9001 bytes)
</content>
</file>

<file path="content/index.md">
<content>
---
title: Home
---
# Home
Real prose that should be indexed.
</content>
</file>
`
	ragDir, _ := writeCorpusFixture(t, bundle, "")

	res, err := Load(LoadOptions{RagDir: ragDir})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	for _, c := range res.Corpus.Chunks {
		if strings.HasSuffix(c.SourcePath, "/") {
			t.Errorf("a directory listing entered the corpus: id=%q url=%q", c.ID, c.URL)
		}
		if strings.Contains(c.URL, "//") {
			t.Errorf("chunk %q has a malformed citation URL %q", c.ID, c.URL)
		}
	}
	if len(res.Corpus.Chunks) == 0 {
		t.Fatal("the real page should still be indexed")
	}
	if res.Corpus.DocumentCount != 1 {
		t.Errorf("DocumentCount = %d, want 1 (the listing is not a document)", res.Corpus.DocumentCount)
	}
}
