package asset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
)

func TestCopyAssets(t *testing.T) {
	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(filepath.Join(assetDir, "css"), 0755)
	_ = os.MkdirAll(filepath.Join(assetDir, "testdata"), 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "css", "style.css"), []byte("h1 { color: blue; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "testdata", "ignore.txt"), []byte("ignore me"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed: %v", err)
	}

	// Verify copied files
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "css", "style.css")); os.IsNotExist(err) {
		t.Errorf("style.css was not copied")
	}

	// Verify skipped testdata
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "testdata")); !os.IsNotExist(err) {
		t.Errorf("testdata was copied, but should have been skipped")
	}
}

func TestCopyAssets_EmptyAssetDir(t *testing.T) {
	cfg := config.Config{
		AssetDir:  "",
		OutputDir: t.TempDir(),
	}
	err := CopyAssets(cfg)
	if err != nil {
		t.Errorf("Expected nil error for empty AssetDir, got: %v", err)
	}
}
func TestCopyAssets_SkipGoAndGitignore(t *testing.T) {
	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(assetDir, 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.go"), []byte("package main"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed: %v", err)
	}

	// Verify copied files
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.go")); !os.IsNotExist(err) {
		t.Errorf("main.go was copied, but should have been skipped")
	}
}

func TestCopyAssetsGitNotAvailable(t *testing.T) {
	// Temporarily stub PATH to make git unavailable
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", "")

	tempDir := t.TempDir()

	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	// Create asset dir and some files
	_ = os.MkdirAll(filepath.Join(assetDir, "css"), 0755)

	_ = os.WriteFile(filepath.Join(assetDir, "main.css"), []byte("body { color: red; }"), 0600)
	_ = os.WriteFile(filepath.Join(assetDir, "css", "style.css"), []byte("h1 { color: blue; }"), 0600)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets failed when git is not available: %v", err)
	}

	// Verify copied files even when git check-ignore is skipped
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "main.css")); os.IsNotExist(err) {
		t.Errorf("main.css was not copied")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "assets", "css", "style.css")); os.IsNotExist(err) {
		t.Errorf("style.css was not copied")
	}
}
