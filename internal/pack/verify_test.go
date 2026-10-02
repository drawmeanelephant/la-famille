package pack

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testEntry struct {
	name string
	data []byte
}

func testManifest(entries ...testEntry) Manifest {
	m := Manifest{
		SchemaVersion: SchemaVersion,
		Site:          Site{Name: "Test site"},
		Provenance:    Provenance{Generator: "la-famille", Version: "dev"},
	}
	for _, entry := range entries {
		m.Members = append(m.Members, Member{
			Path: entry.name, Size: int64(len(entry.data)),
			SHA256: fmt.Sprintf("%x", sha256.Sum256(entry.data)),
		})
	}
	m.ContentRoot = contentRoot(m.Members)
	return m
}

func manifestJSON(t testing.TB, m Manifest) []byte {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testArchive(t testing.TB, data []byte, entries ...testEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	for _, entry := range append([]testEntry{{name: ManifestName, data: data}}, entries...) {
		if err := writeHeader(writer, entry.name, int64(len(entry.data))); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func wantVerifyError(t *testing.T, archive []byte, want string) {
	t.Helper()
	if _, err := Verify(bytes.NewReader(archive)); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Verify error = %v, want %q", err, want)
	}
}

func TestVerifyUnknownListedMember(t *testing.T) {
	// The builder allowlist is deliberately not the verifier's schema policy.
	entry := testEntry{"future/unknown.bin", []byte{0, 1, 2, 255}}
	m := testManifest(entry)
	got, err := Verify(bytes.NewReader(testArchive(t, manifestJSON(t, m), entry)))
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentRoot != m.ContentRoot || got.Members[0] != m.Members[0] {
		t.Fatalf("verified manifest = %+v, want %+v", got, m)
	}
	m.SchemaVersion = 2
	wantVerifyError(t, testArchive(t, manifestJSON(t, m), entry), "unsupported pack schema version 2")
}

func TestVerifyTamperedMemberReportsBothHashes(t *testing.T) {
	entry := testEntry{"graph.json", []byte("original")}
	m := testManifest(entry)
	entry.data = []byte("tampered")
	actual := fmt.Sprintf("%x", sha256.Sum256(entry.data))
	_, err := Verify(bytes.NewReader(testArchive(t, manifestJSON(t, m), entry)))
	if err == nil {
		t.Fatal("tampered pack verified")
	}
	for _, want := range []string{entry.name, "expected " + m.Members[0].SHA256, "actual " + actual} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, missing %q", err, want)
		}
	}
}

func TestVerifyMemberInventory(t *testing.T) {
	entry := testEntry{"graph.json", []byte("data")}
	data := manifestJSON(t, testManifest(entry))
	tests := []struct {
		name    string
		entries []testEntry
		want    string
	}{
		{"missing", nil, `missing archive member "graph.json"`},
		{"unlisted", []testEntry{{"extra.json", []byte("data")}}, `unlisted archive member "extra.json"`},
		{"duplicate", []testEntry{entry, entry}, `duplicate archive member "graph.json"`},
		{"duplicate manifest", []testEntry{{ManifestName, data}}, "duplicate archive member"},
		{"wrong size", []testEntry{{entry.name, []byte("longer")}}, "size mismatch: expected 4, actual 6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantVerifyError(t, testArchive(t, data, tt.entries...), tt.want)
		})
	}
}

func TestVerifyUnsafePaths(t *testing.T) {
	for _, name := range []string{
		"../graph.json", "/graph.json", "a/../graph.json", "./graph.json",
		"a//graph.json", "a\\graph.json", "C:/graph.json",
		"a/\nindex.html", ".", "a/\x7findex.html",
	} {
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			entry := testEntry{name, []byte("data")}
			m := testManifest(entry)
			wantVerifyError(t, testArchive(t, manifestJSON(t, m)), "unsafe member path")
			// Also reject unsafe headers with an otherwise valid manifest.
			valid := testManifest(testEntry{"graph.json", []byte("data")})
			wantVerifyError(t, testArchive(t, manifestJSON(t, valid), entry), "unsafe member path")
		})
	}
	for _, name := range []string{"", "graph.json/", strings.Repeat("a", MaxPathBytes+1), "a/\xff"} {
		if err := validatePath(name); err == nil {
			t.Fatalf("accepted unsafe path %q", name)
		}
	}
}

