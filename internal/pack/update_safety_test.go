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
	"strings"
	"testing"
)

func updatePair(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	base, target, delta := filepath.Join(dir, "base.tar"), filepath.Join(dir, "target.tar"), filepath.Join(dir, "delta.tar")
	writeTestPack(t, base, testEntry{"a", []byte("unchanged")}, testEntry{"b", []byte("old")}, testEntry{"c", []byte("removed")})
	writeTestPack(t, target, testEntry{"a", []byte("unchanged")}, testEntry{"b", []byte("new")}, testEntry{"d", []byte("added")})
	if _, err := Diff(base, target, delta); err != nil {
		t.Fatal(err)
	}
	return base, target, delta
}

func replaceDeltaMetadata(t *testing.T, data []byte, change func(*DeltaMetadata)) []byte {
	t.Helper()
	entries := archiveEntries(t, data)
	var d DeltaMetadata
	if err := json.Unmarshal(entries[0].data, &d); err != nil {
		t.Fatal(err)
	}
	change(&d)
	var err error
	entries[0].data, err = json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return rawArchive(t, entries...)
}

func wantApplyFailure(t *testing.T, base, delta, want string) {
	t.Helper()
	before := readTestFile(t, base)
	outputDir := t.TempDir()
	result := filepath.Join(outputDir, "result.tar")
	_, err := Apply(base, delta, result)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Apply error = %v, want %q", err, want)
	}
	if !bytes.Equal(readTestFile(t, base), before) {
		t.Fatal("failed apply modified the base")
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed apply left output files: %v, %v", entries, err)
	}
}

func TestApplyRejectsInvalidDeltas(t *testing.T) {
	base, _, delta := updatePair(t)
	valid := readTestFile(t, delta)
	mutateEntry := func(change func([]testEntry)) []byte {
		entries := archiveEntries(t, valid)
		change(entries)
		return rawArchive(t, entries...)
	}
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"corrupted payload", mutateEntry(func(e []testEntry) { e[2].data[0] ^= 1 }), "SHA256 mismatch"},
		{"corrupted target manifest", mutateEntry(func(e []testEntry) { e[1].data[0] ^= 1 }), "target manifest SHA256 mismatch"},
		{"truncated metadata", valid[:520], "truncated delta"},
		{"truncated payload", valid[:len(valid)-1024-510], "truncated"},
		{"missing end blocks", valid[:len(valid)-1024], "truncated archive"},
		{"one end block", valid[:len(valid)-512], "two zero"},
		{"trailing bytes", append(append([]byte{}, valid...), 1), "trailing archive data"},
		{"unsafe header", mutateEntry(func(e []testEntry) { e[2].name = "../payload" }), "unsafe member path"},
		{"missing payload", rawArchive(t, archiveEntries(t, valid)[:2]...), "missing archive member"},
		{"duplicate payload", rawArchive(t, append(archiveEntries(t, valid), archiveEntries(t, valid)[2])...), "duplicate archive member"},
		{"unchanged payload", rawArchive(t, append(archiveEntries(t, valid), testEntry{"a", []byte("unchanged")})...), "unlisted archive member"},
		{"unsupported schema", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.SchemaVersion = 2 }), "unsupported delta schema"},
		{"unsafe metadata path", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Added = []string{"../d"} }), "unsafe member path"},
		{"duplicate metadata path", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Added = []string{"b"} }), "duplicate"},
		{"unsorted paths", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Removed = []string{"z", "c"} }), "sorted"},
		{"payload absent in target", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Added = []string{"z"} }), "absent from target"},
		{"removed present in target", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Removed = []string{"a"} }), "remains in target"},
		{"missing removal", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.Removed = []string{} }), "inventory does not match"},
		{"wrong change kind", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) {
			d.Added = []string{"b", "d"}
			d.Changed = []string{}
		}), "inventory does not match"},
		{"wrong base", replaceDeltaMetadata(t, valid, func(d *DeltaMetadata) { d.BaseSHA256 = strings.Repeat("0", 64) }), "wrong base SHA256"},
		{"metadata bound", mutateEntry(func(e []testEntry) { e[0].data = bytes.Repeat([]byte(" "), MaxManifestSize+1) }), "at most"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := filepath.Join(t.TempDir(), "bad.tar")
			writeInput(t, name, tt.data)
			wantApplyFailure(t, base, name, tt.want)
		})
	}
}

func TestApplyRejectsWrongBaseWithSameContentRoot(t *testing.T) {
	base, _, delta := updatePair(t)
	entries := archiveEntries(t, readTestFile(t, base))
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Size: int64(len(e.data)), Mode: 0600, Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other.tar")
	writeInput(t, other, buf.Bytes())
	m, err := VerifyFile(other)
	if err != nil {
		t.Fatal(err)
	}
	original, err := VerifyFile(base)
	if err != nil || original.ContentRoot != m.ContentRoot {
		t.Fatalf("content roots changed: %+v, %+v, %v", original, m, err)
	}
	wantApplyFailure(t, other, delta, "wrong base SHA256")
}

