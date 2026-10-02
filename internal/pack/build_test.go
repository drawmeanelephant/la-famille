package pack

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func buildInputs(t *testing.T) BuildOptions {
	t.Helper()
	root := t.TempDir()
	options := BuildOptions{
		OutputDir: filepath.Join(root, "public"), RagDir: filepath.Join(root, "rag"),
		Site: Site{Name: "Test"}, Provenance: Provenance{Generator: "la-famille", Version: "dev"},
	}
	for _, dir := range []string{options.OutputDir, options.RagDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"backlinks.json", "graph.json", "meta.json", "search.json", "site-manifest.json"} {
		writeInput(t, filepath.Join(options.OutputDir, name), []byte("{}"))
	}
	writeInput(t, filepath.Join(options.RagDir, "rag-content.md"), []byte("content"))
	return options
}

func writeInput(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAllowlistAndDeterminism(t *testing.T) {
	options := buildInputs(t)
	for _, name := range []string{"graph/data.json", "tags/index.html", "tags/pottery/index.html", "categories/index.html", "categories/guide/index.html"} {
		writeInput(t, filepath.Join(options.OutputDir, name), []byte(name))
	}
	for _, name := range []string{
		".env", ".la-famille-cache.json", "index.html", "graph/index.html",
		"diff.json", "diff.txt", "content/page.md", ".github/workflows/ci.yml",
		"tags/secret.txt", "categories/secret.json", "rag-system.md", "rag-config.md",
	} {
		writeInput(t, filepath.Join(options.OutputDir, name), []byte("excluded"))
	}
	for _, name := range []string{"rag-system.md", "rag-config.md", ".env", "source.go"} {
		writeInput(t, filepath.Join(options.RagDir, name), []byte("excluded"))
	}
	first := filepath.Join(t.TempDir(), "first.tar")
	m, err := Build(options, first)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, member := range m.Members {
		paths = append(paths, member.Path)
	}
	want := []string{"backlinks.json", "categories/guide/index.html", "categories/index.html", "graph.json", "graph/data.json",
		"meta.json", "rag-content.md", "search.json", "site-manifest.json", "tags/index.html", "tags/pottery/index.html"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	verified, err := VerifyFile(first)
	if err != nil || !reflect.DeepEqual(verified, m) {
		t.Fatalf("verified = %+v, error = %v", verified, err)
	}
	// Filesystem metadata and destination names are not pack inputs.
	for _, name := range paths {
		dir := options.OutputDir
		if name == "rag-content.md" {
			dir = options.RagDir
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.Chtimes(target, time.Now(), time.Now()); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(target, 0644); err != nil {
			t.Fatal(err)
		}
	}
	second := filepath.Join(t.TempDir(), "second.tar")
	if _, err := Build(options, second); err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("identical inputs did not produce byte-identical packs")
	}
	reader := tar.NewReader(bytes.NewReader(a))
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Format != tar.FormatUSTAR || header.Mode != 0644 || header.Uid != 0 || header.Gid != 0 ||
			header.Uname != "" || header.Gname != "" || header.ModTime.Unix() != 0 {
			t.Fatalf("non-canonical header: %+v", header)
		}
	}
}

func TestBuildRefusesExistingDestination(t *testing.T) {
	options := buildInputs(t)
	destination := filepath.Join(t.TempDir(), "existing")
	writeInput(t, destination, []byte("keep"))
	if _, err := Build(options, destination); err == nil {
		t.Fatal("overwrote existing destination")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing destination changed: %s, %v", data, err)
	}
}

func TestBuildMissingAndNonregularInputs(t *testing.T) {
	tests := []struct {
		name   string
		change func(BuildOptions)
		want   string
	}{
		{"missing graph", func(o BuildOptions) {
			if err := os.Remove(filepath.Join(o.OutputDir, "graph.json")); err != nil {
				t.Fatal(err)
			}
		}, "graph.json"},
		{"missing rag", func(o BuildOptions) {
			if err := os.Remove(filepath.Join(o.RagDir, "rag-content.md")); err != nil {
				t.Fatal(err)
			}
		}, "rag-content.md"},
		{"symlink file", func(o BuildOptions) {
			name := filepath.Join(o.OutputDir, "graph.json")
			if err := os.Remove(name); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("meta.json", name); err != nil {
				t.Fatal(err)
			}
		}, "non-regular"},
		{"symlink taxonomy", func(o BuildOptions) {
			if err := os.Symlink(o.RagDir, filepath.Join(o.OutputDir, "tags")); err != nil {
				t.Fatal(err)
			}
		}, "non-regular taxonomy"},
		{"symlink graph parent", func(o BuildOptions) {
			if err := os.Symlink(o.RagDir, filepath.Join(o.OutputDir, "graph")); err != nil {
				t.Fatal(err)
			}
		}, "symlink parent"},
		{"directory member", func(o BuildOptions) {
			name := filepath.Join(o.OutputDir, "graph.json")
			if err := os.Remove(name); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(name, 0755); err != nil {
				t.Fatal(err)
			}
		}, "non-regular"},
		{"oversized", func(o BuildOptions) {
			f, err := os.OpenFile(filepath.Join(o.OutputDir, "graph.json"), os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.Truncate(MaxMemberSize + 1); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}, "at most"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := buildInputs(t)
			tt.change(options)
			destination := filepath.Join(t.TempDir(), "pack.tar")
			if _, err := Build(options, destination); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Build error = %v, want %q", err, tt.want)
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("failed build left a pack: %v", err)
			}
		})
	}
}

func TestBuildSourceChangesAndWriteErrors(t *testing.T) {
	options := buildInputs(t)
	root, err := os.OpenRoot(options.OutputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	src := source{root: root, path: "graph.json"}
	member, err := fingerprint(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"[]", "", "longer"} {
		t.Run(content, func(t *testing.T) {
			writeInput(t, filepath.Join(options.OutputDir, src.path), []byte(content))
			var output bytes.Buffer
			err := writeArchive(&output, []byte("{}"), []Member{member}, []source{src})
			if err == nil {
				t.Fatal("changed source was accepted")
			}
		})
	}
	if err := writeArchive(failingWriter{}, []byte("{}"), nil, nil); err == nil {
		t.Fatal("write error was ignored")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
