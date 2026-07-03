package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
)

func TestFixtures(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "assets", "testdata", "sites")
	fixtures, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("failed to read fixtures directory: %v", err)
	}

	templateFile := filepath.Join("..", "..", "templates", "layout.html")

	for _, f := range fixtures {
		if !f.IsDir() {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			contentDir := filepath.Join(fixtureDir, f.Name(), "content")
			expectedDir := filepath.Join(fixtureDir, f.Name(), "expected")

			outputDir := t.TempDir()

			cfg := config.DefaultConfig()
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile

			if _, err := generator.Build(cfg); err != nil {
				t.Fatalf("run failed: %v", err)
			}

			// Check all files in expectedDir exist in outputDir and match
			err = filepath.WalkDir(expectedDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(expectedDir, path)
				if err != nil {
					return err
				}

				actualPath := filepath.Join(outputDir, relPath)
				// If the expected file is under 'pages/', it maps to the root of the output directory
				if strings.HasPrefix(relPath, "pages"+string(filepath.Separator)) {
					actualPath = filepath.Join(outputDir, relPath[len("pages")+1:])
				}

				actualContent, err := os.ReadFile(actualPath)
				if err != nil {
					t.Errorf("missing expected file %s (checked %s): %v", relPath, actualPath, err)
					return nil
				}

				expectedContent, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read expected file %s: %v", relPath, err)
				}

				actualStr := strings.ReplaceAll(string(actualContent), "\r\n", "\n")
				expectedStr := strings.ReplaceAll(string(expectedContent), "\r\n", "\n")
				if actualStr != expectedStr {
					t.Errorf("content mismatch in %s:\nExpected:\n%s\nActual:\n%s\n", relPath, expectedStr, actualStr)
				}

				return nil
			})

			if err != nil {
				t.Fatalf("walk failed: %v", err)
			}
		})
	}
}
