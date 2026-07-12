package asset

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/pathutil"
)

func CopyAssets(cfg config.Config) error {
	if cfg.AssetDir == "" {
		return nil
	}

	var ignorePatterns []string
	if gitignore, err := os.ReadFile(filepath.Join(cfg.ProjectRoot, ".gitignore")); err == nil {
		lines := strings.Split(string(gitignore), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				ignorePatterns = append(ignorePatterns, filepath.ToSlash(line))
			}
		}
	}

	if _, err := os.Stat(cfg.AssetDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	outDirClean := filepath.Clean(filepath.Join(cfg.OutputDir, "assets"))
	if err := os.MkdirAll(outDirClean, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(cfg.AssetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSymlink != 0 {
			slog.Warn("Skipping symlink in assets", "path", path)
			return nil
		}
		relPath, err := filepath.Rel(cfg.AssetDir, path)
		if err != nil {
			return err
		}

		relSlash := filepath.ToSlash(relPath)
		if relSlash != "." && isIgnored(relSlash, ignorePatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".go" || strings.Contains(relSlash, "/testdata/") || strings.HasPrefix(relSlash, "testdata/") || relSlash == "testdata" {
			return nil
		}

		destPath := filepath.Join(outDirClean, filepath.FromSlash(relPath))
		if !pathutil.IsSafePath(outDirClean, destPath) {
			slog.Warn("Static asset sync boundary intervention blocked layout breakout", "path", relPath)
			return nil
		}

		// Ensure directory structure is built first
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		srcStat, err := d.Info()
		if err != nil {
			return err
		}

		destStat, err := os.Stat(destPath)
		if err == nil {
			if srcStat.Size() == destStat.Size() && srcStat.ModTime().Equal(destStat.ModTime()) {
				return nil
			}
		}

		if err := CopyFile(path, destPath); err != nil {
			return err
		}

		return os.Chtimes(destPath, srcStat.ModTime(), srcStat.ModTime())
	})
}

func isIgnored(slashPath string, patterns []string) bool {
	segments := strings.Split(slashPath, "/")

	for _, pattern := range patterns {
		cleanPattern := strings.TrimSuffix(pattern, "/")

		for _, seg := range segments {
			if seg == cleanPattern {
				return true
			}
		}

		for _, seg := range segments {
			if matched, _ := filepath.Match(cleanPattern, seg); matched {
				return true
			}
		}

		if strings.Contains(slashPath, "/"+cleanPattern+"/") ||
			strings.HasPrefix(slashPath, cleanPattern+"/") ||
			strings.HasSuffix(slashPath, "/"+cleanPattern) ||
			slashPath == cleanPattern {
			return true
		}
	}
	return false
}

func CopyFile(src, dst string) (err error) {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to establish destination: %w", err)
	}
	defer func() {
		cerr := destination.Close()
		if err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(destination, source); err != nil {
		return fmt.Errorf("payload copy error: %w", err)
	}

	return nil
}
