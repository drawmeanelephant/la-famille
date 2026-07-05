package pathutil

import (
	"path/filepath"
	"strings"
)

// IsSafePath checks if targetPath resides lexically within baseDir.
// Resolving both paths to absolute paths ensures consistent drive-letter casing
// and absolute vs. relative uniformity across all platforms (such as Windows C: vs c:).
func IsSafePath(baseDir, targetPath string) bool {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(baseAbs, targetAbs)
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
