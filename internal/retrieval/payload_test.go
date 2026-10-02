package retrieval

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const payloadBundle = `<file path="notes/research/birds.md">
<content>
Meadow bird counts measure migration.
</content>
</file>

<file path="notes/research/maps.md">
<content>
Contour maps guide meadow surveys.
</content>
</file>
`

func payloadArtifacts() map[string][]byte {
	return map[string][]byte{
		"meta.json":          []byte(`{"birds":{"title":"Bird Observation Notes","url":"/field-guide/birds/"},"maps":{"title":"Contour Map Notes","url":"/field-guide/maps/"}}`),
		"search.json":        []byte(`[]`),
		"site-manifest.json": []byte(`{"version":2,"pages":[{"source_path":"birds.md","identity":"birds"},{"source_path":"maps.md","identity":"maps"}]}`),
		"graph.json":         []byte(`{"nodes":{"birds":{"type":"page","render":true},"maps":{"type":"page","render":true}},"edges":[["birds","maps"]]}`),
		"backlinks.json":     []byte(`{"maps":["birds"]}`),
	}
}

func TestLoadPayloadMatchesDirectoryCorpus(t *testing.T) {
	dir := t.TempDir()
	artifacts := payloadArtifacts()
	for name, data := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "rag-content.md"), []byte(payloadBundle), 0600); err != nil {
		t.Fatal(err)
	}
	directory, err := Load(LoadOptions{RagDir: dir, OutputDir: dir, ContentDir: "notes/research"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := LoadPayload(dir, strings.NewReader(payloadBundle), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(payload.Corpus, directory.Corpus) {
		t.Fatalf("payload differs from directory corpus:\n%+v\n%+v", payload.Corpus, directory.Corpus)
	}
	if payload.Corpus.DocumentCount != 2 || payload.Corpus.ChunkCount != 2 || len(payload.Corpus.Graph.Edges) != 1 {
		t.Fatalf("unexpected corpus: %+v", payload.Corpus)
	}
	for _, chunk := range payload.Corpus.Chunks {
		if chunk.Title == "" || !strings.HasPrefix(chunk.URL, "/field-guide/") {
			t.Fatalf("metadata not enriched: %+v", chunk)
		}
	}
}

func TestLoadPayloadRejectsMalformedArtifacts(t *testing.T) {
	for _, name := range []string{"meta.json", "search.json", "graph.json", "backlinks.json", "site-manifest.json"} {
		t.Run(name, func(t *testing.T) {
			artifacts := payloadArtifacts()
			artifacts[name] = []byte("{")
			_, err := LoadPayload("fixture.tar", strings.NewReader(payloadBundle), artifacts)
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("error = %v, want artifact %s", err, name)
			}
		})
	}
	for _, name := range []string{"meta.json", "search.json", "graph.json", "backlinks.json", "site-manifest.json"} {
		t.Run(name+" null", func(t *testing.T) {
			artifacts := payloadArtifacts()
			artifacts[name] = []byte("null")
			_, err := LoadPayload("fixture.tar", strings.NewReader(payloadBundle), artifacts)
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("null artifact error = %v, want %s", err, name)
			}
		})
	}
	for _, body := range []string{
		"", "not a RAG bundle", "<file path=\"content/a.md\">\nbody",
		"<file path=\"content/a.md\">\n</content>\n</file>",
		"<file path=\"content/a.md\">\n<content>\nbody\n</content>",
		"<file path=\"content/a.md\">\n<content>\n<content>\nbody\n</content>\n</file>",
		"<content>\nbody\n</content>", "</file>",
		"<file path=\"content/a.md\">\n<content>\n\n</content>\n</file>",
		"<file path=\"content/a.md\">\n<content>garbage\nbody\n</content>\n</file>",
		"<file path=\"content/a.md\"\n<content>\nbody\n</content>\n</file>",
	} {
		t.Run(body, func(t *testing.T) {
			_, err := LoadPayload("fixture.tar", strings.NewReader(body), nil)
			if err == nil || !strings.Contains(err.Error(), "rag-content.md") {
				t.Fatalf("error = %v, want malformed content", err)
			}
		})
	}
}

