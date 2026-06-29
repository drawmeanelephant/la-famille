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
	err = os.WriteFile(filepath.Join(tempDir, "internal", "foo", "foo.go"), []byte("package foo"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.MkdirAll(filepath.Join(tempDir, "assets"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tempDir, "assets", "logo.png"), []byte("PNG"), 0644)
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
