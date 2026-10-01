package diff

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tbuddy/la-famille/internal/sitedata"
)

func TestExtractArchiveRejectsUnsafeEntries(t *testing.T) {
	for _, header := range []*tar.Header{
		{Name: "../escape", Typeflag: tar.TypeReg, Size: 0},
		{Name: "/absolute", Typeflag: tar.TypeReg, Size: 0},
		{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/tmp"},
		{Name: "hard", Typeflag: tar.TypeLink, Linkname: "../other"},
	} {
		t.Run(header.Name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := tar.NewWriter(&buffer)
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if extractArchive(t.TempDir(), buffer.Bytes()) == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	if err := writer.WriteHeader(&tar.Header{Name: "content/page.md", Typeflag: tar.TypeReg, Size: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := extractArchive(root, buffer.Bytes()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "content", "page.md"))
	if err != nil || string(data) != "body" {
		t.Fatalf("archive data = %q, %v", data, err)
	}
}

func TestBuildRevisionPreservesNestedProjectAndCleansArchive(t *testing.T) {
	top := t.TempDir()
	root := filepath.Join(top, "sites", "sub")
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	if err := writer.WriteHeader(&tar.Header{Name: "sites/sub/content/page.md", Typeflag: tar.TypeReg, Size: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	original := runGit
	t.Cleanup(func() { runGit = original })
	runGit = func(dir string, args ...string) ([]byte, error) {
		if args[0] == "archive" {
			if dir != top {
				t.Fatal("archive did not run from Git root")
			}
			return buffer.Bytes(), nil
		}
		if dir != root {
			t.Fatal("project query did not run from project root")
		}
		if args[1] == "--show-toplevel" {
			return []byte(top + "\n"), nil
		}
		return []byte("sites/sub/\n"), nil
	}
	var extracted string
	_, err := buildRevision(root, "commit", func(dir string) (sitedata.Manifest, error) {
		extracted = dir
		data, err := os.ReadFile(filepath.Join(dir, "content", "page.md"))
		if err != nil || string(data) != "body" {
			t.Fatalf("nested snapshot = %q, %v", data, err)
		}
		return sitedata.Manifest{Version: 2}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(extracted); !os.IsNotExist(err) {
		t.Fatal("temporary archive was not cleaned")
	}
}
