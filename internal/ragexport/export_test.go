package ragexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestRunExport_ProjectRoot(t *testing.T) {
	// Create a temp directory to represent our project
	tempDir := t.TempDir()

	// Create some files inside the project
	err := os.MkdirAll(filepath.Join(tempDir, "internal", "foo"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "internal", "foo", "foo.go"), []byte("package foo"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(tempDir, "assets"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "assets", "logo.png"), []byte("PNG"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// We'll run the export from a DIFFERENT working directory
	invokeDir := t.TempDir()

	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	// Verify the output exists in my-rag
	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	// The path in the bundle should be relative to ProjectRoot
	expectedPath := "<file path=\"internal/foo/foo.go\">"
	if !strings.Contains(string(content), expectedPath) {
		t.Errorf("Expected system bundle to contain %q, but it didn't.\nContent:\n%s", expectedPath, content)
	}

	configBundlePath := filepath.Join(invokeDir, "my-rag", "rag-config.md")
	cfgContent, err := os.ReadFile(configBundlePath)
	if err != nil {
		t.Fatalf("Failed to read config bundle: %v", err)
	}

	expectedAssetPath := "assets/logo.png"
	if !strings.Contains(string(cfgContent), expectedAssetPath) {
		t.Errorf("Expected config bundle to contain %q, but it didn't.\nContent:\n%s", expectedAssetPath, cfgContent)
	}
}

func TestRunExport_RootLevelMatch(t *testing.T) {
	tempDir := t.TempDir()

	// Should be included (root)
	err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("Root README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "root.go"), []byte("package main"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Should be excluded (nested)
	err = os.MkdirAll(filepath.Join(tempDir, "nested"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "README.md"), []byte("Nested README"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "nested", "nested.go"), []byte("package nested"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	invokeDir := t.TempDir()
	cfg := config.Config{
		ProjectRoot: tempDir,
		RagDir:      filepath.Join(invokeDir, "my-rag"),
	}

	err = RunExport(cfg)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	systemBundlePath := filepath.Join(invokeDir, "my-rag", "rag-system.md")
	content, err := os.ReadFile(systemBundlePath)
	if err != nil {
		t.Fatalf("Failed to read system bundle: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "<file path=\"README.md\">") {
		t.Errorf("Expected system bundle to contain root README.md")
	}
	if strings.Contains(contentStr, "<file path=\"nested/README.md\">") {
		t.Errorf("Expected system bundle NOT to contain nested/README.md")
	}

	if !strings.Contains(contentStr, "<file path=\"root.go\">") {
		t.Errorf("Expected system bundle to contain root.go")
	}
	if strings.Contains(contentStr, "<file path=\"nested/nested.go\">") {
		t.Errorf("Expected system bundle NOT to contain nested/nested.go")
	}
}
