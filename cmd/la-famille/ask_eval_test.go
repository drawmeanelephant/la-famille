package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestAskEvalFlagRunsGoldenDatasetWithoutStartingServer(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = projectRoot

	cmd := setupAskCmd(cfg)
	cmd.SetArgs([]string{
		"--eval", "assets/testdata/ask-eval/golden-questions.json",
		"--host", "0.0.0.0",
	})
	var output bytes.Buffer
	cmd.SetOut(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ask --eval: %v", err)
	}
	for _, want := range []string{
		"Ask This Site evaluation: ask-this-site-phase-1",
		"provider=fake",
		"other-page-heading",
		"unanswerable",
		"Recall@5: 1.00",
		"PASS",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("eval output missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "open: http://") {
		t.Errorf("eval started the web server:\n%s", output.String())
	}
}
