package pack

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

type BuildOptions struct {
	OutputDir  string
	RagDir     string
	Site       Site
	Provenance Provenance
}

type source struct {
	root *os.Root
	path string
}

// Build packages existing outputs, never regenerating them or scanning source.
// The destination must not exist. A failed write removes only the file created
// by this invocation.
func Build(options BuildOptions, destination string) (Manifest, error) {
	output, err := os.OpenRoot(options.OutputDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("open public output: %w", err)
	}
	defer output.Close()
	rag, err := os.OpenRoot(options.RagDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("open RAG output: %w", err)
	}
	defer rag.Close()
	sources, err := collectSources(output, rag)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion, Site: options.Site, Provenance: options.Provenance,
		Members: make([]Member, 0, len(sources)),
	}
	var total int64
	for _, src := range sources {
		member, err := fingerprint(src)
		if err != nil {
			return Manifest{}, err
		}
		total += member.Size
		if total > MaxTotalSize {
			return Manifest{}, fmt.Errorf("payload exceeds total size bound %d", MaxTotalSize)
		}
		manifest.Members = append(manifest.Members, member)
	}
	manifest.ContentRoot = contentRoot(manifest.Members)
	if err := manifest.validate(); err != nil {
		return Manifest{}, err
	}
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return Manifest{}, err
	}
	if data.Len() > MaxManifestSize {
		return Manifest{}, fmt.Errorf("manifest exceeds %d bytes", MaxManifestSize)
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return Manifest{}, fmt.Errorf("create pack (destination must not exist): %w", err)
	}
	writeErr := writeArchive(f, data.Bytes(), manifest.Members, sources)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(destination)
		if writeErr != nil {
			return Manifest{}, writeErr
		}
		return Manifest{}, closeErr
	}
	return manifest, nil
}

func collectSources(output, rag *os.Root) ([]source, error) {
	sources := []source{{root: rag, path: "rag-content.md"}}
	for _, name := range []string{"backlinks.json", "graph.json", "meta.json", "search.json", sitedata.ManifestFileName} {
		sources = append(sources, source{root: output, path: name})
	}
	if _, err := output.Lstat("graph"); err == nil {
		if err := regularParents(output, "graph/data.json"); err != nil {
			return nil, err
		}
		if _, err := output.Lstat("graph/data.json"); err == nil {
			sources = append(sources, source{root: output, path: "graph/data.json"})
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	// Taxonomy output consists of index.html files, not a JSON index.
	entries := 0
	for _, group := range []string{"tags", "categories"} {
		info, err := output.Lstat(group)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("non-regular taxonomy path %q", group)
		}
		err = fs.WalkDir(output.FS(), group, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				if name == group && os.IsNotExist(err) {
					return nil
				}
				return err
			}
			entries++
			if entries > MaxMembers*4 {
				return fmt.Errorf("taxonomy scan exceeds %d entries", MaxMembers*4)
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("non-regular taxonomy path %q", name)
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(name, "/index.html") {
				sources = append(sources, source{root: output, path: name})
				if len(sources) > MaxMembers {
					return fmt.Errorf("payload exceeds %d members", MaxMembers)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	return sources, nil
}

func regularParents(root *os.Root, name string) error {
	parts := strings.Split(name, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		info, err := root.Lstat(parent)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("non-directory or symlink parent %q", parent)
		}
	}
	return nil
}

func openSource(src source) (*os.File, error) {
	if err := validatePath(src.path); err != nil {
		return nil, err
	}
	if err := regularParents(src.root, src.path); err != nil {
		return nil, err
	}
	info, err := src.root.Lstat(src.path)
	if err != nil {
		return nil, fmt.Errorf("read payload %q (run build and rag first): %w", src.path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("non-regular payload member %q", src.path)
	}
	f, err := src.root.Open(src.path)
	if err != nil {
		return nil, err
	}
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxMemberSize {
		_ = f.Close()
		return nil, fmt.Errorf("payload member %q must be regular and at most %d bytes", src.path, MaxMemberSize)
	}
	return f, nil
}

func fingerprint(src source) (Member, error) {
	f, err := openSource(src)
	if err != nil {
		return Member{}, err
	}
	defer f.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(f, MaxMemberSize+1))
	if err != nil {
		return Member{}, err
	}
	if size > MaxMemberSize {
		return Member{}, fmt.Errorf("payload member %q exceeds %d bytes", src.path, MaxMemberSize)
	}
	return Member{Path: src.path, Size: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

func writeHeader(writer *tar.Writer, name string, size int64) error {
	return writer.WriteHeader(&tar.Header{
		Name: name, Size: size, Mode: 0644, Typeflag: tar.TypeReg,
		ModTime: time.Unix(0, 0).UTC(), Format: tar.FormatUSTAR,
	})
}

func writeArchive(output io.Writer, data []byte, members []Member, sources []source) error {
	writer := tar.NewWriter(output)
	if err := writeHeader(writer, ManifestName, int64(len(data))); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	for i, member := range members {
		if err := writeHeader(writer, member.Path, member.Size); err != nil {
			return err
		}
		f, err := openSource(sources[i])
		if err != nil {
			return err
		}
		hash := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(writer, hash), io.LimitReader(f, member.Size+1))
		closeErr := f.Close()
		if copyErr != nil {
			return fmt.Errorf("write member %q: %w", member.Path, copyErr)
		}
		if closeErr != nil {
			return closeErr
		}
		if size != member.Size || fmt.Sprintf("%x", hash.Sum(nil)) != member.SHA256 {
			return fmt.Errorf("payload member %q changed during build", member.Path)
		}
	}
	return writer.Close()
}
