package pack

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeedStrictSchemaAndLimits(t *testing.T) {
	valid := `{"schema_version":1,"full":{"path":"packs/current.tar","sha256":"` + strings.Repeat("a", 64) +
		`"},"deltas":[{"path":"deltas/update.tar","base_sha256":"` + strings.Repeat("b", 64) + `"}]}`
	if _, err := parseFeed([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"empty":            "",
		"syntax":           "{",
		"trailing":         valid + "{}",
		"scalar":           "1",
		"array":            "[]",
		"invalid UTF8":     valid + string([]byte{255}),
		"version":          strings.Replace(valid, `"schema_version":1`, `"schema_version":2`, 1),
		"missing version":  strings.Replace(valid, `"schema_version":1,`, "", 1),
		"duplicate key":    strings.Replace(valid, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1),
		"case alias":       strings.Replace(valid, `"path":"packs`, `"Path":"packs`, 1),
		"unknown":          strings.Replace(valid, `"path":"packs`, `"extra":true,"path":"packs`, 1),
		"missing full":     `{"schema_version":1,"deltas":[]}`,
		"missing hash":     strings.Replace(valid, `,"sha256":"`+strings.Repeat("a", 64)+`"`, "", 1),
		"bad hash":         strings.Replace(valid, strings.Repeat("a", 64), "bad", 1),
		"uppercase hash":   strings.Replace(valid, strings.Repeat("a", 64), strings.Repeat("A", 64), 1),
		"missing path":     strings.Replace(valid, `"path":"packs/current.tar",`, "", 1),
		"missing deltas":   `{"schema_version":1,"full":{"path":"current.tar","sha256":"` + strings.Repeat("a", 64) + `"}}`,
		"missing base":     strings.Replace(valid, `,"base_sha256":"`+strings.Repeat("b", 64)+`"`, "", 1),
		"bad base":         strings.Replace(valid, strings.Repeat("b", 64), "bad", 1),
		"null field":       strings.Replace(valid, `"path":"packs/current.tar"`, `"path":null`, 1),
		"null array":       `{"schema_version":1,"full":{"path":"current.tar","sha256":"` + strings.Repeat("a", 64) + `"},"deltas":null}`,
		"null entry":       `{"schema_version":1,"full":{"path":"current.tar","sha256":"` + strings.Repeat("a", 64) + `"},"deltas":[null]}`,
		"wrong type":       strings.Replace(valid, `"schema_version":1`, `"schema_version":"1"`, 1),
		"duplicate nested": strings.Replace(valid, `"path":"packs/current.tar"`, `"path":"other.tar","path":"packs/current.tar"`, 1),
		"oversize":         valid + strings.Repeat(" ", MaxFeedSize),
		"deep nesting":     strings.Repeat("[", 33) + "1" + strings.Repeat("]", 33),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseFeed([]byte(data)); err == nil {
				t.Fatal("malformed feed accepted")
			}
		})
	}
	var feed Feed
	if err := json.Unmarshal([]byte(valid), &feed); err != nil {
		t.Fatal(err)
	}
	feed.Deltas = []FeedDelta{}
	data, _ := json.Marshal(feed)
	// Exactly the byte limit is accepted; the next byte is not.
	data = append(data, bytes.Repeat([]byte{' '}, MaxFeedSize-len(data))...)
	if _, err := parseFeed(data); err != nil {
		t.Fatal(err)
	}
	if _, err := parseFeed(append(data, ' ')); err == nil {
		t.Fatal("accepted oversized feed")
	}
	for i := range MaxFeedDeltas + 1 {
		feed.Deltas = append(feed.Deltas, FeedDelta{
			Path:       "delta-" + archiveHash([]byte{byte(i)}) + ".tar",
			BaseSHA256: archiveHash([]byte{byte(i)}),
		})
		if i == MaxFeedDeltas-1 {
			data, _ := json.Marshal(feed)
			if _, err := parseFeed(data); err != nil {
				t.Fatalf("count boundary failed: %v", err)
			}
		}
	}
	data, _ = json.Marshal(feed)
	if _, err := parseFeed(data); err == nil {
		t.Fatal("accepted too many deltas")
	}
}

func TestPullRejectsUnsafeAndDuplicateDeclaredPaths(t *testing.T) {
	paths := []string{"", ".", "..", "../outside.tar", "/absolute.tar", "a/../b", "a/./b", "a//b",
		"a/", `a\b`, "C:/a", "https://example.test/a", "a\x00b", "a\nb", strings.Repeat("a", MaxPathBytes+1), FeedName}
	for _, name := range paths {
		for _, field := range []string{"full", "delta"} {
			t.Run(field+"/"+name, func(t *testing.T) {
				dir, base, before, _, feed := pullFixture(t)
				if field == "full" {
					feed.Full.Path = name
				} else {
					feed.Deltas[0].Path = name
				}
				writeTestFeed(t, dir, feed)
				assertPullFailure(t, dir, base, filepath.Join(t.TempDir(), "output.tar"), "", before)
			})
		}
	}
	for _, name := range []string{"full delta collision", "repeated delta path", "ambiguous base"} {
		t.Run(name, func(t *testing.T) {
			dir, base, before, _, feed := pullFixture(t)
			switch name {
			case "full delta collision":
				feed.Deltas[0].Path = feed.Full.Path
			case "repeated delta path":
				feed.Deltas = append(feed.Deltas, FeedDelta{Path: feed.Deltas[0].Path, BaseSHA256: strings.Repeat("a", 64)})
			case "ambiguous base":
				feed.Deltas = append(feed.Deltas, FeedDelta{Path: "another.tar", BaseSHA256: feed.Deltas[0].BaseSHA256})
			}
			writeTestFeed(t, dir, feed)
			assertPullFailure(t, dir, base, filepath.Join(t.TempDir(), "output.tar"), "duplicate", before)
		})
	}
}

