package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestPackAskBootstrapSkipsUnrelatedConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("invalid: ["), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, args := range [][]string{
		{"ask", "--pack", "/corpus.tar"},
		{"ask", "--pack=/corpus.tar", "--provider=fake"},
	} {
		cfg, err := loadProjectConfig(args)
		if err != nil || cfg.ProjectRoot != root {
			t.Fatalf("pack bootstrap = %+v, %v", cfg, err)
		}
	}
	if _, err := loadProjectConfig([]string{"ask"}); err == nil {
		t.Fatal("directory Ask no longer rejects broken config")
	}
	if packAskInvocation([]string{"ask", "--", "--pack", "/corpus.tar"}) {
		t.Fatal("positional arguments mistaken for a pack flag")
	}
}

func TestPackAskCLIRejectsIncompatibleFlags(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	for _, flag := range []string{
		"--rag-dir=rag", "--output=public", "--rebuild", "--rebuild=false",
		"--config=other.yaml", "--project-root=.", "--eval=questions.json",
		"--eval-k=2", "--eval-compare-graph",
	} {
		t.Run(flag, func(t *testing.T) {
			root := setupRootCmd(config.DefaultConfig())
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs([]string{"ask", "--pack=/corpus.tar", flag})
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), "--pack cannot be combined with") {
				t.Fatalf("flag %s error = %v", flag, err)
			}
		})
	}
	for _, value := range []string{"", "relative.tar", "https://example.test/corpus.tar"} {
		root := setupRootCmd(config.DefaultConfig())
		root.SetArgs([]string{"ask", "--pack=" + value})
		if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "absolute local path") {
			t.Fatalf("pack path %q error = %v", value, err)
		}
	}
}

func TestPackAskConfigGuardAllowsPackLoadError(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	root, state := setupRootCmdState(config.DefaultConfig())
	defer state.closeLogFile()
	guardUnusableConfig(root, state, errors.New("unrelated config is broken"))
	root.SetArgs([]string{"ask", "--pack", filepath.Join(t.TempDir(), "missing.tar"), "--provider=fake", "--no-browser"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root.SetContext(ctx)
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "verify corpus") || strings.Contains(err.Error(), "unrelated config") {
		t.Fatalf("expected pack load error, got %v", err)
	}
}
