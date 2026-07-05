package pathutil

import (
	"path/filepath"
	"strings"
)

// IsSafePath checks if targetPath resides lexically within baseDir.
// It handles volume casing issues on Windows and prevents relative-path breakout attacks.
func IsSafePath(baseDir, targetPath string) bool {
	baseClean := filepath.Clean(baseDir)
	targetClean := filepath.Clean(targetPath)

	rel, err := filepath.Rel(baseClean, targetClean)
	if err != nil {
		return false
	}

	// Normalize separators to a unified forward slash for consistent checks
	relSlash := filepath.ToSlash(rel)

	// If the relative path escapes the directory tree, it is unsafe.
	if relSlash == ".." || strings.HasPrefix(relSlash, "../") {
		return false
	}

	return true
}
