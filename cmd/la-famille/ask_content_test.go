package main

import (
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestAskContentDirIsProjectRootRelative(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		ProjectRoot: root,
		ContentDir:  filepath.Join(root, "vault"),
	}
	if got := askContentDir(cfg); got != "vault" {
		t.Fatalf("askContentDir() = %q, want vault", got)
	}
}

func TestAskContentDirDefaultsToContent(t *testing.T) {
	if got := askContentDir(config.Config{}); got != "content" {
		t.Fatalf("askContentDir() = %q, want content", got)
	}
}
