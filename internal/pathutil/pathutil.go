package pathutil

import (
	"path/filepath"
	"strings"
)

// IsSafePath checks if the target path is safe to use and not escaping the base directory.
func IsSafePath(base, target string) bool {
	return strings.HasPrefix(target, base+string(filepath.Separator)) || target == base
}