func TestLoadPayloadPreservesGeneratedTitleAndSearchBackfill(t *testing.T) {
	bundle := strings.ReplaceAll(payloadBundle, "Meadow bird counts", "# Source heading\nMeadow bird counts")
	artifacts := payloadArtifacts()
	artifacts["meta.json"] = []byte(`{"birds":{"title":"Generated Bird Title","url":"/field-guide/birds/"},"maps":{"url":"/field-guide/maps/"}}`)
	artifacts["search.json"] = []byte(`[{"t":"Search Map Title","u":"/field-guide/maps/"}]`)
	got, err := LoadPayload("fixture.tar", strings.NewReader(bundle), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if got.Corpus.Chunks[0].Title != "Generated Bird Title" || got.Corpus.Chunks[1].Title != "Search Map Title" {
		t.Fatalf("generated titles not preserved: %+v", got.Corpus.Chunks)
	}
}

func TestLoadPayloadUsesManifestPageIdentity(t *testing.T) {
	for _, source := range []string{"notes/research/content/birds.md", "content/birds.md"} {
		bundle := strings.ReplaceAll(payloadBundle, "notes/research/birds.md", source)
		if source == "content/birds.md" {
			bundle = strings.ReplaceAll(bundle, "notes/research/maps.md", "maps.md")
		}
		artifacts := payloadArtifacts()
		artifacts["site-manifest.json"] = []byte(`{"version":2,"pages":[{"source_path":"birds.md","identity":"birds"},{"source_path":"content/birds.md","identity":"content/birds"},{"source_path":"maps.md","identity":"maps"}]}`)
		artifacts["meta.json"] = []byte(`{"content/birds":{"url":"/guide/nested-birds/"}}`)
		got, err := LoadPayload("fixture.tar", strings.NewReader(bundle), artifacts)
		if err != nil {
			t.Fatal(err)
		}
		if chunk, ok := got.Corpus.ChunkByID("content/birds#h0"); !ok || chunk.URL != "/guide/nested-birds/" || chunk.SourcePath != source {
			t.Fatalf("manifest identity lost: %+v", got.Corpus)
		}
	}
}

func TestLoadPayloadKeepsOverlappingSourcePathsDistinct(t *testing.T) {
	bundle := `<file path="notes/research/birds.md">
<content>
Root meadow bird observations.
</content>
</file>
<file path="notes/research/research/birds.md">
<content>
Nested migration bird observations.
</content>
</file>
`
	artifacts := map[string][]byte{
		"site-manifest.json": []byte(`{"version":2,"pages":[{"source_path":"birds.md","identity":"birds"},{"source_path":"research/birds.md","identity":"research/birds"}]}`),
		"meta.json":          []byte(`{"birds":{"title":"Root Birds","url":"/guide/root/"},"research/birds":{"title":"Nested Birds","url":"/guide/nested/"}}`),
	}
	got, err := LoadPayload("fixture.tar", strings.NewReader(bundle), artifacts)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"birds#h0": "Root Birds|/guide/root/", "research/birds#h0": "Nested Birds|/guide/nested/",
	} {
		chunk, ok := got.Corpus.ChunkByID(id)
		if !ok || chunk.Title+"|"+chunk.URL != want {
			t.Fatalf("overlapping source identity %s lost: %+v", id, got.Corpus.Chunks)
		}
	}
}

func TestLoadPayloadRejectsInconsistentManifestPaths(t *testing.T) {
	for _, pages := range []string{
		`[{"source_path":"birds.md","identity":"birds"},{"source_path":"research/birds.md","identity":"research/birds"}]`,
		`[{"source_path":"missing.md","identity":"missing"}]`,
		`[{"source_path":"birds.md","identity":"birds"},{"source_path":"birds.md","identity":"duplicate"}]`,
		`[{"source_path":"birds.md"}]`,
	} {
		artifacts := map[string][]byte{"site-manifest.json": []byte(`{"version":2,"pages":` + pages + `}`)}
		bundle := "<file path=\"notes/research/birds.md\">\n<content>\nMeadow bird observations.\n</content>\n</file>\n"
		if _, err := LoadPayload("fixture.tar", strings.NewReader(bundle), artifacts); err == nil ||
			!strings.Contains(err.Error(), "site-manifest.json") {
			t.Fatalf("incompatible manifest accepted: %s, %v", pages, err)
		}
	}
}
