package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pack"
	"gopkg.in/yaml.v3"
)

func TestPackFeedCommandFlagsAndConfigIndependence(t *testing.T) {
	root := t.TempDir()
	cfg, err := config.DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"pack", "pull", "https://example.com/pack-feed.json", "--output", "out"},
		{"pack", "pull", ".", "--output", "out", "--timeout", "0"},
		{"pack", "watch", "."}, {"pack", "watch", ".", "--state", "state"},
		{"pack", "watch", ".", "--state", "state", "--interval", "0"},
		{"pack", "watch", "https://example.com/pack-feed.json", "--state", "state", "--interval", "1s"},
		{"pack", "publish", "--output="},
		{"pack", "publish", "--output", "feed", "--retain", "9"},
		{"pack", "publish", "--output", "feed", "--previous="},
	} {
		cmd, state := setupRootCmdState(cfg)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted invalid command: %v", args)
		}
		state.closeLogFile()
	}
	cmd, state := setupRootCmdState(cfg)
	defer state.closeLogFile()
	found, _, err := cmd.Find([]string{"pack", "watch"})
	if err != nil || requiresSiteConfig(found) {
		t.Fatal("watch must work without site configuration")
	}
}

func TestPackFeedPublishAndWatchCommands(t *testing.T) {
	root := t.TempDir()
	cfg, err := config.DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"backlinks.json", "graph.json", "meta.json", "search.json", "site-manifest.json"} {
		path := filepath.Join(cfg.OutputDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(cfg.RagDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.RagDir, "rag-content.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(ctx context.Context, args ...string) string {
		t.Helper()
		cmd, state := setupRootCmdState(cfg)
		defer state.closeLogFile()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.ExecuteContext(ctx); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	output := run(context.Background(), "pack", "publish", "--output", "feed")
	if !strings.Contains(output, "0 retained deltas") {
		t.Fatalf("publication: %s", output)
	}
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel when stdout reports the first complete update, not after a sleep.
	cmd, state := setupRootCmdState(cfg)
	defer state.closeLogFile()
	var watched bytes.Buffer
	cmd.SetOut(cancelOnWrite{writer: &watched, cancel: cancel})
	cmd.SetErr(&watched)
	cmd.SetArgs([]string{"pack", "watch", "feed", "--state", "subscriber", "--interval", "1h"})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(watched.String(), "Current pack:") || !strings.Contains(watched.String(), "Mode: full") {
		t.Fatalf("watch: %s", watched.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "subscriber", pack.CurrentName))
	if err != nil {
		t.Fatal(err)
	}
	var current pack.SubscriberState
	if err := json.Unmarshal(data, &current); err != nil {
		t.Fatal(err)
	}
	run(context.Background(), "pack", "verify", filepath.Join("subscriber", current.Path))
}

type cancelOnWrite struct {
	writer *bytes.Buffer
	cancel context.CancelFunc
}

func (w cancelOnWrite) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	w.cancel()
	return n, err
}

// An interrupted pull must exit non-zero with a message, not exit 0 silent
// with no pack on disk (#656).
func TestPackPullCanceledContextFails(t *testing.T) {
	root := t.TempDir()
	cfg, err := config.DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"backlinks.json", "graph.json", "meta.json", "search.json", "site-manifest.json"} {
		path := filepath.Join(cfg.OutputDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(cfg.RagDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.RagDir, "rag-content.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	feedDir := filepath.Join(root, "feed")
	if err := os.MkdirAll(feedDir, 0700); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(feedDir, "full.tar")
	if _, err := pack.Build(pack.BuildOptions{
		OutputDir: cfg.OutputDir, RagDir: cfg.RagDir,
		Site:       pack.Site{Name: "CLI site", URL: "https://example.com"},
		Provenance: pack.Provenance{Generator: "test", Version: "dev"},
	}, full); err != nil {
		t.Fatal(err)
	}
	archive, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	feed := pack.Feed{
		SchemaVersion: pack.FeedVersion,
		Full:          pack.FeedPack{Path: "full.tar", SHA256: fmt.Sprintf("%x", sha256.Sum256(archive))},
		Deltas:        []pack.FeedDelta{},
	}
	data, err := json.Marshal(feed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(feedDir, pack.FeedName), data, 0600); err != nil {
		t.Fatal(err)
	}
	pull := func(ctx context.Context, output string) (string, error) {
		t.Helper()
		cmd := setupPackPullCmd(cfg)
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{feedDir, "--output", output})
		err := cmd.ExecuteContext(ctx)
		return out.String(), err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := pull(ctx, "interrupted.tar")
	if err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("interrupted pull = %q, %v", out, err)
	}
	if strings.Contains(out, "Pulled pack") {
		t.Fatalf("interrupted pull printed success: %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(root, "interrupted.tar")); !os.IsNotExist(statErr) {
		t.Fatalf("interrupted pull left an output: %v", statErr)
	}
	out, err = pull(context.Background(), "done.tar")
	if err != nil || !strings.Contains(out, "mode full") {
		t.Fatalf("control pull = %q, %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(root, "done.tar")); err != nil {
		t.Fatal("control pull produced no pack")
	}
}

func TestFlagshipCorpusFeedWorkflowContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "website.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Concurrency struct {
			Cancel bool `yaml:"cancel-in-progress"`
		}
		Jobs map[string]struct {
			Steps []struct {
				Name string
				Uses string
				Run  string
				With map[string]string
			}
		}
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if workflow.Concurrency.Cancel {
		t.Fatal("production publications must serialize without cancellation")
	}
	restore, build, save, deployment := -1, -1, -1, -1
	for i, step := range workflow.Jobs["build"].Steps {
		if step.Uses == "actions/cache/restore@v6" {
			restore = i
			if step.With["path"] != "${{ runner.temp }}/previous-pack-feed" ||
				step.With["restore-keys"] != "flagship-pack-feed-" {
				t.Fatal("history restore must use the existing bounded cache")
			}
		}
		if strings.Contains(step.Run, "pack publish") {
			build = i
			for _, marker := range []string{"--retain 3", `--previous "$RUNNER_TEMP/previous-pack-feed"`,
				`--output "$GITHUB_WORKSPACE/public/corpus-packs"`, "Cache-Control: no-store",
				"max-age=31536000, immutable"} {
				if !strings.Contains(step.Run, marker) {
					t.Errorf("missing feed publication contract %q", marker)
				}
			}
		}
		if step.Uses == "actions/cache/save@v6" {
			t.Fatal("untrusted builds cannot save production history")
		}
	}
	for i, step := range workflow.Jobs["deploy"].Steps {
		if step.Uses == "cloudflare/wrangler-action@v3" {
			deployment = i
		}
		if step.Uses == "actions/cache/save@v6" {
			save = i
		}
	}
	if restore < 0 || build <= restore || deployment < 0 || save <= deployment {
		t.Fatal("history must restore before building and save only after deployment")
	}
	if pack.DefaultRetainedVersions != 3 || pack.DefaultRequestTimeout != 2*time.Minute {
		t.Fatal("documented subscriber/publisher defaults drifted")
	}
}
