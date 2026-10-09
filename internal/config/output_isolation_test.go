package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baseValidConfig is an ordinary layout: sources beside a public/ output, all
// inside the project root. It must keep validating — the overlap rules exist to
// catch mistakes, not to reject the default arrangement.
func baseValidConfig() Config {
	c := DefaultConfig()
	c.Port = 8080
	return c
}

func TestValidateAcceptsOrdinaryLayout(t *testing.T) {
	if err := baseValidConfig().Validate(); err != nil {
		t.Fatalf("the default layout must validate, got: %v", err)
	}
}

// TestValidateRejectsOutputOverlappingInputs is the guard against a build
// deleting its own sources. A successful build renames the output directory
// aside, installs the staged tree, and removes the backup — so an output
// directory that overlaps an input takes that input with it. Verified before
// the fix: output_dir == content_dir deleted every source file and reported
// success.
func TestValidateRejectsOutputOverlappingInputs(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		expect string
	}{
		{
			name:   "output is the content directory",
			mutate: func(c *Config) { c.OutputDir = c.ContentDir },
			expect: "same directory as ContentDir",
		},
		{
			name:   "output is the asset directory",
			mutate: func(c *Config) { c.OutputDir = c.AssetDir },
			expect: "same directory as AssetDir",
		},
		{
			name:   "output is the template directory",
			mutate: func(c *Config) { c.OutputDir = filepath.Dir(c.Template) },
			expect: "same directory as the template directory",
		},
		{
			name:   "output is the rag directory",
			mutate: func(c *Config) { c.OutputDir = c.RagDir },
			expect: "same directory as RagDir",
		},
		{
			name:   "output is the project root",
			mutate: func(c *Config) { c.OutputDir = c.ProjectRoot },
			expect: "project root",
		},
		{
			name:   "content lives inside the output directory",
			mutate: func(c *Config) { c.ContentDir = filepath.Join(c.OutputDir, "content") },
			expect: "inside OutputDir",
		},
		{
			name:   "assets live inside the output directory",
			mutate: func(c *Config) { c.AssetDir = filepath.Join(c.OutputDir, "assets") },
			expect: "inside OutputDir",
		},
		{
			name:   "output lives inside the content directory",
			mutate: func(c *Config) { c.OutputDir = filepath.Join(c.ContentDir, "public") },
			expect: "is inside ContentDir",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := baseValidConfig()
			c.mutate(&cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected %s to be rejected, got nil", c.name)
			}
			if !strings.Contains(err.Error(), c.expect) {
				t.Errorf("error should explain the overlap (%q), got: %v", c.expect, err)
			}
		})
	}
}

// TestValidateComparesDirectoriesCanonically covers two spellings of one
// directory. A lexical comparison alone would let "./content" and "content"
// past as different paths.
func TestValidateComparesDirectoriesCanonically(t *testing.T) {
	cfg := baseValidConfig()
	cfg.ContentDir = "content"
	cfg.OutputDir = "./content/"

	if err := cfg.Validate(); err == nil {
		t.Error("two spellings of the same directory must still be rejected")
	}
}

// TestValidateAllowsSiblingDirectories keeps the rule from over-reaching:
// directories that merely share a name prefix do not overlap.
func TestValidateAllowsSiblingDirectories(t *testing.T) {
	cfg := baseValidConfig()
	cfg.ContentDir = "content"
	cfg.OutputDir = "content-output"

	if err := cfg.Validate(); err != nil {
		t.Errorf("sibling directories sharing a prefix must be allowed, got: %v", err)
	}
}

// TestValidateAllowsInputsThatAreTheProjectRoot covers two ordinary layouts the
// overlap rule rejected outright. Any input that IS the project root contains
// the output directory by definition — the same reason ProjectRoot itself is
// treated separately — so applying "output inside an input" to them made
// working sites unbuildable with no legal escape.
func TestValidateAllowsInputsThatAreTheProjectRoot(t *testing.T) {
	cases := []struct {
		mutate func(*Config)
		name   string
	}{
		{
			// A single layout.html beside config.yaml: filepath.Dir gives ".".
			name:   "template at the project root",
			mutate: func(c *Config) { c.Template = "layout.html" },
		},
		{
			// A flat site with its markdown at the repository root.
			name:   "content directory is the project root",
			mutate: func(c *Config) { c.ContentDir = "." },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := baseValidConfig()
			c.mutate(&cfg)
			if err := cfg.Validate(); err != nil {
				t.Errorf("this layout must build, got: %v", err)
			}
		})
	}
}

