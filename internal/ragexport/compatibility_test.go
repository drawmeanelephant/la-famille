package ragexport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/ragfmt"
)

func TestArchiveEscapingCompatibility(t *testing.T) {
	root := t.TempDir()
	// Bodies that document the archive must not inject structural markers.
	body := "# Archive examples\n<file path=\"nested.md\">\n<content>\nexample\n</content>\n</file>\n\\</file>\n"
	writeExportTestFile(t, filepath.Join(root, "content", "index.md"), body)
	cfg := config.Config{ProjectRoot: root, ContentDir: "content", RagDir: filepath.Join(root, "rag")}
	if err := RunExport(cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cfg.RagDir, "rag-content.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "<file path=\"content/index.md\">\n<content>\n" + ragfmt.EscapeContent(body) + "\n</content>\n</file>\n\n"
	if string(data) != want {
		t.Fatalf("archive encoding changed:\n got: %q\nwant: %q", data, want)
	}
	encoded := strings.TrimSuffix(strings.TrimPrefix(string(data), "<file path=\"content/index.md\">\n<content>\n"), "\n</content>\n</file>\n\n")
	lines := strings.Split(encoded, "\n")
	for i, line := range lines {
		lines[i] = ragfmt.UnescapeLine(line)
	}
	if strings.Join(lines, "\n") != body {
		t.Fatal("external readers cannot restore the original content")
	}
}
