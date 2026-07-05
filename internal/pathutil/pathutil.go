package pathutil

import (
	"path/filepath"
	"strings"
)

// IsSafePath checks if the given target path is safely within the base directory.
func IsSafePath(base, target string) bool {
	cleanBase := filepath.Clean(base)
	cleanTarget := filepath.Clean(target)

	if cleanTarget == cleanBase {
		return true
	}
	return strings.HasPrefix(cleanTarget, cleanBase+string(filepath.Separator))
}
