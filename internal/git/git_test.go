package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		url           string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{"https://github.com/tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"https://github.com/tbuddy/la-famille", "tbuddy", "la-famille", false},
		{"http://github.com/owner/repo.git", "owner", "repo", false},
		{"git@github.com:tbuddy/la-famille.git", "tbuddy", "la-famille", false},
		{"git@github.com:owner/repo", "owner", "repo", false},
		{"https://gitlab.com/owner/repo", "", "", true},
		{"invalid-url", "", "", true},
	}

	for _, tt := range tests {
		owner, repo, err := ParseOwnerRepo(tt.url)
		if tt.expectError {
			if err == nil {
				t.Errorf("expected error for url %q, got none", tt.url)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for url %q: %v", tt.url, err)
			continue
		}
		if owner != tt.expectedOwner {
			t.Errorf("for url %q expected owner %q, got %q", tt.url, tt.expectedOwner, owner)
		}
		if repo != tt.expectedRepo {
			t.Errorf("for url %q expected repo %q, got %q", tt.url, tt.expectedRepo, repo)
		}
	}
}

// fixtureRepo creates a real git repository with an origin remote, one
// commit, and a checked-out branch of the given name.
func fixtureRepo(t *testing.T, remoteURL, branch string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-b", branch)
	run("remote", "add", "origin", remoteURL)
	run("-c", "user.email=test@example.com", "-c", "user.name=test",
		"commit", "--allow-empty", "-m", "init")
	return dir
}

// Issue #640: every Repo command must run inside Repo.Dir. Run from repo A, a
// Repo bound to B must see B's remote, branch and working tree — the bug let
// the process CWD silently win, which pointed `pr sync` at the wrong
// repository.
func TestRepoCommandsRunInDir(t *testing.T) {
	repoA := fixtureRepo(t, "https://github.com/alice/repo-a.git", "alice-main")
	repoB := fixtureRepo(t, "https://github.com/bob/repo-b.git", "bob-main")
	t.Chdir(repoA)

	b := Repo{Dir: repoB}
	url, err := b.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("GetRemoteURL: %v", err)
	}
	if url != "https://github.com/bob/repo-b.git" {
		t.Errorf("GetRemoteURL = %q, want repo B's remote", url)
	}
	branch, err := b.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "bob-main" {
		t.Errorf("CurrentBranch = %q, want repo B's branch bob-main", branch)
	}
	dirty, err := b.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges: %v", err)
	}
	if dirty {
		t.Error("freshly committed repo B should be clean")
	}
	if err := os.WriteFile(filepath.Join(repoB, "dirty.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if dirty, err = b.HasUncommittedChanges(); err != nil || !dirty {
		t.Errorf("HasUncommittedChanges = %v, %v; want repo B's dirty tree seen", dirty, err)
	}

	// The zero Repo must keep the historical behavior: commands run in the
	// process working directory (repo A here).
	urlA, err := (Repo{}).GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("zero Repo GetRemoteURL: %v", err)
	}
	if urlA != "https://github.com/alice/repo-a.git" {
		t.Errorf("zero Repo GetRemoteURL = %q, want repo A's remote", urlA)
	}
}