func TestVerifyMalformedManifests(t *testing.T) {
	entry := testEntry{"graph.json", []byte("data")}
	m := testManifest(entry)
	valid := string(manifestJSON(t, m))
	tests := []struct {
		name string
		data string
		want string
	}{
		{"invalid JSON", "{", "malformed manifest"},
		{"array", "[]", "malformed manifest"},
		{"trailing JSON", valid + "{}", "trailing JSON"},
		{"duplicate key", strings.Replace(valid, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1), "duplicate"},
		{"nested duplicate", strings.Replace(valid, `"name":"Test site"`, `"name":"Test site","name":"Other"`, 1), "duplicate"},
		{"unknown field", strings.Replace(valid, `"schema_version":1`, `"schema_version":1,"typo":1`, 1), "unknown field"},
		{"case alias", strings.Replace(valid, `"size":4`, `"size":4,"Size":4`, 1), "unknown field"},
		{"provenance alias", strings.Replace(valid, `"version":"dev"`, `"version":"dev","Version":"dev"`, 1), "unknown field"},
		{"site alias", strings.Replace(valid, `"name":"Test site"`, `"name":"Test site","Name":"Test site"`, 1), "unknown field"},
		{"null", strings.Replace(valid, `"size":4`, `"size":null`, 1), "null is not allowed"},
		{"missing size", strings.Replace(valid, `"size":4,`, "", 1), `missing field "size"`},
		{"missing url", strings.Replace(valid, `,"url":""`, "", 1), `missing field "url"`},
		{"missing root", strings.Replace(valid, `,"content_root":"`+m.ContentRoot+`"`, "", 1), `missing field "content_root"`},
		{"missing version", strings.Replace(valid, `"schema_version":1,`, "", 1), `missing field "schema_version"`},
		{"wrong type", strings.Replace(valid, `"size":4`, `"size":"4"`, 1), "malformed manifest"},
		{"nonintegral size", strings.Replace(valid, `"size":4`, `"size":4.1`, 1), "malformed manifest"},
		{"UTF-8", "{\"\xff\":1}", "not UTF-8"},
		{"deep JSON", strings.Repeat("[", 34) + "1" + strings.Repeat("]", 34), "nesting"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantVerifyError(t, testArchive(t, []byte(tt.data), entry), tt.want)
		})
	}
}

func TestVerifyManifestValidation(t *testing.T) {
	entry := testEntry{"graph.json", []byte("data")}
	tests := []struct {
		name   string
		change func(*Manifest)
		want   string
	}{
		{"empty name", func(m *Manifest) { m.Site.Name = "" }, "requires site name"},
		{"empty generator", func(m *Manifest) { m.Provenance.Generator = "" }, "producer"},
		{"empty version", func(m *Manifest) { m.Provenance.Version = "" }, "producer"},
		{"no members", func(m *Manifest) { m.Members = []Member{} }, "member count"},
		{"duplicate", func(m *Manifest) { m.Members = append(m.Members, m.Members[0]) }, "duplicate"},
		{"reserved", func(m *Manifest) { m.Members[0].Path = ManifestName }, "reserved"},
		{"bad hash", func(m *Manifest) { m.Members[0].SHA256 = "invalid" }, "malformed SHA256"},
		{"uppercase hash", func(m *Manifest) { m.Members[0].SHA256 = strings.ToUpper(m.Members[0].SHA256) }, "malformed SHA256"},
		{"bad root", func(m *Manifest) { m.ContentRoot = strings.Repeat("0", 64) }, "content root mismatch"},
		{"negative size", func(m *Manifest) { m.Members[0].Size = -1 }, "size bound"},
		{"oversized member", func(m *Manifest) { m.Members[0].Size = MaxMemberSize + 1 }, "size bound"},
		{"unsorted", func(m *Manifest) {
			m.Members = append(m.Members, Member{Path: "a.json", SHA256: m.Members[0].SHA256})
		}, "sorted"},
		{"too many members", func(m *Manifest) {
			m.Members = make([]Member, MaxMembers+1)
		}, "member count"},
		{"total bound", func(m *Manifest) {
			m.Members = nil
			for i := 0; i < 5; i++ {
				m.Members = append(m.Members, Member{Path: fmt.Sprintf("%d", i), Size: MaxMemberSize, SHA256: strings.Repeat("0", 64)})
			}
		}, "total size bound"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testManifest(entry)
			tt.change(&m)
			wantVerifyError(t, testArchive(t, manifestJSON(t, m), entry), tt.want)
		})
	}
}

