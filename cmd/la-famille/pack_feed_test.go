package main

import (
	"bytes"
	"context"
	"encoding/json"
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
