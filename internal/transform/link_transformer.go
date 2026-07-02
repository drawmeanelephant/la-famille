package transform

import (
	"net/url"
	"path/filepath"
	"strings"
	"sync"

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
	Mu           *sync.Mutex
}

func (t *LinkTransformer) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	sourceID := strings.TrimSuffix(t.CurrentFile, ".md")

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
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

			targetID := strings.TrimSuffix(targetRelPath, ".md")
			if t.Mu != nil {
				t.Mu.Lock()
			}
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceID, targetID})
			t.Backlinks[targetID] = append(t.Backlinks[targetID], sourceID)
			if t.Mu != nil {
				t.Mu.Unlock()
			}

			// Check file map
			meta, exists := t.FileMap[targetRelPath]

			// If target exists and render is explicitly false, keep as .md
			if exists && meta.Render != nil && !*meta.Render {
				// keep it as .md, no change needed
				_ = meta
			} else {
				slug := ""
				if exists && meta != nil {
					slug = meta.Slug
					if slug != "" {
						if !filepath.IsLocal(slug) || strings.Contains(slug, ".") || strings.Contains(slug, string(filepath.Separator)) || strings.Contains(slug, "/") {
							slug = ""
						}
					}
				}

				currOut := GetOutputURL(t.CurrentFile, "")
				targetOut := GetOutputURL(targetRelPath, slug)

				currDir := filepath.Dir(currOut)
				if currDir == "." {
					currDir = ""
				}

				relOut, err := filepath.Rel(currDir, targetOut)
				if err == nil {
					relOutSlash := filepath.ToSlash(relOut)
					if strings.HasSuffix(relOutSlash, "index.html") {
						if relOutSlash == "index.html" {
							relOutSlash = "./"
						} else {
							relOutSlash = strings.TrimSuffix(relOutSlash, "index.html")
						}
					}
					u.Path = relOutSlash
					link.Destination = []byte(u.String())
				}
			}

			if !exists {
				// record target as missing so we can generate stub
				if t.Mu != nil {
					t.Mu.Lock()
				}
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
				if t.Mu != nil {
					t.Mu.Unlock()
				}
			}
		}

		return ast.WalkContinue, nil
	})
}
