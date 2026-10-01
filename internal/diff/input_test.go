package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInputAcceptsManifestAndOutputDirectory(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "public")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "rename-before.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(outputDir, "site-manifest.json")
	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	for _, input := range []string{outputDir, manifestPath, "public"} {
		manifest, err := LoadInput(input, "public", root)
		if err != nil {
			t.Errorf("LoadInput(%q) error = %v", input, err)
			continue
		}
		if len(manifest.Pages) != 4 {
			t.Errorf("LoadInput(%q) pages = %d, want 4", input, len(manifest.Pages))
		}
	}
}

func TestLoadInputAcceptsGitRef(t *testing.T) {
	root := t.TempDir()
	fixture, err := os.ReadFile(filepath.Join("testdata", "rename-before.json"))
	if err != nil {
		t.Fatal(err)
	}
	originalRunGit := runGit
	t.Cleanup(func() { runGit = originalRunGit })
	var calls [][]string
	runGit = func(gitRoot string, args ...string) ([]byte, error) {
		if gitRoot != root {
			t.Errorf("Git root = %q, want %q", gitRoot, root)
		}
		calls = append(calls, append([]string(nil), args...))
		if args[0] == "rev-parse" {
			if args[len(args)-1] != "feature/notes^{commit}" {
				t.Errorf("resolved ref = %q, want feature/notes commit", args[len(args)-1])
			}
			return []byte("0123456789abcdef\n"), nil
		}
		if args[0] == "show" {
			if args[1] != "0123456789abcdef:./dist/site-manifest.json" {
				t.Errorf("show argument = %q", args[1])
			}
			return fixture, nil
		}
		t.Fatalf("unexpected Git command: %v", args)
		return nil, nil
	}

	manifest, err := LoadInput("ref:feature/notes", "dist", root)
	if err != nil {
		t.Fatalf("LoadInput() error = %v", err)
	}
	if len(manifest.Pages) != 4 {
		t.Errorf("pages = %d, want 4", len(manifest.Pages))
	}
	if len(calls) != 2 {
		t.Fatalf("Git calls = %d, want resolve and show", len(calls))
	}
}

func TestLoadInputRejectsGitRefForOutputOutsideProject(t *testing.T) {
	_, err := LoadInput("ref:HEAD", filepath.Join(t.TempDir(), "external"), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "outside project root") {
		t.Fatalf("LoadInput() error = %v, want outside-project-root error", err)
	}
}
