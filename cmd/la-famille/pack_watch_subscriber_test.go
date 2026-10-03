package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/pack"
)

// This is compiled-binary local workflow coverage, not hosted HTTPS or a real
// model demonstration. Network policy remains enabled in the compiled binary.
func TestPackWatchCompiledSubscriber(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("compiled watch signal demonstration requires POSIX signals")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	binary := filepath.Join(work, "la-famille")
	compile := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/la-famille")
	compile.Dir = repo
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, output)
	}
	publisher, subscriber := filepath.Join(work, "publisher"), filepath.Join(work, "subscriber")
	if err := os.CopyFS(publisher, os.DirFS(filepath.Join(repo, "assets/testdata/pack-ask"))); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(subscriber, 0700); err != nil {
		t.Fatal(err)
	}
	run := func(directory string, args ...string) {
		t.Helper()
		command := exec.CommandContext(ctx, binary, args...)
		command.Dir = directory
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, output)
		}
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	loadFeed := func(directory string) pack.Feed {
		t.Helper()
		var feed pack.Feed
		if err := json.Unmarshal(read(filepath.Join(directory, pack.FeedName)), &feed); err != nil {
			t.Fatal(err)
		}
		return feed
	}
	feedDir, nextDir := filepath.Join(work, "feed"), filepath.Join(work, "next")
	run(publisher, "build")
	run(publisher, "rag")
	run(publisher, "pack", "publish", "--output", feedDir)
	initial := loadFeed(feedDir)
	note := filepath.Join(publisher, "notes/research/birds.md")
	if err := os.WriteFile(note, bytes.ReplaceAll(read(note), []byte("seven days"), []byte("three days")), 0600); err != nil {
		t.Fatal(err)
	}
	run(publisher, "build")
	run(publisher, "rag")
	run(publisher, "pack", "publish", "--previous", feedDir, "--output", nextDir)
	next := loadFeed(nextDir)
	if len(next.Deltas) != 1 || next.Deltas[0].BaseSHA256 != initial.Full.SHA256 {
		t.Fatal("publisher did not produce exact-base delta")
	}
	target := read(filepath.Join(nextDir, next.Full.Path))
	delta := read(filepath.Join(nextDir, next.Deltas[0].Path))
	// Delete only the disposable publisher created by this test.
	if err := os.RemoveAll(publisher); err != nil {
		t.Fatal(err)
	}
	watch := exec.CommandContext(ctx, binary, "pack", "watch", feedDir, "--state", "state", "--interval", "50ms")
	watch.Dir = subscriber
	pipe, err := watch.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	watch.Stderr = watch.Stdout
	if err := watch.Start(); err != nil {
		t.Fatal(err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = watch.Process.Kill()
			_ = watch.Wait()
		}
	}()
	lines := make(chan string, 256)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
	}()
	waitLine := func(text string) {
		t.Helper()
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					t.Fatalf("watch exited before %q", text)
				}
				t.Log(line)
				if strings.Contains(line, text) {
					return
				}
			case <-ctx.Done():
				t.Fatalf("watch did not report %q: %v", text, ctx.Err())
			}
		}
	}
	waitLine("Mode: full")
	stateDir := filepath.Join(subscriber, "state")
	oldPointer := read(filepath.Join(stateDir, pack.CurrentName))
	oldPack := filepath.Join(stateDir, initial.Full.Path)
	oldBytes := read(oldPack)
	// Publish complete artifacts before changing the manifest. A corrupt
	// selected delta must not fall back, even though a valid full is available.
	for name, data := range map[string][]byte{next.Full.Path: target, next.Deltas[0].Path: []byte("corrupt delta")} {
		if err := os.WriteFile(filepath.Join(feedDir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	staging := filepath.Join(feedDir, ".manifest-new")
	if err := os.WriteFile(staging, read(filepath.Join(nextDir, pack.FeedName)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(staging, filepath.Join(feedDir, pack.FeedName)); err != nil {
		t.Fatal(err)
	}
	waitLine("Pack poll failed; current version preserved")
	if !bytes.Equal(oldPointer, read(filepath.Join(stateDir, pack.CurrentName))) || !bytes.Equal(oldBytes, read(oldPack)) {
		t.Fatal("corrupt delta changed previous current")
	}
	run(subscriber, "pack", "verify", oldPack)
	if err := os.WriteFile(filepath.Join(feedDir, next.Deltas[0].Path), delta, 0600); err != nil {
		t.Fatal(err)
	}
	waitLine("Mode: delta")
	waitLine("birds")
	if !bytes.Equal(target, read(filepath.Join(stateDir, next.Full.Path))) || !bytes.Equal(oldBytes, read(oldPack)) {
		t.Fatal("watch result was not exact or modified previous pack")
	}
	run(subscriber, "pack", "verify", filepath.Join(stateDir, next.Full.Path))
	if err := watch.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := watch.Wait(); err != nil {
		t.Fatal(err)
	}
	finished = true
	entries, err := os.ReadDir(subscriber)
	if err != nil || len(entries) != 1 || entries[0].Name() != "state" {
		t.Fatal("subscriber contains files outside durable pack state")
	}
}
