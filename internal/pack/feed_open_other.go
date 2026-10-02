//go:build !unix

package pack

import "os"

// Platforms without Unix FIFOs retain confined opening and the caller's
// link, regular-file, size, and identity checks.
func openFeedLeaf(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}
