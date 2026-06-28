package transform

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

func TestLinkTransformer(t *testing.T) {
	renderTrue := true
	renderFalse := false

	tests := []struct {
		name         string
		currentFile  string
		markdown     string
		fileMap      map[string]*content.FileMeta
		expectedHTML string
		expectedMiss map[string][]string
	}{
		{
			name:        "internal link rewritten",
			currentFile: "index.md",
			markdown:    "[Link](page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"page/\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "external link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](http://example.com/page.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"http://example.com/page.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "non-markdown link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](page.txt)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"page.txt\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "link to unrendered md file kept as .md",
			currentFile: "index.md",
			markdown:    "[Link](raw.md)",
			fileMap: map[string]*content.FileMeta{
				"raw.md": {Render: &renderFalse},
			},
			expectedHTML: "<p><a href=\"raw.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "missing link rewritten and tracked",
			currentFile:  "index.md",
			markdown:     "[Link](missing.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"missing/\">Link</a></p>\n",
			expectedMiss: map[string][]string{
				"missing.md": {"index.md"},
			},
		},
		{
			name:        "relative subdirectory link",
			currentFile: "sub/index.md",
			markdown:    "[Link](../page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"../page/\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "path traversal link ignored",
			currentFile:  "index.md",
			markdown:     "[Link](../../../etc/passwd.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"../../../etc/passwd.md\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "multiple identical missing links deduplicate parent",
			currentFile:  "index.md",
			markdown:     "[Link](missing.md) and [Link2](missing.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"missing/\">Link</a> and <a href=\"missing/\">Link2</a></p>\n",
			expectedMiss: map[string][]string{
				"missing.md": {"index.md"},
			},
		},
		{
			name:         "empty target path ignored",
			currentFile:  "index.md",
			markdown:     "[Link](#test)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"#test\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			missingFiles := make(map[string][]string)
			backlinks := make(map[string][]string)
			g := &graph.Graph{
				Nodes: make(map[string]graph.Node),
				Edges: [][2]string{},
			}

			transformer := &LinkTransformer{
				CurrentFile:  tc.currentFile,
				FileMap:      tc.fileMap,
				MissingFiles: missingFiles,
				Backlinks:    backlinks,
				Graph:        g,
			}

			md := goldmark.New(
				goldmark.WithParserOptions(
					parser.WithASTTransformers(
						util.Prioritized(transformer, 100),
					),
				),
			)

			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.markdown), &buf); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tc.expectedHTML {
				t.Errorf("expected HTML %q, got %q", tc.expectedHTML, buf.String())
			}

			if len(missingFiles) != len(tc.expectedMiss) {
				t.Errorf("expected %d missing files, got %d", len(tc.expectedMiss), len(missingFiles))
			}
			for k, v := range tc.expectedMiss {
				if len(missingFiles[k]) != len(v) {
					t.Errorf("missing file %s: expected %d parents, got %d", k, len(v), len(missingFiles[k]))
				}
			}
		})
	}
}
