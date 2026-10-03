package pack

import (
	"fmt"
	"os"
	"path/filepath"
)

// Sync both file and parent directory so the atomic pointer survives a restart.
// The old pointer remains intact until the replacement is complete.
func writeAtomicJSON(directory, name string, data []byte) error {
	f, err := os.CreateTemp(directory, ".pack-metadata-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filepath.Join(directory, name)); err != nil {
		return err
	}
	return syncDirectory(directory)
}

func syncDirectory(directory string) error {
	f, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync state directory: %w", err)
	}
	return nil
}
