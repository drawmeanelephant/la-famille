package pathutil

import (
	"path/filepath"
	"strings"
)

func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

// IsPathWithin reports whether targetPath resides within baseDir after
// resolving both to absolute paths and evaluating symlinks. Equal
// paths or empty inputs are not considered "within".
func IsPathWithin(baseDir, targetPath string) bool {
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(targetPath) == "" {
		return false
	}
	base, err := resolvePath(baseDir)
	if err != nil {
		return false
	}
	target, err := resolvePath(targetPath)
	if err != nil {
		return false
	}
	if base == target {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	relSlash := filepath.ToSlash(rel)
	return relSlash != ".." && !strings.HasPrefix(relSlash, "../") && !filepath.IsAbs(rel)
}

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
