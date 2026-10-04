package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/pack"
)

// The subscriber invokes only the compiled binary. Build/export happen solely
// at the publisher; the source site and full target are gone before updating.
func TestPackPullCompiledSubscriber(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	binary := filepath.Join(work, "la-famille")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/la-famille")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	publisher, subscriber, feedDir := filepath.Join(work, "publisher"), filepath.Join(work, "subscriber"), filepath.Join(work, "feed")
	if err := os.CopyFS(publisher, os.DirFS(filepath.Join(repo, "assets/testdata/pack-subscriber"))); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{subscriber, feedDir} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	run := func(directory string, args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = directory
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, output)
		}
		t.Logf("%v\n%s", args, output)
		return string(output)
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	buildPack := func(name string) {
		t.Helper()
		run(publisher, "build")
		run(publisher, "rag")
		run(publisher, "pack", "build", "--output", filepath.Join(feedDir, name))
	}
	buildPack("before.tar")
	baseBytes := read(filepath.Join(feedDir, "before.tar"))
	feed := pack.Feed{
		SchemaVersion: pack.FeedVersion,
		Full:          pack.FeedPack{Path: "before.tar", SHA256: fmt.Sprintf("%x", sha256.Sum256(baseBytes))},
		Deltas:        []pack.FeedDelta{},
	}
	writeFeed := func() {
		t.Helper()
		data, err := json.Marshal(feed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(feedDir, pack.FeedName), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeFeed()
	cold := run(subscriber, "pack", "pull", feedDir, "--output", "base.tar")
	if !strings.Contains(cold, "mode full") || !bytes.Equal(read(filepath.Join(subscriber, "base.tar")), baseBytes) {
		t.Fatal("subscriber cold pull was not a full byte-identical copy")
	}
	note := filepath.Join(publisher, "notes/research/birds.md")
	updatedNote := bytes.ReplaceAll(read(note), []byte("seven days"), []byte("three days"))
	if err := os.WriteFile(note, updatedNote, 0600); err != nil {
		t.Fatal(err)
	}
	buildPack("after.tar")
	targetBytes := read(filepath.Join(feedDir, "after.tar"))
	run(work, "pack", "diff", filepath.Join(feedDir, "before.tar"), filepath.Join(feedDir, "after.tar"),
		"--output", filepath.Join(feedDir, "delta.tar"))
	feed.Full = pack.FeedPack{Path: "after.tar", SHA256: fmt.Sprintf("%x", sha256.Sum256(targetBytes))}
	feed.Deltas = []pack.FeedDelta{{Path: "delta.tar", BaseSHA256: fmt.Sprintf("%x", sha256.Sum256(baseBytes))}}
	writeFeed()
	if err := os.RemoveAll(publisher); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(feedDir, "after.tar")); err != nil {
		t.Fatal(err)
	}
	update := run(subscriber, "pack", "pull", feedDir, "--base", "base.tar", "--output", "updated.tar")
	if !strings.Contains(update, "mode delta") || !strings.Contains(update, "~ rag-content.md") ||
		!strings.Contains(update, feed.Full.SHA256) {
		t.Fatal("subscriber update did not identify delta mode, member changes, and target identity")
	}
	applied := filepath.Join(subscriber, "updated.tar")
	if !bytes.Equal(read(applied), targetBytes) || !bytes.Equal(read(filepath.Join(subscriber, "base.tar")), baseBytes) {
		t.Fatal("update differed from target or changed subscriber base")
	}
	run(subscriber, "pack", "verify", "base.tar")
	run(subscriber, "pack", "verify", "updated.tar")
	assertSubscriberFiles(t, subscriber)

	// External tools can read the same verified archive and metadata without a
	// model runtime or a source checkout. This replaces assistant-based checks.
	reader := tar.NewReader(bytes.NewReader(read(applied)))
	members := make(map[string][]byte)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		members[header.Name], err = io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
	}
	content := string(members["rag-content.md"])
	if !strings.Contains(content, `<file path="notes/research/birds.md">`) ||
		!strings.Contains(content, "three days") || strings.Contains(content, "seven days") ||
		!strings.Contains(content, `<file path="notes/research/maps.md">`) {
		t.Fatalf("subscriber archive has stale content or incompatible framing: %s", content)
	}
	for _, name := range []string{"meta.json", "search.json", "graph.json", "backlinks.json", "site-manifest.json"} {
		if !json.Valid(members[name]) {
			t.Fatalf("missing or invalid packaged metadata %s: %s", name, members[name])
		}
	}
	var metadata map[string]struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}
	if err := json.Unmarshal(members["meta.json"], &metadata); err != nil {
		t.Fatal(err)
	}
	birds := metadata["birds"]
	if len(metadata) != 2 || birds.Title != "Bird Observation Notes" || birds.URL != "/field-guide/bird-observations/" {
		t.Fatalf("packaged page identity changed: %+v", metadata)
	}
	assertSubscriberFiles(t, subscriber)
}

func assertSubscriberFiles(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "base.tar" || entries[1].Name() != "updated.tar" {
		t.Fatalf("subscriber has files other than its two packs: %v", entries)
	}
}
