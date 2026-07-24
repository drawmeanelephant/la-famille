package retrieval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/ragfmt"
)

// TestParseBundleWithEmbeddedArchiveMarkers is the regression for the broken
// `la-famille rag` -> `la-famille ask` quickstart.
//
// The archive bundles the project's own Go sources and everything under
// content/. Any file that documents the archive format — a parser test, or a
// Markdown page explaining RAG export — used to inject real structure into the
// bundle, corrupting it so the whole archive failed to parse and `ask` refused
// to start.
func TestParseBundleWithEmbeddedArchiveMarkers(t *testing.T) {
	// A source file whose body contains a complete, well-formed archive block.
	// This is exactly the shape of internal/retrieval/chunker_test.go.
	hostile := strings.Join([]string{
		"package retrieval",
		"",
		"func TestChunker(t *testing.T) {",
		"\tbundle := `<file path=\"content/pub.md\">",
		"<content>",
		"# Pub",
		"</content>",
		"</file>",
		"`",
		"}",
	}, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "rag-system.md")

	// Write the bundle the way ragexport does, escaping the body.
	var b strings.Builder
	b.WriteString("<file path=\"internal/retrieval/chunker_test.go\">\n<content>\n")
	b.WriteString(ragfmt.EscapeContent(hostile))
	b.WriteString("\n</content>\n</file>\n\n")
	b.WriteString("<file path=\"internal/second.go\">\n<content>\npackage second\n</content>\n</file>\n\n")

	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	got, err := parseRAGBundle(path)
	if err != nil {
		t.Fatalf("parseRAGBundle returned an error on an escaped bundle: %v", err)
	}
	if len(got.files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(got.files))
	}
	if got.files[0].path != "internal/retrieval/chunker_test.go" {
		t.Fatalf("unexpected first path %q", got.files[0].path)
	}
	// The hostile body must come back byte-for-byte.
	if got.files[0].text != hostile {
		t.Fatalf("body was not restored verbatim:\n got: %q\nwant: %q", got.files[0].text, hostile)
	}
	// The file after it must still be found — the old bug swallowed everything
	// downstream of the first poisoned block.
	if got.files[1].path != "internal/second.go" {
		t.Fatalf("second file lost, got %q", got.files[1].path)
	}
}

// TestParseBundleUnescapedMarkersStillFailLoudly documents that an archive
// written by an older, non-escaping writer is still rejected rather than
// silently mis-chunked.
func TestParseBundleUnescapedMarkersStillFailLoudly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rag-system.md")
	legacy := "<file path=\"a.go\">\n<content>\n</content>\n</file>\n<content>\n</content>\n</file>\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if _, err := parseRAGBundle(path); err == nil {
		t.Fatal("expected an error for an unescaped legacy bundle, got nil")
	}
}
