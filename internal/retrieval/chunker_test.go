package retrieval

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestChunkFileRespectsRenderFalse(t *testing.T) {
	body := "---\ntitle: Hidden\nrender: false\n---\n# Hidden\n\nThis page should not appear in the corpus.\n"
	if got := chunkFile(body, "content/_draft.md"); len(got) != 0 {
		t.Fatalf("render:false must yield zero chunks, got %d: %+v", len(got), got)
	}
}

// TestChunkFileRespectsRenderFalseQuoted asserts the same behaviour for
// the quoted YAML form, which is what goldmark-style frontmatter writers
// commonly produce for booleans.
func TestChunkFileRespectsRenderFalseQuoted(t *testing.T) {
	body := "---\ntitle: Hidden\nrender: \"false\"\n---\n# Hidden\n\nThis page should not appear in the corpus.\n"
	if got := chunkFile(body, "content/h.md"); len(got) != 0 {
		t.Fatalf("render:\"false\" must yield zero chunks, got %d", len(got))
	}
}

func TestChunkFileRenderDefaultsTrue(t *testing.T) {
	body := "---\ntitle: Public\n---\n# Public\n\nThis page should appear.\n"
	got := chunkFile(body, "content/x.md")
	if len(got) != 1 {
		t.Fatalf("missing render key should treat page as renderable, got %d chunks", len(got))
	}
	if got[0].Title != "Public" {
		t.Errorf("title = %q want Public", got[0].Title)
	}
}

func TestChunkFileRenderTruthyForm(t *testing.T) {
	body := "---\ntitle: Public\nrender: \"true\"\n---\n# Public\n\nText.\n"
	got := chunkFile(body, "content/x.md")
	if len(got) != 1 {
		t.Fatalf("render:true should yield 1 chunk, got %d", len(got))
	}
}

func TestChunkFileIDsAreSortedDeterministically(t *testing.T) {
	body := "---\ntitle: Multi\n---\n# Multi\n\n## Alpha\n\nfoo.\n\n## Beta\n\nbar.\n\n## Gamma\n\nbaz.\n"
	chunks := chunkFile(body, "content/multi.md")
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}
	sortedIDs := make([]string, 0, len(chunks))
	for _, c := range chunks {
		sortedIDs = append(sortedIDs, c.ID)
	}
	want := append([]string(nil), sortedIDs...)
	sort.Strings(want)
	for i, c := range chunks {
		if c.ID != want[i] {
			t.Errorf("chunk #%d ID %q out of order (want %q)", i, c.ID, want[i])
		}
	}
}

// TestLoaderExcludesRenderFalse exercises the loader end-to-end with the
// unquoted form of the render key.
func TestLoaderExcludesRenderFalse(t *testing.T) {
	dir := t.TempDir()
	publicBundle := `<file path="content/pub.md">
<content>
---
title: Pub
---
# Pub

visible.
</content>
</file>
`
	hiddenBundle := `<file path="content/hidden.md">
<content>
---
title: Hidden
render: false
---
# Hidden

hidden.
</content>
</file>
`
	path := filepath.Join(dir, "rag-content.md")
	if err := writeArtifact(path, publicBundle+hiddenBundle); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, c := range res.Corpus.Chunks {
		if strings.Contains(c.Text, "hidden.") {
			t.Errorf("render:false chunk leaked into corpus: %+v", c)
		}
	}
}

// TestLoaderExcludesRenderFalseQuoted exercises the loader end-to-end with
// the quoted YAML form of the render key. Regression-guards parseFrontmatter.
func TestLoaderExcludesRenderFalseQuoted(t *testing.T) {
	dir := t.TempDir()
	hidden := `<file path="content/hidden.md">
<content>
---
title: Hidden
render: "false"
---
# Hidden

quoted-hidden prose.
</content>
</file>
`
	public := `<file path="content/pub.md">
<content>
---
title: Pub
---
# Pub

visible.
</content>
</file>
`
	path := filepath.Join(dir, "rag-content.md")
	if err := writeArtifact(path, public+hidden); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res, err := Load(LoadOptions{RagDir: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, c := range res.Corpus.Chunks {
		if strings.Contains(c.Text, "quoted-hidden") {
			t.Errorf(`render:"false" chunk leaked through the loader: %+v`, c)
		}
	}
}
