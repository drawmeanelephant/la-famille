package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func corpusEntries() []testEntry {
	return []testEntry{
		{"meta.json", []byte(`{"birds":{"title":"Bird Notes","url":"/guide/birds/"}}`)},
		{"rag-content.md", []byte("<file path=\"content/birds.md\">\n<content>\nMeadow bird migration counts.\n</content>\n</file>\n")},
	}
}

func corpusPack(t *testing.T, entries []testEntry) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "corpus.tar")
	writeInput(t, name, testArchive(t, manifestJSON(t, testManifest(entries...)), entries...))
	return name
}

func TestLoadCorpusSnapshotSurvivesPathReplacement(t *testing.T) {
	entries := corpusEntries()
	name := corpusPack(t, entries)
	s, err := loadPack(name)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	snapshotName := s.file.Name()
	// Replace the caller's path after verification. Consumption must use the
	// still-open verified snapshot, not this replacement.
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	writeInput(t, name, []byte("unverified replacement"))
	result, err := loadSnapshotCorpus(s, name)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Corpus.Chunks) != 1 || result.Corpus.Chunks[0].Title != "Bird Notes" ||
		result.Corpus.Chunks[0].URL != "/guide/birds/" {
		t.Fatalf("wrong snapshot corpus: %+v", result.Corpus)
	}
	s.close()
	if _, err := os.Stat(snapshotName); !os.IsNotExist(err) {
		t.Fatalf("snapshot not removed: %v", err)
	}
	if _, err := s.file.ReadAt(make([]byte, 1), 0); err == nil {
		t.Fatal("snapshot descriptor not closed")
	}
}

func TestLoadCorpusIgnoresNonCorpusMembers(t *testing.T) {
	base := corpusEntries()
	entries := []testEntry{
		base[0],
		{"rag-config.md", []byte("malformed unrelated config sentinel")},
		base[1],
		{"rag-system.md", []byte("malformed unrelated system sentinel")},
		{"tags/index.html", []byte("<script>unrelated executable sentinel</script>")},
	}
	got, err := LoadCorpus(corpusPack(t, entries))
	if err != nil {
		t.Fatal(err)
	}
	if got.Corpus.DocumentCount != 1 || got.Corpus.ChunkCount != 1 || strings.Contains(got.Corpus.Chunks[0].Text, "sentinel") {
		t.Fatalf("non-corpus members consumed: %+v", got.Corpus)
	}
}

func TestLoadCorpusErrorsAndCleanup(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	valid := corpusEntries()
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"corrupt", []byte("not an archive"), "verify corpus"},
		{"unsupported", func() []byte {
			m := testManifest(valid...)
			m.SchemaVersion = 2
			return testArchive(t, manifestJSON(t, m), valid...)
		}(), "unsupported pack schema"},
		{"missing content", func() []byte {
			entries := valid[:1]
			return testArchive(t, manifestJSON(t, testManifest(entries...)), entries...)
		}(), "requires rag-content.md"},
		{"malformed content", func() []byte {
			entries := []testEntry{{"rag-content.md", []byte("<file path=\"content/birds.md\">\n<content>\nunterminated")}}
			return testArchive(t, manifestJSON(t, testManifest(entries...)), entries...)
		}(), "unterminated"},
		{"malformed metadata", func() []byte {
			entries := corpusEntries()
			entries[0].data = []byte("{")
			return testArchive(t, manifestJSON(t, testManifest(entries...)), entries...)
		}(), "meta.json"},
		{"hash mismatch", func() []byte {
			m := testManifest(valid...)
			entries := corpusEntries()
			entries[1].data = []byte(strings.ReplaceAll(string(entries[1].data), "Meadow", "Bogus!"))
			return testArchive(t, manifestJSON(t, m), entries...)
		}(), "SHA256 mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "corpus.tar")
			writeInput(t, name, tt.data)
			if _, err := LoadCorpus(name); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadCorpus error = %v, want %q", err, tt.want)
			}
			assertNoCorpusSnapshots(t)
		})
	}
	if _, err := LoadCorpus(filepath.Join(t.TempDir(), "missing.tar")); err == nil {
		t.Fatal("missing pack accepted")
	}
	result, err := LoadCorpus(corpusPack(t, valid))
	if err != nil || result.Corpus.ChunkCount != 1 {
		t.Fatalf("valid corpus = %+v, %v", result, err)
	}
	assertNoCorpusSnapshots(t)
}

func assertNoCorpusSnapshots(t *testing.T) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "la-famille-pack-snapshot-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("leaked snapshots = %v, %v", matches, err)
	}
}
