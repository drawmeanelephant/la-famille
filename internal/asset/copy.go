package asset

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
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
					_ = ignorePatterns // use variable
				}
			}
		}

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
				if _, lookErr := exec.LookPath("git"); lookErr != nil {
					log.Printf("Warning: git binary not found in environment, bypassing check-ignore optimization pass")
				} else {
					cmd := exec.Command("git", "check-ignore", "--stdin")
					projectRoot, _ := filepath.Abs(".")
					cmd.Dir = projectRoot
					cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
					out, err := cmd.Output()
					if err != nil {
						var exitErr *exec.ExitError
						if errors.As(err, &exitErr) {
							// exit code 1 means none of the paths are ignored, which is a normal case
							// exit code 128 means outside repository, which happens in tests
							if exitErr.ExitCode() != 1 && exitErr.ExitCode() != 128 {
								log.Printf("Error running git check-ignore: %v (stderr: %q)", err, string(exitErr.Stderr))
							}
						} else {
							log.Printf("Error running git check-ignore: %v", err)
						}
					}

					if len(out) > 0 {
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

				outDirClean := filepath.Clean(filepath.Join(cfg.OutputDir, "assets"))
				destPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
				if !strings.HasPrefix(destPath, outDirClean+string(filepath.Separator)) && destPath != outDirClean {
					log.Printf("Warning: Potential path traversal in asset copying detected: %s. Skipping.", relPath)
					continue
				}
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
