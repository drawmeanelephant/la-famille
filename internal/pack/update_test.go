package pack

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func readTestFile(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeTestPack(t *testing.T, name string, entries ...testEntry) []byte {
	t.Helper()
	data := testArchive(t, manifestJSON(t, testManifest(entries...)), entries...)
	writeInput(t, name, data)
	return data
}

func archiveEntries(t testing.TB, data []byte) []testEntry {
	t.Helper()
	var entries []testEntry
	r := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := r.Next()
		if err == io.EOF {
			return entries
		}
		if err != nil {
			t.Fatal(err)
		}
		payload, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, testEntry{header.Name, payload})
	}
}

func rawArchive(t testing.TB, entries ...testEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, entry := range entries {
		if err := writeJSONMember(w, entry.name, entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDiffApplyMemberChanges(t *testing.T) {
	unchanged := testEntry{"a/unknown.bin", []byte{0, 255, 1}}
	b := testEntry{"b", []byte("before")}
	tests := []struct {
		name                    string
		before, after           []testEntry
		added, removed, changed []string
	}{
		{"no change", []testEntry{unchanged}, []testEntry{unchanged}, []string{}, []string{}, []string{}},
		{"addition", []testEntry{unchanged}, []testEntry{unchanged, b}, []string{"b"}, []string{}, []string{}},
		{"modification", []testEntry{unchanged, b}, []testEntry{unchanged, {"b", []byte("after")}}, []string{}, []string{}, []string{"b"}},
		{"removal", []testEntry{unchanged, b}, []testEntry{unchanged}, []string{}, []string{"b"}, []string{}},
		{"mixed", []testEntry{unchanged, b, {"c", []byte("removed")}},
			[]testEntry{unchanged, {"b", []byte{}}, {"d", []byte("new")}},
			[]string{"d"}, []string{"c"}, []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			before, after := filepath.Join(dir, "before.tar"), filepath.Join(dir, "after.tar")
			baseData := writeTestPack(t, before, tt.before...)
			targetData := writeTestPack(t, after, tt.after...)
			delta, repeated := filepath.Join(dir, "delta.tar"), filepath.Join(dir, "repeated.tar")
			report, err := Diff(before, after, delta)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report.Added, tt.added) || !reflect.DeepEqual(report.Removed, tt.removed) ||
				!reflect.DeepEqual(report.Changed, tt.changed) {
				t.Fatalf("report = %+v", report)
			}
			if report.BaseSHA256 != fmt.Sprintf("%x", sha256.Sum256(baseData)) {
				t.Fatal("delta did not bind to exact base bytes")
			}
			if _, err := Diff(before, after, repeated); err != nil {
				t.Fatal(err)
			}
			deltaData := readTestFile(t, delta)
			if !bytes.Equal(deltaData, readTestFile(t, repeated)) {
				t.Fatal("repeated delta is not byte-identical")
			}
			entries := archiveEntries(t, deltaData)
			if len(entries) != 2+len(tt.added)+len(tt.changed) ||
				entries[0].name != DeltaName || entries[1].name != ManifestName {
				t.Fatalf("delta inventory = %+v", entries)
			}
			if !bytes.Equal(entries[1].data, archiveEntries(t, targetData)[0].data) {
				t.Fatal("delta did not retain the complete target manifest bytes")
			}
			for _, entry := range entries[2:] {
				if bytes.Equal(entry.data, unchanged.data) {
					t.Fatal("unchanged payload leaked into delta")
				}
			}
			result := filepath.Join(dir, "result.tar")
			if _, err := Apply(before, delta, result); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(readTestFile(t, result), targetData) {
				t.Fatal("applied pack differs from target bytes")
			}
			if !bytes.Equal(readTestFile(t, before), baseData) {
				t.Fatal("base was modified")
			}
			if _, err := VerifyFile(result); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeltaManifestOnlyChangesAndNameCollisions(t *testing.T) {
	dir := t.TempDir()
	before, after := filepath.Join(dir, "before"), filepath.Join(dir, "after")
	entries := []testEntry{
		{"pack-delta.json", []byte("unknown")},
		{"payload/0000", []byte("also unknown")},
	}
	writeTestPack(t, before, entries...)
	entries[0].data = []byte("replaced unknown")
	m := testManifest(entries...)
	m.Site.Name = "Renamed"
	m.Provenance.Version = "next"
	target := testArchive(t, manifestJSON(t, m), entries...)
	writeInput(t, after, target)
	delta, result := filepath.Join(dir, "delta"), filepath.Join(dir, "result")
	if _, err := Diff(before, after, delta); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(before, delta, result); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readTestFile(t, result), target) {
		t.Fatal("unknown names or producer/site changes not preserved")
	}
	// Site/provenance changes alone still replace the target manifest.
	delta2, result2 := filepath.Join(dir, "delta2"), filepath.Join(dir, "result2")
	m.Provenance.Version = "manifest-only"
	target2 := testArchive(t, manifestJSON(t, m), entries...)
	writeInput(t, filepath.Join(dir, "after2"), target2)
	report, err := Diff(after, filepath.Join(dir, "after2"), delta2)
	if err != nil || len(report.Added)+len(report.Removed)+len(report.Changed) != 0 {
		t.Fatalf("manifest-only diff = %+v, %v", report, err)
	}
	if len(archiveEntries(t, readTestFile(t, delta2))) != 2 {
		t.Fatal("manifest-only delta contained payloads")
	}
	if _, err := Apply(after, delta2, result2); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readTestFile(t, result2), target2) {
		t.Fatal("manifest-only changes lost")
	}
}

func TestPackLedgerComparison(t *testing.T) {
	dir := t.TempDir()
	a := sitedata.Manifest{Version: 2, OutputCaptured: true, Pages: []sitedata.ManifestPage{
		{Identity: "index", Title: "Index", Tags: []string{"old"}},
		{Identity: "page", Title: "Page"},
	}}
	b := a
	b.Pages = append([]sitedata.ManifestPage{}, a.Pages...)
	b.Pages[0].Tags = []string{"new"}
	b.Pages[0].OutboundLinks = []string{"page"}
	encode := func(m sitedata.Manifest) []byte {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	before, after := filepath.Join(dir, "before"), filepath.Join(dir, "after")
	writeTestPack(t, before, testEntry{"site-manifest.json", encode(a)})
	writeTestPack(t, after, testEntry{"site-manifest.json", encode(b)})
	report, err := Diff(before, after, filepath.Join(dir, "delta"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Ledger == nil || len(report.Ledger.AddedEdges) != 1 || len(report.Ledger.TaxonomyChanges) != 2 ||
		!strings.Contains(report.Summary(), "Graph edges: 1 added, 0 removed") {
		t.Fatalf("Ledger report = %+v", report)
	}
}

func TestSnapshotDoesNotReopenOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base")
	entry := testEntry{"future.bin", []byte("verified")}
	writeTestPack(t, path, entry)
	s, err := loadPack(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	// Replace the original input after verification.
	writeInput(t, path, []byte("untrusted replacement"))
	var out bytes.Buffer
	w := tar.NewWriter(&out)
	m := s.manifest.Members[0]
	if err := copyVerifiedMember(w, m.Path, m, s.member(m)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(archiveEntries(t, out.Bytes())[0].data, entry.data) {
		t.Fatal("copy consumed the reopened original")
	}
	// Even private snapshot bytes are checked again while consumed.
	if _, err := s.file.WriteAt([]byte("X"), s.offsets[m.Path]); err != nil {
		t.Fatal(err)
	}
	if err := copyVerifiedMember(tar.NewWriter(io.Discard), m.Path, m, s.member(m)); err == nil {
		t.Fatal("modified snapshot bytes copied without verification")
	}
}
