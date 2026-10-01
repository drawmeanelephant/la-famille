package diff

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func buildRevision(root, revision string, builder func(string) (sitedata.Manifest, error)) (sitedata.Manifest, error) {
	top, err := runGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return sitedata.Manifest{}, err
	}
	data, err := runGit(strings.TrimSpace(string(top)), "archive", "--format=tar", revision)
	if err != nil {
		return sitedata.Manifest{}, err
	}
	prefix, err := runGit(root, "rev-parse", "--show-prefix")
	if err != nil {
		return sitedata.Manifest{}, err
	}
	dir, err := os.MkdirTemp("", "la-famille-diff-")
	if err != nil {
		return sitedata.Manifest{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := extractArchive(dir, data); err != nil {
		return sitedata.Manifest{}, err
	}
	return builder(filepath.Join(dir, strings.TrimSpace(string(prefix))))
}

func extractArchive(root string, data []byte) error {
	reader := tar.NewReader(bytes.NewReader(data))
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.FromSlash(header.Name)
		if !filepath.IsLocal(name) {
			return fmt.Errorf("unsafe Git archive path %q", header.Name)
		}
		path := filepath.Join(root, name)
		switch header.Typeflag {
		case tar.TypeXGlobalHeader:
			// git archive records the commit ID in a global PAX header.
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(f, reader, header.Size)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			// Do not allow archived links to escape the isolated snapshot.
			return fmt.Errorf("unsupported Git archive entry %q (type %d); use built manifests for symlinked sites", header.Name, header.Typeflag)
		}
	}
}
