package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/ask"
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
	if err := os.CopyFS(publisher, os.DirFS(filepath.Join(repo, "assets/testdata/pack-ask"))); err != nil {
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

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(work, "ask.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	server := exec.CommandContext(ctx, binary, "ask", "--pack", applied, "--provider", "fake", "--no-browser", "--port", port)
	server.Dir, server.Stdout, server.Stderr = subscriber, log, log
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	}()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	url := "http://" + address
	deadline := time.Now().Add(15 * time.Second)
	for {
		response, err := client.Get(url + "/api/status")
		if err == nil {
			var status ask.Status
			decodeErr := json.NewDecoder(response.Body).Decode(&status)
			_ = response.Body.Close()
			if decodeErr != nil || response.StatusCode != http.StatusOK || !status.Ready ||
				status.Provider != "fake" || status.DocumentCount != 2 || !status.LoopbackOnly {
				t.Fatalf("subscriber status = %+v, %v", status, decodeErr)
			}
			t.Logf("Subscriber status: %+v", status)
			break
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			t.Fatalf("Ask did not start: %v\n%s", err, read(log.Name()))
		}
		time.Sleep(20 * time.Millisecond)
	}
	response, err := client.Post(url+"/api/ask", "application/json",
		strings.NewReader(`{"question":"What is the migration survey interval?"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var answer ask.AnswerResponse
	data, err := io.ReadAll(response.Body)
	if err != nil || json.Unmarshal(data, &answer) != nil || response.StatusCode != http.StatusOK ||
		answer.Status != "answered" || len(answer.Sources) != 1 || len(answer.DroppedCitations) != 0 {
		t.Fatalf("subscriber answer: %s, %v", data, err)
	}
	source := answer.Sources[0]
	if source.Title != "Bird Observation Notes" || source.URL != "/field-guide/bird-observations/" ||
		!strings.Contains(source.Excerpt, "three days") || strings.Contains(source.Excerpt, "seven days") {
		t.Fatalf("subscriber retrieved stale or incorrect evidence: %+v", source)
	}
	t.Logf("Subscriber answer: %s", data)
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
