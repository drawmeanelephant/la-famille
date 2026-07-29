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
	FileMap      map[string]*content.FileMeta
	MissingFiles map[string][]string
	Backlinks    map[string][]string
	Graph        *graph.Graph
	Mu           *sync.Mutex
	CurrentFile  string
}

func (t *LinkTransformer) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	if t == nil {
		return
	}
	sourceID := strings.TrimSuffix(t.CurrentFile, ".md")
	if m, ok := t.FileMap[t.CurrentFile]; ok && m.Render != nil && !*m.Render {
		sourceID = t.CurrentFile
	}

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			dest := string(link.Destination)
			u, err := url.Parse(dest)
			// Ignore if parse fails, or it's an absolute url (like http://...), or not a .md file
			if err != nil || u.IsAbs() || strings.HasPrefix(dest, "//") || !strings.HasSuffix(u.Path, ".md") {
				return ast.WalkContinue, nil
			}

			// Path could be root-relative or relative.
			var targetRelPath string
			if strings.HasPrefix(u.Path, "/") {
				// Root-relative link: resolve relative to the base ContentDir root.
				// Since all keys in t.FileMap are relative paths from ContentDir,
				// we just need to strip the leading slash and clean.
				targetRelPath = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(u.Path, "/")))
			} else {
				// Relative link: resolve relative to the directory of CurrentFile.
				dir := filepath.Dir(t.CurrentFile)
				targetRelPath = filepath.ToSlash(filepath.Clean(dir + "/" + u.Path))
				if dir == "." {
					targetRelPath = filepath.ToSlash(filepath.Clean(u.Path))
				}
			}

			// Prevent path traversal
			if !filepath.IsLocal(filepath.FromSlash(targetRelPath)) || strings.Contains(dest, "%2E%2E") {
				return ast.WalkContinue, nil
			}

			// Check file map
			meta, exists := t.FileMap[targetRelPath]

			targetID := strings.TrimSuffix(targetRelPath, ".md")
			if exists && meta.Render != nil && !*meta.Render {
				targetID = targetRelPath
			}

			if t.Mu != nil {
				t.Mu.Lock()
			}
			t.Graph.Edges = append(t.Graph.Edges, [2]string{sourceID, targetID})
			t.Backlinks[targetID] = append(t.Backlinks[targetID], sourceID)
			if t.Mu != nil {
				t.Mu.Unlock()
			}

			// If target exists and render is explicitly false, keep as .md
			if exists && meta.Render != nil && !*meta.Render {
				// keep it as .md, no change needed
				_ = meta
			} else {
				slug := ""
				if exists && meta != nil {
					// GetOutputURL applies IsUsableSlug itself, so the raw
					// frontmatter value can go straight through.
					slug = meta.Slug
				}

				currRender := true
				if m, ok := t.FileMap[t.CurrentFile]; ok && m.Render != nil && !*m.Render {
					currRender = false
				}
				currOut := GetOutputURL(t.CurrentFile, "", currRender)
				targetRender := true
				if m, ok := t.FileMap[targetRelPath]; ok && m.Render != nil && !*m.Render {
					targetRender = false
				}
				targetOut := GetOutputURL(targetRelPath, slug, targetRender)

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
