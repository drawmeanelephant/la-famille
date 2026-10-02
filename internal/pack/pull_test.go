package pack

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func archiveHash(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func alternateArchiveMode(data []byte) []byte {
	data = bytes.Clone(data)
	copy(data[100:107], "0000600")
	copy(data[148:156], "        ")
	sum := 0
	for _, b := range data[:512] {
		sum += int(b)
	}
	copy(data[148:156], fmt.Sprintf("%06o\x00 ", sum))
	return data
}

func writeTestFeed(t *testing.T, directory string, feed Feed) {
	t.Helper()
	data, err := json.Marshal(feed)
	if err != nil {
		t.Fatal(err)
	}
	writeInput(t, filepath.Join(directory, FeedName), data)
}

func pullFixture(t *testing.T) (string, string, []byte, []byte, Feed) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(t.TempDir(), "base.tar")
	unknown := testEntry{"future/opaque.bin", []byte{0, 255, 1}}
	before := writeTestPack(t, base, unknown,
		testEntry{"rag-content.md", []byte("old fact")}, testEntry{"removed", []byte("gone")})
	target := writeTestPack(t, filepath.Join(dir, "current.tar"), unknown,
		testEntry{"new", []byte("added")}, testEntry{"rag-content.md", []byte("new fact")})
	if _, err := Diff(base, filepath.Join(dir, "current.tar"), filepath.Join(dir, "delta.tar")); err != nil {
		t.Fatal(err)
	}
	feed := Feed{
		SchemaVersion: FeedVersion,
		Full:          FeedPack{Path: "current.tar", SHA256: archiveHash(target)},
		Deltas:        []FeedDelta{{Path: "delta.tar", BaseSHA256: archiveHash(before)}},
	}
	writeTestFeed(t, dir, feed)
	return dir, base, before, target, feed
}

func assertPullFailure(t *testing.T, dir, base, output, expected string, before []byte) {
	t.Helper()
	_, err := Pull(dir, base, output)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q failure, got %v", expected, err)
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("failed pull left a completed output: %v", err)
	}
	if base != "" && !bytes.Equal(readTestFile(t, base), before) {
		t.Fatal("failed pull changed the base")
	}
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(output), ".la-famille-pack-output-*"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("failed pull left staging files: %v, %v", temps, err)
	}
}

func TestPullColdAndDelta(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	cold := filepath.Join(t.TempDir(), "cold.tar")
	result, err := Pull(dir, "", cold)
	if err != nil || result.Mode != "full" || result.Fallback || result.Changes != nil ||
		result.TargetSHA256 != feed.Full.SHA256 {
		t.Fatalf("cold = %+v, %v", result, err)
	}
	if !bytes.Equal(readTestFile(t, cold), target) {
		t.Fatal("cold pull was not byte-identical")
	}
	if _, err := VerifyFile(cold); err != nil {
		t.Fatal(err)
	}
	// Delta mode must not open or read the target full archive.
	if err := os.Remove(filepath.Join(dir, feed.Full.Path)); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "updated.tar")
	result, err = Pull(dir, base, output)
	if err != nil || result.Mode != "delta" || result.Fallback ||
		result.TargetSHA256 != archiveHash(target) || result.Changes == nil {
		t.Fatalf("update = %+v, %v", result, err)
	}
	if !reflect.DeepEqual(result.Changes.Added, []string{"new"}) ||
		!reflect.DeepEqual(result.Changes.Removed, []string{"removed"}) ||
		!reflect.DeepEqual(result.Changes.Changed, []string{"rag-content.md"}) {
		t.Fatalf("changes = %+v", result.Changes)
	}
	if !bytes.Equal(readTestFile(t, output), target) {
		t.Fatal("delta pull was not byte-identical, including the unknown member")
	}
	if !bytes.Equal(readTestFile(t, base), before) {
		t.Fatal("successful update changed the base")
	}
	if _, err := VerifyFile(output); err != nil {
		t.Fatal(err)
	}
}

func TestPullExplicitFallbackAndExactBaseIdentity(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	// A different legal tar mode gives the same payload but a different identity.
	before = alternateArchiveMode(before)
	writeInput(t, base, before)
	if _, err := VerifyFile(base); err != nil {
		t.Fatal(err)
	}
	if archiveHash(before) == feed.Deltas[0].BaseSHA256 {
		t.Fatal("fixture did not change exact archive identity")
	}
	// An unselected corrupt delta is never read.
	writeInput(t, filepath.Join(dir, feed.Deltas[0].Path), []byte("not a delta"))
	output := filepath.Join(t.TempDir(), "fallback.tar")
	result, err := Pull(dir, base, output)
	if err != nil || result.Mode != "full" || !result.Fallback || result.Changes == nil {
		t.Fatalf("fallback = %+v, %v", result, err)
	}
	if !bytes.Equal(readTestFile(t, output), target) || !bytes.Equal(readTestFile(t, base), before) {
		t.Fatal("fallback was not exact or changed the base")
	}
}

func TestPullColdRetainsAlternateLayoutButDeltaRequiresAdvertisedIdentity(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	target = alternateArchiveMode(target)
	if _, err := Verify(bytes.NewReader(target)); err != nil {
		t.Fatal(err)
	}
	writeInput(t, filepath.Join(dir, feed.Full.Path), target)
	feed.Full.SHA256 = archiveHash(target)
	writeTestFeed(t, dir, feed)
	output := filepath.Join(t.TempDir(), "cold.tar")
	if _, err := Pull(dir, "", output); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readTestFile(t, output), target) {
		t.Fatal("cold copy canonicalized a verified alternate layout")
	}
	assertPullFailure(t, dir, base, filepath.Join(t.TempDir(), "updated.tar"),
		"target archive SHA256 mismatch", before)
}