func TestDiffAndApplyRejectInvalidFullPacks(t *testing.T) {
	base, target, delta := updatePair(t)
	valid := readTestFile(t, base)
	corrupt := archiveEntries(t, valid)
	corrupt[1].data[0] ^= 1
	entry := testEntry{"../unsafe", []byte("payload")}
	oversized := testManifest(testEntry{"a", []byte("payload")})
	oversized.Members[0].Size = MaxMemberSize + 1
	oversized.ContentRoot = contentRoot(oversized.Members)
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"corrupted", rawArchive(t, corrupt...), "SHA256 mismatch"},
		{"truncated", valid[:len(valid)-512], "two zero"},
		{"unsafe path", testArchive(t, manifestJSON(t, testManifest(entry)), entry), "unsafe member path"},
		{"oversized member", testArchive(t, manifestJSON(t, oversized)), "size bound"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := filepath.Join(t.TempDir(), "bad.tar")
			writeInput(t, bad, tt.data)
			wantApplyFailure(t, bad, delta, tt.want)
			for _, inputs := range [][2]string{{bad, target}, {base, bad}} {
				dir := t.TempDir()
				if _, err := Diff(inputs[0], inputs[1], filepath.Join(dir, "delta")); err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("Diff error = %v, want %q", err, tt.want)
				}
				files, err := os.ReadDir(dir)
				if err != nil || len(files) != 0 {
					t.Fatalf("failed diff left files: %v, %v", files, err)
				}
			}
		})
	}
}

func TestUpdateArchiveBoundsAndSnapshotCleanup(t *testing.T) {
	base, target, delta := updatePair(t)
	temps := t.TempDir()
	t.Setenv("TMPDIR", temps)
	huge := filepath.Join(t.TempDir(), "huge.tar")
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxArchiveSize + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	wantApplyFailure(t, base, huge, "at most")
	wantApplyFailure(t, huge, delta, "at most")
	for _, input := range []string{huge, filepath.Join(t.TempDir(), "missing"), t.TempDir()} {
		if _, err := Diff(base, input, filepath.Join(t.TempDir(), "delta")); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	outputDir := t.TempDir()
	goodDelta := filepath.Join(outputDir, "delta.tar")
	if _, err := Diff(base, target, goodDelta); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(base, goodDelta, filepath.Join(outputDir, "result.tar")); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(temps)
	if err != nil || len(files) != 0 {
		t.Fatalf("snapshot files leaked: %v, %v", files, err)
	}
}

func TestUpdateNeverOverwritesDestinations(t *testing.T) {
	base, target, delta := updatePair(t)
	for _, kind := range []string{"base", "delta", "existing", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "output")
			switch kind {
			case "base":
				dest = base
			case "delta":
				dest = delta
			case "existing":
				writeInput(t, dest, []byte("keep"))
			case "symlink":
				if err := os.Symlink(base, dest); err != nil {
					t.Fatal(err)
				}
			}
			original := readTestFile(t, dest)
			if _, err := Diff(base, target, dest); err == nil {
				t.Fatal("diff overwrote destination")
			}
			if _, err := Apply(base, delta, dest); err == nil {
				t.Fatal("apply overwrote destination")
			}
			if !bytes.Equal(readTestFile(t, dest), original) {
				t.Fatal("destination modified")
			}
		})
	}
}

func TestPublishVerifiesBeforeExclusivePublication(t *testing.T) {
	for _, stage := range []string{"write", "verify", "concurrent destination"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "result")
			err := publishArchive(dest, func(w io.Writer) error {
				if stage == "write" {
					return io.ErrUnexpectedEOF
				}
				_, err := w.Write([]byte("bytes"))
				if stage == "concurrent destination" {
					writeInput(t, dest, []byte("keep"))
				}
				return err
			}, func(r io.Reader) error {
				if stage == "verify" {
					return io.ErrUnexpectedEOF
				}
				_, err := io.Copy(io.Discard, r)
				return err
			})
			if err == nil {
				t.Fatal("publication unexpectedly succeeded")
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if stage == "concurrent destination" {
				if len(files) != 1 || string(readTestFile(t, dest)) != "keep" {
					t.Fatal("concurrent destination overwritten or temporary file leaked")
				}
			} else if len(files) != 0 {
				t.Fatal("failed write/verify left output")
			}
		})
	}
}

func TestDeltaRejectsOversizedTarget(t *testing.T) {
	base, _, delta := updatePair(t)
	entries := archiveEntries(t, readTestFile(t, delta))
	target, err := parseManifest(entries[1].data)
	if err != nil {
		t.Fatal(err)
	}
	target.Members[0].Size = MaxMemberSize + 1
	target.ContentRoot = contentRoot(target.Members)
	entries[1].data = manifestJSON(t, target)
	data := replaceDeltaMetadata(t, rawArchive(t, entries...), func(d *DeltaMetadata) {
		d.TargetManifestSHA256 = fmt.Sprintf("%x", sha256.Sum256(entries[1].data))
	})
	bad := filepath.Join(t.TempDir(), "bad")
	writeInput(t, bad, data)
	wantApplyFailure(t, base, bad, "size bound")
}

func FuzzVerifyDelta(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte("x"), 512))
	m := testManifest(testEntry{"future.bin", []byte{}})
	target := manifestJSON(f, m)
	d := DeltaMetadata{
		SchemaVersion: 1, BaseSHA256: strings.Repeat("0", 64),
		TargetManifestSHA256: fmt.Sprintf("%x", sha256.Sum256(target)),
		Added:                []string{}, Changed: []string{}, Removed: []string{},
	}
	data, err := json.Marshal(d)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(rawArchive(f, testEntry{DeltaName, data}, testEntry{ManifestName, target}))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxManifestSize {
			t.Skip()
		}
		_, _, _, _, _ = verifyDelta(bytes.NewReader(data))
	})
}
