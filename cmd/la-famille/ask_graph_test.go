package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func cliGraphProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAskGraphComparisonRunsBothLexicalArms(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = cliGraphProjectRoot(t)
	cmd := setupAskCmd(cfg)
	cmd.SetArgs([]string{"--eval", "assets/testdata/ask-eval/golden-questions-graph.json",
		"--eval-compare-graph", "--embeddings", "--no-embeddings"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"graph off", "graph on", "comparison passed: true",
		"sensor → registry → assay", "cited/named all pages: true", "Single-page non-dilution: true"} {
		if !strings.Contains(out.String(), label) {
			t.Fatalf("missing %q in CLI report:\n%s", label, out.String())
		}
	}
	if strings.Contains(out.String(), "open: http://") {
		t.Fatal("eval started a listener")
	}
}

func TestAskGraphComparisonRejectsEmbeddingAndServingModes(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = cliGraphProjectRoot(t)
	for _, args := range [][]string{
		{"--eval-compare-graph"},
		{"--eval", "assets/testdata/ask-eval/golden-questions-graph.json", "--eval-compare-graph", "--embeddings"},
		{"--eval", "assets/testdata/ask-eval/golden-questions-graph.json", "--graph-expansion", "--embeddings"},
	} {
		cmd := setupAskCmd(cfg)
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted invalid graph flags: %v", args)
		}
	}
}
