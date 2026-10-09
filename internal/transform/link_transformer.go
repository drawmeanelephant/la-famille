package transform

import (
	"bytes"
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
	FileMap            map[string]*content.FileMeta
	MissingFiles       map[string][]string
	MissingTitles      map[string]string
	Backlinks          map[string][]string
	WikiHeadingTargets map[string]map[string]bool
	Graph              *graph.Graph
	Mu                 *sync.Mutex
	CurrentFile        string
}

func (t *LinkTransformer) Transform(node *ast.Document, reader text.Reader, _ parser.Context) {
	if t == nil {
		return
	}
	sourceID := strings.TrimSuffix(t.CurrentFile, ".md")
	if m, ok := t.FileMap[t.CurrentFile]; ok && m.Render != nil && !*m.Render {
		sourceID = t.CurrentFile
	}

	var excludedLinks []*ast.Link
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if link, ok := n.(*ast.Link); ok {
			dest := string(link.Destination)
			if target, heading, isWikiLink := ParseWikiLinkDestination(dest); isWikiLink {
				if t.transformWikiLink(link, target, heading) {
					excludedLinks = append(excludedLinks, link)
				}
				return ast.WalkContinue, nil
			}
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
			if exists && !content.IsPublished(meta) {
				excludedLinks = append(excludedLinks, link)
				return ast.WalkContinue, nil
			}

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
				// Keep the raw-file extension, but still rewrite the path relative
				// to the generated HTML page. A page such as about/index.html that
				// links to a root-level unrendered.md needs ../unrendered.md; leaving
				// the source-relative spelling untouched produces a broken publish
				// artifact.
				currRender := true
				if m, ok := t.FileMap[t.CurrentFile]; ok && m.Render != nil && !*m.Render {
					currRender = false
				}
				currOut := GetOutputURL(t.CurrentFile, "", currRender)
				targetOut := GetOutputURL(targetRelPath, "", false)
				currDir := filepath.Dir(currOut)
				if currDir == "." {
					currDir = ""
				}
				if relOut, relErr := filepath.Rel(currDir, targetOut); relErr == nil {
					u.Path = filepath.ToSlash(relOut)
					link.Destination = []byte(u.String())
				}
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
	// Removing a node during ast.Walk detaches its next sibling and can end
	// traversal early. Keep all links in place until every target is scanned.
	for _, link := range excludedLinks {
		unwrapWikiLink(link)
	}
	if reader != nil && t.WikiHeadingTargets != nil {
		t.addWikiHeadingIDs(node, reader.Source())
	}
}

func (t *LinkTransformer) addWikiHeadingIDs(node ast.Node, source []byte) {
	wanted := t.WikiHeadingTargets[t.CurrentFile]
	if len(wanted) == 0 {
		return
	}
	ids := parser.NewContext().IDs()
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		id := string(ids.Generate(headingText(heading, source), ast.KindHeading))
		if wanted[id] {
			heading.SetAttributeString("id", id)
		}
		return ast.WalkContinue, nil
	})
}

// headingText returns the heading's full raw text. Multi-line headings (Setext
// underlines can stack several source lines into one heading) join every line
// with a single space so the generated id matches the fragment a wiki-link
// author types, e.g. [[page#Foo Bar]] -> foo-bar for "Foo\nBar\n===".
func headingText(heading *ast.Heading, source []byte) []byte {
	lines := heading.Lines()
	var text []byte
	if lines == nil {
		return text
	}
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		line := bytes.TrimSpace(segment.Value(source))
		if len(line) == 0 {
			continue
		}
		if len(text) > 0 {
			text = append(text, ' ')
		}
		text = append(text, line...)
	}
	return text
}

// transformWikiLink returns true when the caller must unwrap an excluded link
// after the AST walk completes.
func (t *LinkTransformer) transformWikiLink(link *ast.Link, target, heading string) bool {
	targetRelPath, meta, exists := ResolveWikiTarget(t.CurrentFile, target, t.FileMap)
	if !exists {
		targetRelPath = UnresolvedWikiTargetPath(t.CurrentFile, target)
	} else if !content.IsPublished(meta) {
		return true
	}

	targetID := strings.TrimSuffix(targetRelPath, ".md")
	targetRender := true
	slug := ""
	if exists && meta != nil {
		if meta.Render != nil && !*meta.Render {
			targetID = targetRelPath
			targetRender = false
		} else {
			slug = meta.Slug
		}
	}

	if t.Mu != nil {
		t.Mu.Lock()
	}
	t.Graph.Edges = append(t.Graph.Edges, [2]string{t.sourceID(), targetID})
	t.Backlinks[targetID] = append(t.Backlinks[targetID], t.sourceID())
	if !exists {
		if t.MissingFiles != nil {
			parents := t.MissingFiles[targetRelPath]
			if !containsString(parents, t.CurrentFile) {
				t.MissingFiles[targetRelPath] = append(parents, t.CurrentFile)
			}
		}
		if t.MissingTitles != nil {
			title := WikiTargetTitle(target)
			if current := t.MissingTitles[targetRelPath]; current == "" || title < current {
				t.MissingTitles[targetRelPath] = title
			}
		}
	}
	if t.Mu != nil {
		t.Mu.Unlock()
	}

	currentRender := true
	currentSlug := ""
	if currentMeta, ok := t.FileMap[t.CurrentFile]; ok && currentMeta != nil {
		currentRender = currentMeta.Render == nil || *currentMeta.Render
		currentSlug = currentMeta.Slug
	}
	currentOut := GetOutputURL(t.CurrentFile, currentSlug, currentRender)
	targetOut := GetOutputURL(targetRelPath, slug, targetRender)
	currentDir := filepath.Dir(currentOut)
	if currentDir == "." {
		currentDir = ""
	}
	relative, err := filepath.Rel(currentDir, targetOut)
	if err != nil {
		return false
	}
	if targetRender && strings.HasSuffix(filepath.ToSlash(relative), "index.html") {
		relative = strings.TrimSuffix(filepath.ToSlash(relative), "index.html")
		if relative == "index.html" {
			relative = "./"
		}
	}
	destination := url.URL{Path: filepath.ToSlash(relative)}
	destination.Fragment = WikiHeadingFragment(heading)
	link.Destination = []byte(destination.String())
	return false
}

func (t *LinkTransformer) sourceID() string {
	id := strings.TrimSuffix(t.CurrentFile, ".md")
	if meta, ok := t.FileMap[t.CurrentFile]; ok && meta != nil && meta.Render != nil && !*meta.Render {
		return t.CurrentFile
	}
	return id
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func unwrapWikiLink(link *ast.Link) {
	parent := link.Parent()
	if parent == nil {
		return
	}
	for child := link.FirstChild(); child != nil; {
		next := child.NextSibling()
		parent.InsertBefore(parent, link, child)
		child = next
	}
	parent.RemoveChild(parent, link)
}
