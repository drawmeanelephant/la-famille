package pack

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// VerifyFile inspects a local pack without extracting or executing any member.
func VerifyFile(name string) (Manifest, error) {
	f, err := os.Open(name)
	if err != nil {
		return Manifest{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Manifest{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxArchiveSize {
		return Manifest{}, fmt.Errorf("pack must be a regular file of at most %d bytes", MaxArchiveSize)
	}
	return Verify(f)
}

// Verify streams a strictly bounded USTAR archive. Header blocks are parsed
// individually so archive/tar cannot consume hidden PAX/GNU extension records
// before their type and size have been checked.
func Verify(input io.Reader) (Manifest, error) {
	r := io.LimitReader(input, MaxArchiveSize+1)
	header, err := nextHeader(r)
	if err != nil {
		return Manifest{}, fmt.Errorf("read pack manifest header: %w", err)
	}
	if header.Name != ManifestName || header.Size > MaxManifestSize {
		return Manifest{}, fmt.Errorf("first member must be %s, at most %d bytes", ManifestName, MaxManifestSize)
	}
	data, err := io.ReadAll(io.LimitReader(r, header.Size))
	if err != nil {
		return Manifest{}, err
	}
	if int64(len(data)) != header.Size {
		return Manifest{}, fmt.Errorf("truncated manifest")
	}
	if err := readPadding(r, header.Size); err != nil {
		return Manifest{}, err
	}
	manifest, err := parseManifest(data)
	if err != nil {
		return Manifest{}, err
	}
	listed := make(map[string]Member, len(manifest.Members))
	for _, member := range manifest.Members {
		listed[member.Path] = member
	}
	seen := map[string]bool{ManifestName: true}
	for {
		header, err := nextHeader(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("read pack member header: %w", err)
		}
		if seen[header.Name] {
			return Manifest{}, fmt.Errorf("duplicate archive member %q", header.Name)
		}
		seen[header.Name] = true
		member, ok := listed[header.Name]
		if !ok {
			return Manifest{}, fmt.Errorf("unlisted archive member %q", header.Name)
		}
		if header.Size != member.Size {
			return Manifest{}, fmt.Errorf("member %q size mismatch: expected %d, actual %d", member.Path, member.Size, header.Size)
		}
		hash := sha256.New()
		size, err := io.Copy(hash, io.LimitReader(r, header.Size))
		if err != nil {
			return Manifest{}, fmt.Errorf("read member %q: %w", member.Path, err)
		}
		if size != member.Size {
			return Manifest{}, fmt.Errorf("member %q size mismatch: expected %d, actual %d (truncated)", member.Path, member.Size, size)
		}
		actual := fmt.Sprintf("%x", hash.Sum(nil))
		if actual != member.SHA256 {
			return Manifest{}, fmt.Errorf("member %q SHA256 mismatch: expected %s, actual %s", member.Path, member.SHA256, actual)
		}
		if err := readPadding(r, size); err != nil {
			return Manifest{}, fmt.Errorf("member %q: %w", member.Path, err)
		}
	}
	for _, member := range manifest.Members {
		if !seen[member.Path] {
			return Manifest{}, fmt.Errorf("missing archive member %q", member.Path)
		}
	}
	return manifest, nil
}

func nextHeader(r io.Reader) (*tar.Header, error) {
	var block [512]byte
	if _, err := io.ReadFull(r, block[:]); err != nil {
		return nil, fmt.Errorf("truncated archive header: %w", err)
	}
	if allZero(block[:]) {
		if _, err := io.ReadFull(r, block[:]); err != nil || !allZero(block[:]) {
			return nil, fmt.Errorf("archive requires two zero end blocks")
		}
		var extra [1]byte
		if n, err := r.Read(extra[:]); n != 0 || err != io.EOF {
			return nil, fmt.Errorf("trailing archive data")
		}
		return nil, io.EOF
	}
	if block[156] != tar.TypeReg || string(block[257:263]) != "ustar\x00" {
		return nil, fmt.Errorf("only regular USTAR members are supported")
	}
	header, err := tar.NewReader(bytes.NewReader(block[:])).Next()
	if err != nil {
		return nil, err
	}
	if header.Format != tar.FormatUSTAR || header.Linkname != "" || header.Size < 0 {
		return nil, fmt.Errorf("invalid regular USTAR header")
	}
	if err := validatePath(header.Name); err != nil {
		return nil, err
	}
	return header, nil
}

func readPadding(r io.Reader, size int64) error {
	var padding [512]byte
	n := (512 - size%512) % 512
	if _, err := io.ReadFull(r, padding[:n]); err != nil {
		return fmt.Errorf("truncated member padding: %w", err)
	}
	if !allZero(padding[:n]) {
		return fmt.Errorf("nonzero member padding")
	}
	return nil
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}
