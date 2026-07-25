package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

// newScaffoldSite lays out a content directory beside an "outside" directory
// holding a file that must never be touched.
func newScaffoldSite(t *testing.T) (root, contentDir, outsideDir string) {
	t.Helper()
	root = t.TempDir()
	contentDir = filepath.Join(root, "content")
	outsideDir = filepath.Join(root, "outside")
	for _, d := range []string{contentDir, outsideDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "existing.md"), []byte("PRECIOUS_ORIGINAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, contentDir, outsideDir
}

func runNew(t *testing.T, contentDir string, args ...string) error {
	t.Helper()
	cmd := setupNewCmd(config.Config{ContentDir: contentDir})
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	cmd.SetArgs(append(args, "--content", contentDir))
	return cmd.Execute()
}

// TestNewRefusesSymlinkedDirectoryComponent covers the escape: containment was
// checked lexically, so a symlinked directory inside the content tree satisfied
// the check while sending the write wherever the link pointed.
func TestNewRefusesSymlinkedDirectoryComponent(t *testing.T) {
	_, contentDir, outsideDir := newScaffoldSite(t)

	if err := os.Symlink(outsideDir, filepath.Join(contentDir, "link")); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	err := runNew(t, contentDir, "link/pwn")
	if err == nil {
		t.Fatal("expected `new link/pwn` to be refused")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("error should name the symlink, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "pwn.md")); statErr == nil {
		t.Error("a file was created outside the content directory")
	}
}

// TestNewRefusesSymlinkedDestinationEvenWithForce pins the harder case: --force
// means "replace the file you named", never "follow this link and truncate
// whatever is on the other end".
func TestNewRefusesSymlinkedDestinationEvenWithForce(t *testing.T) {
	_, contentDir, outsideDir := newScaffoldSite(t)
	victim := filepath.Join(outsideDir, "existing.md")

	if err := os.Symlink(victim, filepath.Join(contentDir, "direct.md")); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	if err := runNew(t, contentDir, "direct", "--force"); err == nil {
		t.Fatal("expected a symlinked destination to be refused even with --force")
	}

	body, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("the linked-to file should still exist: %v", err)
	}
	if string(body) != "PRECIOUS_ORIGINAL" {
		t.Errorf("the file behind the symlink was modified: %q", body)
	}
}

// TestNewStillScaffoldsOrdinaryPaths keeps the guard from over-reaching.
func TestNewStillScaffoldsOrdinaryPaths(t *testing.T) {
	_, contentDir, _ := newScaffoldSite(t)

	if err := runNew(t, contentDir, "posts/hello"); err != nil {
		t.Fatalf("ordinary scaffolding must still work: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(contentDir, "posts", "hello.md"))
	if err != nil {
		t.Fatalf("expected the file to be created: %v", err)
	}
	if !strings.Contains(string(body), "title:") {
		t.Errorf("scaffolded file is missing frontmatter: %q", body)
	}
}

// TestNewForceStillReplacesARegularFile confirms --force keeps working for the
// case it is actually for.
func TestNewForceStillReplacesARegularFile(t *testing.T) {
	_, contentDir, _ := newScaffoldSite(t)
	existing := filepath.Join(contentDir, "page.md")
	if err := os.WriteFile(existing, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runNew(t, contentDir, "page", "--force"); err != nil {
		t.Fatalf("--force must still replace a regular file: %v", err)
	}
	body, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "OLD") {
		t.Error("--force did not replace the file")
	}
}

// TestNewWithoutForceRefusesAnExistingFile pins the unchanged default.
func TestNewWithoutForceRefusesAnExistingFile(t *testing.T) {
	_, contentDir, _ := newScaffoldSite(t)
	existing := filepath.Join(contentDir, "page.md")
	if err := os.WriteFile(existing, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := runNew(t, contentDir, "page"); err == nil {
		t.Fatal("expected an existing file to be refused without --force")
	}
	body, _ := os.ReadFile(existing)
	if string(body) != "OLD" {
		t.Errorf("the existing file was modified: %q", body)
	}
}