func TestPullRejectsSymlinkMediatedSources(t *testing.T) {
	for _, name := range []string{"manifest", "full", "selected delta", "unselected delta", "parent", "escaping parent", "feed directory", "feed directory trailing slash"} {
		t.Run(name, func(t *testing.T) {
			dir, base, before, _, feed := pullFixture(t)
			selectedBase := base
			switch name {
			case "manifest", "full", "selected delta", "unselected delta":
				file := FeedName
				switch name {
				case "full":
					file = feed.Full.Path
				case "selected delta", "unselected delta":
					file = feed.Deltas[0].Path
				}
				original := filepath.Join(dir, file)
				target := filepath.Join(dir, "real-"+file)
				if err := os.Rename(original, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, original); err != nil {
					t.Fatal(err)
				}
				if name == "unselected delta" {
					selectedBase = "" // Cold mode still rejects a declared symlink.
				}
			case "parent", "escaping parent":
				target := filepath.Join(dir, "real")
				if name == "escaping parent" {
					target = t.TempDir()
				} else if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(dir, "linked")); err != nil {
					t.Fatal(err)
				}
				// Missing leaf still cannot hide a symlink parent.
				feed.Full.Path = "linked/missing.tar"
				writeTestFeed(t, dir, feed)
			case "feed directory", "feed directory trailing slash":
				link := filepath.Join(t.TempDir(), "feed")
				if err := os.Symlink(dir, link); err != nil {
					t.Fatal(err)
				}
				dir = link
				if name == "feed directory trailing slash" {
					dir += string(filepath.Separator)
				}
			}
			assertPullFailure(t, dir, selectedBase, filepath.Join(t.TempDir(), "output.tar"), "symlink", before)
			if !bytes.Equal(readTestFile(t, base), before) {
				t.Fatal("source validation changed the base")
			}
		})
	}
}

func TestPullResourceLimitsAndMalformedFeedCleanup(t *testing.T) {
	for _, name := range []string{"feed size", "full size", "delta size", "malformed", "missing feed", "non-regular"} {
		t.Run(name, func(t *testing.T) {
			dir, base, before, _, feed := pullFixture(t)
			switch name {
			case "feed size", "full size", "delta size":
				file, size := FeedName, int64(MaxFeedSize+1)
				if name == "full size" {
					file, size = feed.Full.Path, MaxArchiveSize+1
					feed.Deltas = []FeedDelta{}
					writeTestFeed(t, dir, feed)
				}
				if name == "delta size" {
					file, size = feed.Deltas[0].Path, MaxArchiveSize+1
				}
				if err := os.Truncate(filepath.Join(dir, file), size); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				writeInput(t, filepath.Join(dir, FeedName), []byte(`{"schema_version":1,"schema_version":1}`))
			case "missing feed":
				if err := os.Remove(filepath.Join(dir, FeedName)); err != nil {
					t.Fatal(err)
				}
			case "non-regular":
				feed.Full.Path = "directory"
				if err := os.Mkdir(filepath.Join(dir, feed.Full.Path), 0700); err != nil {
					t.Fatal(err)
				}
				writeTestFeed(t, dir, feed)
			}
			assertPullFailure(t, dir, base, filepath.Join(t.TempDir(), "output.tar"), "", before)
		})
	}
}

func TestFeedSnapshotRetainsOpenedVerifiedBytes(t *testing.T) {
	dir, base, before, target, feed := pullFixture(t)
	root, err := openFeedRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	input, err := openFeedSource(root, feed.Deltas[0].Path, MaxArchiveSize)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	delta, metadata, err := loadOpenDelta(input)
	if err != nil {
		t.Fatal(err)
	}
	defer delta.close()
	baseSnapshot, err := loadPack(base)
	if err != nil {
		t.Fatal(err)
	}
	defer baseSnapshot.close()
	// Destroy every original archive path after capture. Reconstruction must
	// still consume the verified snapshot bytes, not reopened originals.
	writeInput(t, filepath.Join(dir, feed.Deltas[0].Path), []byte("replacement"))
	writeInput(t, base, []byte("replacement"))
	if err := os.Remove(filepath.Join(dir, feed.Full.Path)); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "output.tar")
	if _, err := applySnapshots(baseSnapshot, delta, metadata, output, feed.Full.SHA256); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(readTestFile(t, output), target) || baseSnapshot.hash != archiveHash(before) {
		t.Fatal("consumed unverified replacement bytes")
	}
}
