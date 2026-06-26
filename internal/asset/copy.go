package asset

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

// CopyAssets copies files from the configured AssetDir to OutputDir/assets,
// skipping testdata directories and checking for path traversal.
func CopyAssets(cfg config.Config) error {
	if cfg.AssetDir != "" {
		ignorePatterns := []string{}
		if gitignore, err := os.ReadFile(".gitignore"); err == nil {
			lines := strings.Split(string(gitignore), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					// Normalize pattern for filepath.Match
					if strings.HasSuffix(line, "/") {
						line = strings.TrimSuffix(line, "/")
					}
					if strings.HasPrefix(line, "/") {
						line = strings.TrimPrefix(line, "/")
					}
					ignorePatterns = append(ignorePatterns, line)
				}
			}
		}

		if _, err := os.Stat(cfg.AssetDir); err == nil {
			err = filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip testdata subdirectories
				if d.IsDir() && d.Name() == "testdata" {
					return filepath.SkipDir
				}

				if filepath.Ext(path) == ".go" {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Check gitignore patterns
				for _, pattern := range ignorePatterns {
					matched, _ := filepath.Match(pattern, d.Name())
					if matched || d.Name() == pattern {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					// Also check if relative path matches
					rel, _ := filepath.Rel(".", path)
					matchedRel, _ := filepath.Match(pattern, rel)
					if matchedRel || rel == pattern || strings.HasPrefix(rel, pattern+"/") {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}

				if d.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(cfg.AssetDir, path)
				if err != nil {
					return err
				}

				if !filepath.IsLocal(filepath.FromSlash(relPath)) {
					log.Printf("Warning: Potential path traversal in asset copying detected: %s. Skipping.", relPath)
					return nil
				}
				destPath := filepath.Join(cfg.OutputDir, "assets", relPath)
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return err
				}

				input, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				return os.WriteFile(destPath, input, 0644)
			})
			if err != nil {
				return fmt.Errorf("failed to copy assets: %w", err)
			}
		}
	}
	return nil
}
