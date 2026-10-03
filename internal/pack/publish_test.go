package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPublishFirstRepeatEditAndRetention(t *testing.T) {
	build := buildInputs(t)
	writeInput(t, filepath.Join(build.RagDir, "rag-system.md"), []byte("excluded secret"))
	writeInput(t, filepath.Join(build.OutputDir, ".env"), []byte("excluded secret"))
	dir := t.TempDir()
	first := filepath.Join(dir, "first")
	feed, err := Publish(PublishOptions{Build: build}, first)
	if err != nil || len(feed.Deltas) != 0 {
		t.Fatalf("first: %+v %v", feed, err)
	}
	currentBytes := readTestFile(t, filepath.Join(first, feed.Full.Path))
	if feed.Full.Path != versionPath(archiveHash(currentBytes)) {
		t.Fatal("artifact is not content addressed")
	}
	m, err := VerifyFile(filepath.Join(first, feed.Full.Path))
	if err != nil || len(m.Members) != 6 {
		t.Fatalf("allowlist: %+v %v", m, err)
	}
	repeat := filepath.Join(dir, "repeat")
	repeated, err := Publish(PublishOptions{Build: build, Previous: first}, repeat)
	if err != nil || !reflect.DeepEqual(feed, repeated) ||
		!bytes.Equal(readTestFile(t, filepath.Join(first, FeedName)), readTestFile(t, filepath.Join(repeat, FeedName))) {
		t.Fatalf("repeat not deterministic: %+v %v", repeated, err)
	}
	previous, old := repeat, repeated
	for i := range 4 {
		writeInput(t, filepath.Join(build.RagDir, "rag-content.md"), []byte{byte(i)})
		next := filepath.Join(t.TempDir(), "feed")
		result, err := Publish(PublishOptions{Build: build, Previous: previous, Retain: 3}, next)
		if err != nil || len(result.Deltas) != min(i+1, 2) || result.Deltas[0].BaseSHA256 != old.Full.SHA256 {
			t.Fatalf("edit %d: %+v %v", i, result, err)
		}
		output := filepath.Join(t.TempDir(), "pulled.tar")
		pulled, err := Pull(next, filepath.Join(previous, old.Full.Path), output)
		if err != nil || pulled.Mode != "delta" ||
			!bytes.Equal(readTestFile(t, output), readTestFile(t, filepath.Join(next, result.Full.Path))) {
			t.Fatalf("published delta: %+v %v", pulled, err)
		}
		packs, _ := os.ReadDir(filepath.Join(next, "packs"))
		deltas, _ := os.ReadDir(filepath.Join(next, "deltas"))
		if len(packs) != len(result.Deltas)+1 || len(deltas) != len(result.Deltas) {
			t.Fatal("retention is not bounded")
		}
		// Previous deployment remains byte-identical.
		if !bytes.Equal(readTestFile(t, filepath.Join(previous, old.Full.Path)),
			readTestFile(t, filepath.Join(next, versionPath(old.Full.SHA256)))) {
			t.Fatal("retained version changed")
		}
		previous, old = next, result
	}
}

func TestPublishedDeltaRoundtripChecksExactTarget(t *testing.T) {
	dir, base, _, target, _ := pullFixture(t)
	valid := filepath.Join(t.TempDir(), "valid.tar")
	if err := checkPublishedDelta(base, filepath.Join(dir, "delta.tar"), valid, archiveHash(target)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(target, readTestFile(t, valid)) {
		t.Fatal("roundtrip changed target bytes")
	}
	invalid := filepath.Join(t.TempDir(), "invalid.tar")
	if err := checkPublishedDelta(base, filepath.Join(dir, "delta.tar"), invalid, strings.Repeat("0", 64)); err == nil {
		t.Fatal("roundtrip accepted wrong exact target hash")
	}
	if _, err := os.Stat(invalid); !os.IsNotExist(err) {
		t.Fatal("roundtrip published target on hash mismatch")
	}
}

func TestPublishMissingHistoryAndFailures(t *testing.T) {
	for _, name := range []string{"no directory", "no manifest", "no prior pack", "corrupt pack", "corrupt manifest", "existing output", "build error", "retention"} {
		t.Run(name, func(t *testing.T) {
			build := buildInputs(t)
			previous := filepath.Join(t.TempDir(), "history")
			old, err := Publish(PublishOptions{Build: build}, previous)
			if err != nil {
				t.Fatal(err)
			}
			writeInput(t, filepath.Join(build.RagDir, "rag-content.md"), []byte("edited fact"))
			destination := filepath.Join(t.TempDir(), "new")
			options := PublishOptions{Build: build, Previous: previous}
			fail := false
			switch name {
			case "no directory":
				options.Previous = filepath.Join(t.TempDir(), "missing")
			case "no manifest":
				_ = os.Remove(filepath.Join(previous, FeedName))
			case "no prior pack":
				_ = os.Remove(filepath.Join(previous, old.Full.Path))
			case "corrupt pack":
				writeInput(t, filepath.Join(previous, old.Full.Path), []byte("bad"))
				fail = true
			case "corrupt manifest":
				writeInput(t, filepath.Join(previous, FeedName), []byte("{}"))
				fail = true
			case "existing output":
				writeInput(t, filepath.Join(destination, "sentinel"), []byte("user file"))
				fail = true
			case "build error":
				_ = os.Remove(filepath.Join(build.RagDir, "rag-content.md"))
				fail = true
			case "retention":
				options.Retain = 9
				fail = true
			}
			feed, err := Publish(options, destination)
			if fail {
				if err == nil {
					t.Fatal("invalid publication succeeded")
				}
				if name == "existing output" {
					if string(readTestFile(t, filepath.Join(destination, "sentinel"))) != "user file" {
						t.Fatal("overwrote existing output")
					}
				} else if _, err := os.Lstat(destination); !os.IsNotExist(err) {
					t.Fatal("failed publication left output")
				}
			} else if err != nil || len(feed.Deltas) != 0 {
				t.Fatalf("missing baseline must publish full only: %+v %v", feed, err)
			}
		})
	}
}
