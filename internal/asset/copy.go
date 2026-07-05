package asset

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
)

// CopyAssets copies files from the configured AssetDir to OutputDir/assets,
// skipping testdata directories, handling .gitignore patterns natively, and checking for path traversal.
func CopyAssets(cfg config.Config) error {
	if cfg.AssetDir == "" {
		return nil
	}

	// 1. Read and parse local .gitignore patterns natively
	var ignorePatterns []string
	if gitignore, err := os.ReadFile(filepath.Join(cfg.ProjectRoot, ".gitignore")); err == nil {
		lines := strings.Split(string(gitignore), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Convert to unified forward slashes for matching consistency
			ignorePatterns = append(ignorePatterns, filepath.ToSlash(line))
		}
	}

	if _, err := os.Stat(cfg.AssetDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	targetDir := filepath.Join(cfg.OutputDir, "assets")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// 2. Walk directory to gather asset files
	var paths []string
	err := filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
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

	// 3. Process and filter gathered assets
	for _, path := range paths {
		if filepath.Ext(path) == ".go" {
			continue
		}

		// Skip testdata in path structures
		if strings.Contains(path, "/testdata/") || strings.Contains(path, "\\testdata\\") ||
			strings.HasSuffix(path, "/testdata") || strings.HasSuffix(path, "\\testdata") {
			continue
		}

		// Native ignore check
		if isIgnored(path, ignorePatterns) {
			continue
		}

		relPath, err := filepath.Rel(cfg.AssetDir, path)
		if err != nil {
			return err
		}

		outDirClean := filepath.Clean(filepath.Join(cfg.OutputDir, "assets"))
		destPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))

		// Guard against directory escape
		if !strings.HasPrefix(destPath, outDirClean+string(filepath.Separator)) && destPath != outDirClean {
			slog.Warn("Potential path traversal in asset copying detected. Skipping.", "path", relPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		srcStat, err := os.Stat(path)
		if err != nil {
			return err
		}

		destStat, err := os.Stat(destPath)
		if err == nil {
			if srcStat.Size() == destStat.Size() && srcStat.ModTime().Equal(destStat.ModTime()) {
				continue
			}
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := CopyFile(path, destPath); err != nil {
			return err
		}

		if err := os.Chtimes(destPath, srcStat.ModTime(), srcStat.ModTime()); err != nil {
			return err
		}
	}

	return nil
}

// isIgnored evaluates a filepath against parsed .gitignore strings natively.
func isIgnored(path string, patterns []string) bool {
	slashPath := filepath.ToSlash(path)
	segments := strings.Split(slashPath, "/")

	for _, pattern := range patterns {
		cleanPattern := strings.TrimSuffix(pattern, "/")

		// Match individual path segments (exact matches)
		for _, seg := range segments {
			if seg == cleanPattern {
				return true
			}
		}

		// Match basic wildcards (e.g., *.log or temp*)
		for _, seg := range segments {
			if matched, _ := filepath.Match(cleanPattern, seg); matched {
				return true
			}
		}

		// Match absolute containment pathways
		if strings.Contains(slashPath, "/"+cleanPattern+"/") ||
			strings.HasPrefix(slashPath, cleanPattern+"/") ||
			strings.HasSuffix(slashPath, "/"+cleanPattern) ||
			slashPath == cleanPattern {
			return true
		}
	}
	return false
}

// CopyFile streams the contents of src to dst using a buffer.
func CopyFile(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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

	if err = destination.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination: %w", err)
	}

	return nil
}
