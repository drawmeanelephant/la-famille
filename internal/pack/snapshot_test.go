package pack

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifiedSnapshotSurvivesPathReplacement(t *testing.T) {
	entries := []testEntry{
		{"meta.json", []byte(`{"birds":{"title":"Bird Notes","url":"/guide/birds/"}}`)},
		{"rag-content.md", []byte("<file path=\"content/birds.md\">\n<content>\nMeadow bird counts.\n</content>\n</file>\n")},
	}
	name := filepath.Join(t.TempDir(), "corpus.tar")
	writeInput(t, name, testArchive(t, manifestJSON(t, testManifest(entries...)), entries...))
	s, err := loadPack(name)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	snapshotName := s.file.Name()
	// Keep this shared snapshot guarantee independently of the retired reader.
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	writeInput(t, name, []byte("unverified replacement"))
	for i, member := range s.manifest.Members {
		data, err := io.ReadAll(s.member(member))
		if err != nil || member.Path != entries[i].name || !bytes.Equal(data, entries[i].data) {
			t.Fatalf("verified member %s changed: %q, %v", member.Path, data, err)
		}
	}
	s.close()
	if _, err := os.Stat(snapshotName); !os.IsNotExist(err) {
		t.Fatalf("snapshot not removed: %v", err)
	}
	if _, err := s.file.ReadAt(make([]byte, 1), 0); err == nil {
		t.Fatal("snapshot descriptor not closed")
	}
}
