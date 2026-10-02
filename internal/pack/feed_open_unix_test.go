//go:build darwin || linux

package pack

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestFeedLeafRejectsLinksAndDoesNotWaitForFIFO(t *testing.T) {
	dir := t.TempDir()
	writeInput(t, filepath.Join(dir, "regular"), []byte("verified source"))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Symlink("regular", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if f, err := openFeedSource(root, "link", MaxArchiveSize); err == nil {
		_ = f.Close()
		t.Fatal("source open followed a symlink")
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		f, err := openFeedLeaf(root, "fifo")
		if f != nil {
			_ = f.Close()
		}
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		// Unblock a regressed blocking open before reporting the failure.
		writer, err := os.OpenFile(fifo, os.O_RDWR|syscall.O_NONBLOCK, 0)
		if err == nil {
			defer writer.Close()
		}
		<-done
		t.Fatal("leaf open waited for a FIFO writer")
	}
	if _, err := openFeedSource(root, "fifo", MaxArchiveSize); err == nil {
		t.Fatal("FIFO accepted as a feed source")
	}
}
