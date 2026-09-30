package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestAskNoEmbeddingsKeepsGoldenBaseline(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	for _, args := range [][]string{
		{"--eval", "assets/testdata/ask-eval/golden-questions.json", "--no-embeddings"},
		{"--eval", "assets/testdata/ask-eval/golden-questions.json", "--embeddings", "--no-embeddings"},
	} {
		cmd := setupAskCmd(cfg)
		cmd.SetArgs(args)
		var b bytes.Buffer
		cmd.SetOut(&b)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), "Ranker: BM25-lite ·") || !strings.Contains(b.String(), "Recall@5: 1.0000") {
			t.Fatalf("disabled embeddings changed evaluation: %s", b.String())
		}
	}
}
