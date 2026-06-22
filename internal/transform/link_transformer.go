package transform

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
)

type LinkTransformer struct {
	CurrentFile  string // The current file being processed (e.g., docs/index.md)
	FileMap      map[string]*content.FileMeta
	MissingFiles map[string][]string // map[targetFile]parents
	Backlinks    map[string][]string
	Graph        *graph.Graph
}

func (t *LinkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	sourceId := strings.TrimSuffix(t.CurrentFile, ".md")

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			dest := string(link.Destination)
			u, err := url.Parse(dest)
			// Ignore if parse fails, or it's an absolute url (like http://...), or not a .md file
			if err != nil || u.IsAbs() || !strings.HasSuffix(u.Path, ".md") {
				return ast.WalkContinue, nil
			}

			// Path is relative, like "../file.md" or "file.md"
			// Need to resolve it relative to the directory of CurrentFile
			dir := filepath.Dir(t.CurrentFile)
			// filepath.Join uses OS separators, but we want to stick to slashes
			targetRelPath := filepath.ToSlash(filepath.Clean(dir + "/" + u.Path))
			if dir == "." {
				targetRelPath = filepath.ToSlash(filepath.Clean(u.Path))
			}

			// Prevent path traversal
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) {
				return ast.WalkContinue, nil
			}

			targetId := strings.TrimSuffix(targetRelPath, ".md")
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceId, targetId})
			t.Backlinks[targetId] = append(t.Backlinks[targetId], sourceId)

			// Check file map
			meta, exists := t.FileMap[targetRelPath]
			if exists {
				// if render is explicitly false, it will be a raw .md file, so we leave the link as .md
				if meta.Render != nil && !*meta.Render {
					// keep it as .md, no change needed
				} else {
					// otherwise, it will be rendered to .html
					u.Path = strings.TrimSuffix(u.Path, ".md") + ".html"
					link.Destination = []byte(u.String())
				}
			} else {
				// missing file! rewrite to .html, and record missing file
				u.Path = strings.TrimSuffix(u.Path, ".md") + ".html"
				link.Destination = []byte(u.String())

				// record target as missing so we can generate stub
				parents := t.MissingFiles[targetRelPath]
				found := false
				for _, p := range parents {
					if p == t.CurrentFile {
						found = true
						break
					}
				}
				if !found {
					t.MissingFiles[targetRelPath] = append(parents, t.CurrentFile)
				}
			}
		}

		return ast.WalkContinue, nil
	})
}
