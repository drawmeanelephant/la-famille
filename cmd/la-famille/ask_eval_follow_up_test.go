package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestAskEvalHardSetAndDepth(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		fail bool
		want string
	}{
		{"hard failure", []string{"--eval", "assets/testdata/ask-eval/golden-questions-hard.json"}, true, "sensor-warranty-absent"},
		{"depth override", []string{"--eval", "assets/testdata/ask-eval/golden-questions.json", "--eval-k", "8"}, false, "comparison skipped"},
		{"invalid depth", []string{"--eval", "assets/testdata/ask-eval/golden-questions.json", "--eval-k", "101"}, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.ProjectRoot = root
			cmd := setupAskCmd(cfg)
			cmd.SetArgs(tc.args)
			var output bytes.Buffer
			cmd.SetOut(&output)
			err := cmd.Execute()
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v, expected failure=%t", err, tc.fail)
			}
			if tc.want != "" && !strings.Contains(output.String(), tc.want) {
				t.Fatalf("report missing %q:\n%s", tc.want, output.String())
			}
		})
	}
}

func TestAskEvalDatasetGateFailure(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "assets", "testdata", "ask-eval", "golden-questions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var dataset map[string]any
	if err := json.Unmarshal(data, &dataset); err != nil {
		t.Fatal(err)
	}
	dataset["minimum_precision_at_k"] = 0.9
	// Question-specific overrides let all individual questions pass while
	// the aggregate precision gate fails, testing the command's error text.
	for _, site := range dataset["sites"].([]any) {
		for _, question := range site.(map[string]any)["questions"].([]any) {
			question.(map[string]any)["minimum_precision_at_k"] = 0
		}
	}
	data, err = json.Marshal(dataset)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dataset.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.ProjectRoot = root
	cmd := setupAskCmd(cfg)
	cmd.SetArgs([]string{"--eval", path})
	var output bytes.Buffer
	cmd.SetOut(&output)
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "0 questions failed") || !strings.Contains(err.Error(), "precision floor passed=false") {
		t.Fatalf("aggregate gate failure hidden: %v", err)
	}
	if !strings.Contains(output.String(), "evaluation passed: false") {
		t.Fatal("failure report not written before error")
	}
}
