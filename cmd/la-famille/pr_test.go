package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/github"
)

func TestPRSyncFlagDefaults(t *testing.T) {
	flags := map[string]string{
		"base":                  "",
		"apply":                 "false",
		"required-label":        github.DefaultRequiredLabel,
		"close-conflicts":       "false",
		"allow-no-checks":       "false",
		"publish-local-changes": "false",
	}
	for name, want := range flags {
		f := prSyncCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("missing flag %q", name)
		}
		if f.DefValue != want {
			t.Errorf("flag %s DefValue=%q, want %q", name, f.DefValue, want)
		}
	}

	bot := prSyncCmd.Flags().Lookup("bot-author")
	if bot == nil {
		t.Fatal("missing bot-author flag")
	}
	// StringSlice default is serialized as CSV-like value
	if !strings.Contains(bot.DefValue, "google-labs-jules") || !strings.Contains(bot.DefValue, "google-labs-code") {
		t.Errorf("bot-author default = %q", bot.DefValue)
	}

	if prSyncCmd.Flags().Lookup("head-prefix") == nil {
		t.Fatal("missing head-prefix flag")
	}
	if prSyncCmd.Flags().Lookup("dry-run") != nil {
		t.Fatal("must not add separate --dry-run flag; --apply is the mutation switch")
	}
}

func TestPRSyncMissingToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	cmd := prSyncCmd
	cmd.SetArgs([]string{})
	// Reset flags between tests by re-parsing empty
	_ = cmd.Flags().Parse([]string{})

	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error without GITHUB_TOKEN")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error = %v", err)
	}
}

func TestPRSyncEmptyRequiredLabelRejected(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	cmd := prSyncCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Flags().Set("required-label", ""); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	// restore default for other tests
	_ = cmd.Flags().Set("required-label", github.DefaultRequiredLabel)
	if err == nil {
		t.Fatal("expected error for empty required-label")
	}
	if !strings.Contains(err.Error(), "required-label") {
		t.Errorf("error = %v", err)
	}
}

func TestPRSyncDefaultIsDryRun(t *testing.T) {
	apply := prSyncCmd.Flags().Lookup("apply")
	if apply == nil || apply.DefValue != "false" {
		t.Fatalf("apply default = %v", apply)
	}
}

func TestFormatSyncResultViaPackage(t *testing.T) {
	// Stable summary is produced by github.FormatSyncResult and written to cmd stdout.
	var buf bytes.Buffer
	result := github.SyncResult{
		Owner:      "o",
		Repo:       "r",
		BaseBranch: "master",
		Apply:      false,
		Inspected:  0,
	}
	if err := github.FormatSyncResult(&buf, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Mode: dry-run") {
		t.Errorf("output=%s", buf.String())
	}
}

func TestCronSyncWorkflowPolicy(t *testing.T) {
	content := readRepoFile(t, ".github/workflows/cron-sync.yml")
	if !strings.Contains(content, "name: Clear the Litterbox") {
		t.Error("job name must remain Clear the Litterbox")
	}
	if !strings.Contains(content, "pr-litterbox") {
		t.Error("cron-sync should use concurrency group pr-litterbox")
	}
	if !strings.Contains(content, "checks: read") {
		t.Error("cron-sync should request checks: read")
	}

	// Inspect only the live invocation (comments may document disabled flags).
	runIdx := strings.Index(content, "go run ./cmd/la-famille pr sync")
	if runIdx < 0 {
		t.Fatal("cron-sync missing pr sync invocation")
	}
	runBlock := content[runIdx:]
	if end := strings.Index(runBlock, "\n\n"); end > 0 {
		runBlock = runBlock[:end]
	}
	if !strings.Contains(runBlock, "--apply") {
		t.Error("cron-sync must pass --apply")
	}
	if !strings.Contains(runBlock, "--required-label litterbox-approved") {
		t.Error("cron-sync must require litterbox-approved")
	}
	if !strings.Contains(runBlock, "--base master") {
		t.Error("cron-sync should explicitly target master")
	}
	for _, forbidden := range []string{"--close-conflicts", "--publish-local-changes", "--allow-no-checks"} {
		if strings.Contains(runBlock, forbidden) {
			t.Errorf("cron-sync invocation must not pass %s", forbidden)
		}
	}
}

func TestJulesCINoIndependentMerge(t *testing.T) {
	content := readRepoFile(t, ".github/workflows/jules-ci.yml")
	if strings.Contains(content, "gh pr merge") {
		t.Error("jules-ci must not run gh pr merge")
	}
	if strings.Contains(content, "verify-and-merge") {
		t.Error("job should be renamed away from verify-and-merge")
	}
	if !strings.Contains(content, "verify-jules-pr") {
		t.Error("expected verify-jules-pr job name")
	}
	if strings.Contains(content, "pull-requests: write") {
		t.Error("jules-ci should not have PR write permissions")
	}
}

func TestDocsAgreeWithPolicy(t *testing.T) {
	prDoc := readRepoFile(t, "content/docs/pr.md")
	cliDoc := readRepoFile(t, "content/docs/cli.md")
	for _, content := range []string{prDoc, cliDoc} {
		if !strings.Contains(content, "--apply") {
			t.Error("docs must mention --apply")
		}
		if !strings.Contains(content, "litterbox-approved") {
			t.Error("docs must mention litterbox-approved")
		}
	}
	if strings.Contains(prDoc, "Defaults to `main`") {
		t.Error("pr.md must not claim hardcoded main default")
	}
	if !strings.Contains(prDoc, "dry-run") && !strings.Contains(prDoc, "dry run") {
		t.Error("pr.md should document dry-run default")
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// cmd/la-famille -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}
