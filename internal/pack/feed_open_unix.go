//go:build unix

package pack

import (
	"os"
	"syscall"
)

// Avoid waiting for a writer if a regular source is replaced by a FIFO.
// os.Root resolves links itself; the caller's Lstat and identity checks, not
// O_NOFOLLOW, enforce the feed's link-free source policy.
func openFeedLeaf(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