// Issue #642: `asset_dir: "."` used to validate and then fail on every build
// when the copier refused to walk a root containing the output directory.
// Unlike content_dir — which the generator only reads — the asset directory
// is mirrored wholesale into the output, so an asset root that IS the project
// root has no safe interpretation. It must now fail validation with a clear
// message.
func TestValidateRejectsAssetDirEqualToProjectRoot(t *testing.T) {
	cfg := baseValidConfig()
	cfg.AssetDir = "."

	err := cfg.Validate()
	if err == nil {
		t.Fatal("asset_dir == project root must be rejected")
	}
	if !strings.Contains(err.Error(), "AssetDir") || !strings.Contains(err.Error(), "project root") {
		t.Errorf("error should explain that the asset dir is the project root, got: %v", err)
	}
}

// Issue #642: a regular file at the asset_dir path used to fail mid-build
// with an error blaming the staging destination. ValidateResolved must catch
// it instead, naming asset_dir; a missing asset dir stays legal.
func TestValidateResolvedRejectsFileAssetDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "assets"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	resolved, err := DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	err = resolved.ValidateResolved()
	if err == nil {
		t.Fatal("a regular-file asset_dir must fail validation")
	}
	if !strings.Contains(err.Error(), `asset_dir "assets" is not a directory`) {
		t.Errorf("error should name asset_dir and the non-directory problem, got: %v", err)
	}
}

func TestValidateResolvedAllowsMissingAssetDir(t *testing.T) {
	root := t.TempDir()
	resolved, err := DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.ValidateResolved(); err != nil {
		t.Errorf("a missing asset dir is legal, the copier tolerates it; got: %v", err)
	}
}

func TestValidateResolvedAllowsDirectoryAssetDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	resolved, err := DefaultConfig().ResolvePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.ValidateResolved(); err != nil {
		t.Errorf("an ordinary asset dir must still validate, got: %v", err)
	}
}

// TestValidateRejectsRagDirContainingProjectRoot covers #643: a RAG output
// directory that is — or contains — the project root makes the export's own
// exclusion match every source file, so the command would write empty
// bundles and report success. Validation must refuse the layout.
func TestValidateRejectsRagDirContainingProjectRoot(t *testing.T) {
	t.Run("rag dir is the project root", func(t *testing.T) {
		cfg := baseValidConfig()
		cfg.RagDir = "."
		if err := cfg.Validate(); err == nil {
			t.Fatal("rag_dir equal to the project root must be rejected")
		} else if !strings.Contains(err.Error(), "RagDir") {
			t.Errorf("error should name RagDir, got: %v", err)
		}
	})

	t.Run("resolved rag dir is the project root", func(t *testing.T) {
		root := t.TempDir()
		cfg := baseValidConfig()
		cfg.ProjectRoot = root
		cfg.RagDir = root
		if err := cfg.ValidateResolved(); err == nil {
			t.Fatal("a resolved RagDir equal to the project root must be rejected")
		} else if !strings.Contains(err.Error(), "RagDir") {
			t.Errorf("error should name RagDir, got: %v", err)
		}
	})

	t.Run("rag dir contains the project root", func(t *testing.T) {
		root := t.TempDir()
		cfg := baseValidConfig()
		// The project root must exist for canonical comparison: only then do
		// both sides resolve through the same symlinked path components.
		if err := os.MkdirAll(filepath.Join(root, "site"), 0755); err != nil {
			t.Fatal(err)
		}
		cfg.ProjectRoot = filepath.Join(root, "site")
		cfg.RagDir = root
		if err := cfg.ValidateResolved(); err == nil {
			t.Fatal("a RagDir containing the project root must be rejected")
		} else if !strings.Contains(err.Error(), "RagDir") {
			t.Errorf("error should name RagDir, got: %v", err)
		}
	})
}

func TestValidateAllowsRagArchiveInsidePublishOutput(t *testing.T) {
	cfg := baseValidConfig()
	cfg.RagDir = filepath.Join(cfg.OutputDir, "rag-archive")
	if err := cfg.Validate(); err != nil {
		t.Fatalf("RAG output below public is a supported Pages layout, got: %v", err)
	}
}

// TestValidateStillRejectsOutputEqualToARootInput keeps the relaxation narrow:
// an input at the project root may contain the output, but may not BE it.
func TestValidateStillRejectsOutputEqualToARootInput(t *testing.T) {
	cfg := baseValidConfig()
	cfg.ContentDir = "."
	cfg.OutputDir = "."

	if err := cfg.Validate(); err == nil {
		t.Error("output equal to a content directory at the project root must still be refused")
	}
}
