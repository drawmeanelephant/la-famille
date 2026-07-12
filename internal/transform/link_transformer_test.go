package transform

import (
	"bytes"
	"testing"
	"strings"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/ast"

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

func TestLinkTransformerExtended(t *testing.T) {
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
			name:         "mailto link ignored",
			currentFile:  "index.md",
			markdown:     "[Email](mailto:test@example.com)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"mailto:test@example.com\">Email</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "tel link ignored",
			currentFile:  "index.md",
			markdown:     "[Phone](tel:1234567890)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"tel:1234567890\">Phone</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "javascript link ignored",
			currentFile:  "index.md",
			markdown:     "[JS](javascript:alert(1))",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"\">JS</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "protocol relative link ignored",
			currentFile:  "index.md",
			markdown:     "[Rel](//example.com/page.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"//example.com/page.md\">Rel</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "root relative link processed",
			currentFile: "sub/index.md",
			markdown:    "[Root](/page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"../page/\">Root</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "fragment only link ignored",
			currentFile:  "index.md",
			markdown:     "[Frag](#section)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"#section\">Frag</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:         "percent encoded traversal link ignored",
			currentFile:  "index.md",
			markdown:     "[Trav](%2E%2E%2F%2E%2E%2Fetc%2Fpasswd.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"%2E%2E%2F%2E%2E%2Fetc%2Fpasswd.md\">Trav</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "query and fragment preserved",
			currentFile: "index.md",
			markdown:    "[Link](page.md?view=all#install)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue},
			},
			expectedHTML: "<p><a href=\"page/?view=all#install\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "slug target",
			currentFile: "index.md",
			markdown:    "[Link](page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue, Slug: "custom-slug"},
			},
			expectedHTML: "<p><a href=\"custom-slug/\">Link</a></p>\n",
			expectedMiss: map[string][]string{},
		},
		{
			name:        "invalid slug ignored",
			currentFile: "index.md",
			markdown:    "[Link](page.md)",
			fileMap: map[string]*content.FileMeta{
				"page.md": {Render: &renderTrue, Slug: "../invalid"},
			},
			expectedHTML: "<p><a href=\"page/\">Link</a></p>\n",
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
			name:         "multiple identical missing links deduplicate parent",
			currentFile:  "index.md",
			markdown:     "[Link](missing.md) and [Link2](missing.md)",
			fileMap:      map[string]*content.FileMeta{},
			expectedHTML: "<p><a href=\"missing/\">Link</a> and <a href=\"missing/\">Link2</a></p>\n",
			expectedMiss: map[string][]string{
				"missing.md": {"index.md"},
			},
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

func TestLinkTransformerRenderFalse(t *testing.T) {
	fileMap := map[string]*content.FileMeta{
		"raw.md": {Render: new(bool)}, // false
		"doc.md": {Render: nil},       // implicitly true
	}
	*fileMap["raw.md"].Render = false

	g := &graph.Graph{Nodes: make(map[string]graph.Node)}
	backlinks := make(map[string][]string)
	missing := make(map[string][]string)

	transformer := &LinkTransformer{
		CurrentFile:  "index.md",
		FileMap:      fileMap,
		MissingFiles: missing,
		Backlinks:    backlinks,
		Graph:        g,
	}

	source := []byte(`Link to [raw](raw.md) and [doc](doc.md)`)

	node := parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
	).Parse(text.NewReader(source))

	transformer.Transform(node.(*ast.Document), text.NewReader(source), nil)

	var buf bytes.Buffer
	renderer := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(html.NewRenderer(), 1000)))

	err := renderer.Render(&buf, source, node)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outHTML := buf.String()
	if !strings.Contains(outHTML, `href="raw.md"`) {
		t.Errorf("Expected href=\"raw.md\", got: %s", outHTML)
	}
	if !strings.Contains(outHTML, `href="doc/"`) {
		t.Errorf("Expected href=\"doc/\", got: %s", outHTML)
	}
}
