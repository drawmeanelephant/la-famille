package asset

import (
	"time"

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

func TestCopyAssets_Incremental(t *testing.T) {
	tempDir := t.TempDir()
	assetDir := filepath.Join(tempDir, "assets")
	outputDir := filepath.Join(tempDir, "public")

	_ = os.MkdirAll(assetDir, 0755)

	mockAssetPath := filepath.Join(assetDir, "mock.txt")
	_ = os.WriteFile(mockAssetPath, []byte("initial content"), 0600)

	initialTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	_ = os.Chtimes(mockAssetPath, initialTime, initialTime)

	cfg := config.Config{
		AssetDir:  assetDir,
		OutputDir: outputDir,
	}

	err := CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets initial failed: %v", err)
	}

	destPath := filepath.Join(outputDir, "assets", "mock.txt")
	destStat1, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat dest file: %v", err)
	}

	if !destStat1.ModTime().Equal(initialTime) {
		t.Errorf("Expected mod time %v, got %v", initialTime, destStat1.ModTime())
	}

	err = CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets unchanged failed: %v", err)
	}

	destStat2, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat dest file: %v", err)
	}
	if !destStat1.ModTime().Equal(destStat2.ModTime()) {
		t.Errorf("Mod time changed unexpectedly, expected %v, got %v", destStat1.ModTime(), destStat2.ModTime())
	}

	_ = os.WriteFile(mockAssetPath, []byte("updated content"), 0600)
	updatedTime := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
	_ = os.Chtimes(mockAssetPath, updatedTime, updatedTime)

	err = CopyAssets(cfg)
	if err != nil {
		t.Fatalf("CopyAssets updated failed: %v", err)
	}

	destStat3, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat dest file: %v", err)
	}
	if !destStat3.ModTime().Equal(updatedTime) {
		t.Errorf("Expected updated mod time %v, got %v", updatedTime, destStat3.ModTime())
	}
}