func TestVerifyMalformedArchives(t *testing.T) {
	entry := testEntry{"graph.json", []byte("data")}
	valid := testArchive(t, manifestJSON(t, testManifest(entry)), entry)
	manifestHeader := func(name string, size int64, kind byte, format tar.Format) []byte {
		var out bytes.Buffer
		writer := tar.NewWriter(&out)
		if err := writer.WriteHeader(&tar.Header{Name: name, Size: size, Typeflag: kind, Format: format, Linkname: ""}); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	badChecksum := append([]byte(nil), valid...)
	badChecksum[0] ^= 1
	nonzeroPadding := append([]byte(nil), valid...)
	nonzeroPadding[512+len(manifestJSON(t, testManifest(entry)))] = 1
	secondHeader := 512 + ((len(manifestJSON(t, testManifest(entry)))+511)/512)*512
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"empty", nil, "truncated archive"},
		{"random", bytes.Repeat([]byte("x"), 512), "USTAR"},
		{"checksum", badChecksum, "invalid tar header"},
		{"no end blocks", valid[:len(valid)-1024], "truncated archive"},
		{"one end block", valid[:len(valid)-512], "two zero"},
		{"garbage trailer", append(append([]byte{}, valid...), 1), "trailing archive data"},
		{"extra zero block", append(append([]byte{}, valid...), make([]byte, 512)...), "trailing archive data"},
		{"truncated manifest", valid[:520], "truncated manifest"},
		{"truncated padding", valid[:512+len(manifestJSON(t, testManifest(entry)))], "truncated member padding"},
		{"nonzero padding", nonzeroPadding, "nonzero member padding"},
		{"truncated payload", valid[:secondHeader+512+2], "expected 4, actual 2"},
		{"not manifest first", manifestHeader("graph.json", 0, tar.TypeReg, tar.FormatUSTAR), "first member"},
		{"manifest bound", manifestHeader(ManifestName, MaxManifestSize+1, tar.TypeReg, tar.FormatUSTAR), "at most"},
		{"directory", manifestHeader("directory/", 0, tar.TypeDir, tar.FormatUSTAR), "regular USTAR"},
		{"symlink", manifestHeader("symlink", 0, tar.TypeSymlink, tar.FormatUSTAR), "regular USTAR"},
		{"GNU", manifestHeader(ManifestName, 0, tar.TypeReg, tar.FormatGNU), "regular USTAR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { wantVerifyError(t, tt.data, tt.want) })
	}
	// Raw extension headers are rejected before allocating or reading bodies.
	for _, kind := range []byte{tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNUSparse, tar.TypeLink, tar.TypeFifo} {
		header := append([]byte(nil), valid[:512]...)
		header[156] = kind
		wantVerifyError(t, header, "regular USTAR")
	}
}

func TestVerifyFileBoundsAndErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := VerifyFile(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing file verified")
	}
	if _, err := VerifyFile(dir); err == nil {
		t.Fatal("directory verified")
	}
	name := filepath.Join(dir, "huge.tar")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxArchiveSize + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(name); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("oversized file error = %v", err)
	}
}

func TestContentRootKnownEncoding(t *testing.T) {
	members := []Member{{Path: "a", Size: 0, SHA256: strings.Repeat("0", 64)}}
	data := `[{"path":"a","size":0,"sha256":"` + strings.Repeat("0", 64) + `"}]`
	want := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
	if got := contentRoot(members); got != want {
		t.Fatalf("root = %s, want %s", got, want)
	}
}

func FuzzVerify(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte("x"), 512))
	entry := testEntry{"future/empty.bin", []byte{}}
	f.Add(testArchive(f, manifestJSON(f, testManifest(entry)), entry))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxManifestSize {
			t.Skip()
		}
		_, _ = Verify(bytes.NewReader(data))
	})
}
