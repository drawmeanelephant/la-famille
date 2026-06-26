package asset

import (
	"fmt"
	"log"
	"io"
	"os"
	"os/exec"
	"strings"
	"path/filepath"

	"github.com/tbuddy/la-famille/internal/config"
)

// CopyAssets copies files from the configured AssetDir to OutputDir/assets,
// skipping testdata directories and checking for path traversal.
func CopyAssets(cfg config.Config) error {
	if cfg.AssetDir != "" {
		if _, err := os.Stat(cfg.AssetDir); err == nil {
			targetDir := filepath.Join(cfg.OutputDir, "assets")
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			var paths []string
			err = filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					paths = append(paths, path)
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Batch check gitignore
			ignoredPaths := make(map[string]bool)
			if len(paths) > 0 {
				cmd := exec.Command("git", "check-ignore", "--stdin")
				cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
				if out, err := cmd.Output(); err == nil || len(out) > 0 {
					lines := strings.Split(strings.TrimSpace(string(out)), "\n")
					for _, line := range lines {
						if line != "" {
							// check-ignore returns absolute or relative paths depending on input. Since we passed relative, it should return relative.
							// let's use the exact string returned to populate the map.
							ignoredPaths[line] = true
						}
					}
				}
			}

			for _, path := range paths {
				if ignoredPaths[path] {
					continue
				}

				if filepath.Ext(path) == ".go" {
					continue
				}

				// Skip testdata in the path
				if strings.Contains(path, "/testdata/") || strings.Contains(path, "\\testdata\\") || strings.HasSuffix(path, "/testdata") || strings.HasSuffix(path, "\\testdata") {
					continue
				}

				relPath, err := filepath.Rel(cfg.AssetDir, path)
				if err != nil {
					return err
				}

				if !filepath.IsLocal(filepath.FromSlash(relPath)) {
					log.Printf("Warning: Potential path traversal in asset copying detected: %s. Skipping.", relPath)
					continue
				}
				destPath := filepath.Join(cfg.OutputDir, "assets", relPath)
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}

				if err := CopyFile(path, destPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CopyFile streams the contents of src to dst using a buffer.
func CopyFile(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		cerr := destination.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(destination, source); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Ensure the write is flushed to disk
	if err = destination.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination: %w", err)
	}

	return nil
}
