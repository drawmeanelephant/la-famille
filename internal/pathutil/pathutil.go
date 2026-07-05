package pathutil

import (
	"path/filepath"
)

// IsSafePath checks if the target path is safely contained within the base path,
// preventing path traversal attacks.
func IsSafePath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return filepath.IsLocal(rel)
}