func TestPullDeltaDoesNotOverwrite(t *testing.T) {
	dir, base, before, _, _ := pullFixture(t)
	for _, name := range []string{"base", "file", "directory", "hard link", "symlink"} {
		t.Run(name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output")
			switch name {
			case "base":
				output = base
			case "file":
				writeInput(t, output, []byte("sentinel"))
			case "directory":
				if err := os.Mkdir(output, 0700); err != nil {
					t.Fatal(err)
				}
			case "hard link":
				if err := os.Link(base, output); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(base, output); err != nil {
					t.Fatal(err)
				}
			}
			info, err := os.Lstat(output)
			if err != nil {
				t.Fatal(err)
			}
			var old []byte
			if !info.IsDir() {
				old = readTestFile(t, output)
			}
			if _, err := Pull(dir, base, output); err == nil {
				t.Fatal("delta pull overwrote an existing destination")
			}
			after, err := os.Lstat(output)
			if err != nil || !os.SameFile(info, after) || !bytes.Equal(readTestFile(t, base), before) {
				t.Fatal("failed delta pull changed the destination or base")
			}
			if !info.IsDir() && !bytes.Equal(readTestFile(t, output), old) {
				t.Fatal("failed delta pull changed destination contents")
			}
			temps, _ := filepath.Glob(filepath.Join(filepath.Dir(output), ".la-famille-pack-output-*"))
			if len(temps) != 0 {
				t.Fatal("failed delta pull left staging files")
			}
		})
	}
}

func TestPullSelectedDeltaFailuresNeverFallback(t *testing.T) {
	for _, name := range []string{"corrupt", "truncated", "wrong base", "inventory", "target digest", "missing"} {
		t.Run(name, func(t *testing.T) {
			dir, base, before, _, feed := pullFixture(t)
			delta := filepath.Join(dir, feed.Deltas[0].Path)
			expected := "selected delta"
			switch name {
			case "corrupt":
				writeInput(t, delta, []byte("corrupt"))
			case "truncated":
				data := readTestFile(t, delta)
				writeInput(t, delta, data[:len(data)-1])
			case "wrong base", "inventory":
				entries := archiveEntries(t, readTestFile(t, delta))
				var metadata DeltaMetadata
				if err := json.Unmarshal(entries[0].data, &metadata); err != nil {
					t.Fatal(err)
				}
				if name == "wrong base" {
					metadata.BaseSHA256 = strings.Repeat("0", 64)
				} else {
					metadata.Removed = []string{}
				}
				entries[0].data, _ = json.Marshal(metadata)
				writeInput(t, delta, rawArchive(t, entries...))
			case "target digest":
				feed.Full.SHA256 = strings.Repeat("0", 64)
				writeTestFeed(t, dir, feed)
				expected = "target archive SHA256 mismatch"
			case "missing":
				if err := os.Remove(delta); err != nil {
					t.Fatal(err)
				}
			}
			// Keep a valid full target available: a silent fallback would succeed.
			assertPullFailure(t, dir, base, filepath.Join(t.TempDir(), "output.tar"), expected, before)
		})
	}
}

func TestPullFullFailuresAndExistingDestinations(t *testing.T) {
	for _, name := range []string{"corrupt full", "target digest", "corrupt base", "existing", "base", "output symlink"} {
		t.Run(name, func(t *testing.T) {
			dir, base, before, target, feed := pullFixture(t)
			feed.Deltas = []FeedDelta{}
			output := filepath.Join(t.TempDir(), "output.tar")
			expected := ""
			switch name {
			case "corrupt full":
				writeInput(t, filepath.Join(dir, feed.Full.Path), []byte("corrupt"))
				expected = "full pack"
			case "target digest":
				feed.Full.SHA256 = strings.Repeat("0", 64)
				expected = "target archive SHA256 mismatch"
			case "corrupt base":
				before = []byte("corrupt base")
				writeInput(t, base, before)
				expected = "base pack"
			case "existing":
				writeInput(t, output, []byte("existing sentinel"))
			case "base":
				output = base
			case "output symlink":
				if err := os.Symlink(base, output); err != nil {
					t.Fatal(err)
				}
			}
			writeTestFeed(t, dir, feed)
			if name == "existing" || name == "base" || name == "output symlink" {
				info, err := os.Lstat(output)
				if err != nil {
					t.Fatal(err)
				}
				old := readTestFile(t, output)
				if _, err := Pull(dir, base, output); err == nil {
					t.Fatal("overwrote an existing destination")
				}
				after, err := os.Lstat(output)
				if err != nil || !os.SameFile(info, after) || !bytes.Equal(old, readTestFile(t, output)) {
					t.Fatal("failed pull changed destination")
				}
				if !bytes.Equal(readTestFile(t, base), before) {
					t.Fatal("failed pull changed base")
				}
			} else {
				assertPullFailure(t, dir, base, output, expected, before)
			}
			if name != "corrupt full" && !bytes.Equal(readTestFile(t, filepath.Join(dir, feed.Full.Path)), target) {
				t.Fatal("pull changed the source")
			}
		})
	}
}
