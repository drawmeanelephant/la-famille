package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtures(t *testing.T) {
	sitesDir := filepath.Join("..", "..", "testdata", "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		t.Fatalf("failed to read sites dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fixtureName := entry.Name()
		t.Run(fixtureName, func(t *testing.T) {
			fixtureDir := filepath.Join(sitesDir, fixtureName)
			contentDir := filepath.Join(fixtureDir, "content")
			expectedDir := filepath.Join(fixtureDir, "expected")

			tempDir := t.TempDir()
			templateFile := filepath.Join(tempDir, "layout.html")
			os.MkdirAll(filepath.Dir(templateFile), 0755)
			os.WriteFile(templateFile, []byte("<html><head><title>{{.Title}}</title></head><body><h1>{{.Title}}</h1>{{.Content}}</body></html>"), 0644)

			outputDir := filepath.Join(tempDir, "public")

			if err := run(contentDir, templateFile, outputDir); err != nil {
				t.Fatalf("run failed for %s: %v", fixtureName, err)
			}

			compareJSONFiles(t, filepath.Join(expectedDir, "graph.json"), filepath.Join(outputDir, "graph.json"))
			compareJSONFiles(t, filepath.Join(expectedDir, "backlinks.json"), filepath.Join(outputDir, "backlinks.json"))
			compareJSONFiles(t, filepath.Join(expectedDir, "meta.json"), filepath.Join(outputDir, "meta.json"))

			expectedPagesDir := filepath.Join(expectedDir, "pages")
			actualFiles := make(map[string]string)
			filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() { return nil }
				if filepath.Ext(path) == ".json" { return nil }

				rel, _ := filepath.Rel(outputDir, path)
				content, _ := os.ReadFile(path)
				actualFiles[rel] = string(content)
				return nil
			})

			expectedFiles := make(map[string]bool)
			filepath.Walk(expectedPagesDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() { return nil }
				rel, _ := filepath.Rel(expectedPagesDir, path)
				expectedFiles[rel] = true
				return nil
			})

			for expectedRel := range expectedFiles {
				if _, ok := actualFiles[expectedRel]; !ok {
					t.Errorf("Fixture %s: Expected generated file %s is missing", fixtureName, expectedRel)
				}
			}

			for actualRel := range actualFiles {
				if !expectedFiles[actualRel] {
					t.Errorf("Fixture %s: Unexpected generated file %s found", fixtureName, actualRel)
				}
			}
		})
	}
}

func compareJSONFiles(t *testing.T, expectedFile, actualFile string) {
	t.Helper()
	expectedBytes, err := os.ReadFile(expectedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("failed to read expected JSON: %v", err)
	}

	actualBytes, err := os.ReadFile(actualFile)
	if err != nil {
		t.Fatalf("failed to read actual JSON: %v", err)
	}

	var expected, actual interface{}
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatalf("failed to parse expected JSON: %v", err)
	}
	if err := json.Unmarshal(actualBytes, &actual); err != nil {
		t.Fatalf("failed to parse actual JSON: %v", err)
	}

	eBytes, _ := json.MarshalIndent(expected, "", "  ")
	aBytes, _ := json.MarshalIndent(actual, "", "  ")

	if !bytes.Equal(eBytes, aBytes) {
		t.Errorf("JSON mismatch in %s\nExpected:\n%s\n\nActual:\n%s", expectedFile, string(eBytes), string(aBytes))
	}
}
