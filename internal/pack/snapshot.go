package pack

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// snapshot retains an opaque archive, not extracted members. Offsets point
// into the exact stream verified while copying into this private open file.
type snapshot struct {
	file         *os.File
	hash         string
	manifest     Manifest
	manifestJSON []byte
	offsets      map[string]int64
}

func (s *snapshot) close() {
	_ = s.file.Close()
	_ = os.Remove(s.file.Name())
}

func (s *snapshot) member(m Member) io.Reader {
	return io.NewSectionReader(s.file, s.offsets[m.Path], m.Size)
}

func captureSnapshot(name string, verify func(io.Reader, *snapshot) error) (*snapshot, error) {
	input, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxArchiveSize {
		return nil, fmt.Errorf("archive must be a regular file of at most %d bytes", MaxArchiveSize)
	}
	f, err := os.CreateTemp("", "la-famille-pack-snapshot-*")
	if err != nil {
		return nil, err
	}
	s := &snapshot{file: f}
	hash := sha256.New()
	reader := io.TeeReader(io.LimitReader(input, MaxArchiveSize+1), io.MultiWriter(f, hash))
	if err := verify(reader, s); err != nil {
		s.close()
		return nil, err
	}
	s.hash = fmt.Sprintf("%x", hash.Sum(nil))
	return s, nil
}

func loadPack(name string) (*snapshot, error) {
	return captureSnapshot(name, func(r io.Reader, s *snapshot) error {
		var err error
		s.manifest, s.manifestJSON, s.offsets, err = verifyPack(r)
		return err
	})
}

// publishArchive verifies through the same descriptor before atomically
// publishing with an exclusive hard link. Rename would overwrite a destination
// created concurrently. The temporary file is on the destination filesystem.
func publishArchive(destination string, write func(io.Writer) error, verify func(io.Reader) error) error {
	f, err := os.CreateTemp(filepath.Dir(destination), ".la-famille-pack-output-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	if err := write(f); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := verify(f); err != nil {
		return fmt.Errorf("verify reconstructed archive: %w", err)
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Link(f.Name(), destination); err != nil {
		return fmt.Errorf("create archive (destination must not exist): %w", err)
	}
	return nil
}
