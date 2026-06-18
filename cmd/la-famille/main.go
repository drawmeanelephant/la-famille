package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type Page struct {
	Title   string
	Author  string
	Date    string
	Content template.HTML
}

type FileMeta struct {
	RelPath string
	Title   string
	Author  string
	Date    string
	Render  *bool
	Content []byte
	Rest    []byte // The content after frontmatter
}

type GraphNode struct {
	Type         string   `json:"type"`
	Render       bool     `json:"render"`
	Missing      bool     `json:"missing,omitempty"`
	ReferencedBy []string `json:"referenced_by,omitempty"`
}

type Graph struct {
	Nodes map[string]GraphNode `json:"nodes"`
	Edges [][]string           `json:"edges"`
}

type MetaData struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Date   string `json:"date"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: la-famille build <contentDir> [outputDir]")
	}

	command := os.Args[1]

	// tests sometimes pass flags to main, ignore them and assume test path
	if strings.HasPrefix(command, "-test.") {
		if err := run("content", "templates/layout.html", "public"); err != nil {
			log.Fatal(err)
		}
		return
	}

	if command != "build" {
		log.Fatalf("Unknown command: %s", command)
	}

	contentDir := "content"
	if len(os.Args) >= 3 {
		contentDir = os.Args[2]
	}

	outputDir := "public"
	if len(os.Args) >= 4 {
		outputDir = os.Args[3]
	}

	templateFile := "templates/layout.html"

	if err := run(contentDir, templateFile, outputDir); err != nil {
		log.Fatal(err)
	}
}

func run(contentDir, templateFile, outputDir string) error {
	// 1. Parse templates
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template file: %w", err)
	}

	// 2. Pass 1: Walk content dir and gather metadata
	fileMap := make(map[string]*FileMeta)
	err = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}
		// Always use forward slashes for internal map keys to match web links
		relPath = filepath.ToSlash(relPath)

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var matter struct {
			Title  string `yaml:"title"`
			Author string `yaml:"author"`
			Date   string `yaml:"date"`
			Render *bool  `yaml:"render"`
		}

		rest, err := frontmatter.Parse(bytes.NewReader(content), &matter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = content
		}

		fileMap[relPath] = &FileMeta{
			RelPath: relPath,
			Title:   matter.Title,
			Author:  matter.Author,
			Date:    matter.Date,
			Render:  matter.Render,
			Content: content,
			Rest:    rest,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk content directory: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)

	// Reusable buffer for markdown conversion
	var buf bytes.Buffer
	var edges [][]string

	// 3. Pass 2: Process files
	for relPath, meta := range fileMap {
		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}

		outPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
		if shouldRender {
			outPath = outPath[:len(outPath)-len(filepath.Ext(outPath))] + ".html"
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		if !shouldRender {
			// Just copy the file
			if err := os.WriteFile(outPath, meta.Content, 0644); err != nil {
				return err
			}
			continue
		}

		// Set up goldmark with AST transformer
		transformer := &linkTransformer{
			CurrentFile:  relPath,
			FileMap:      fileMap,
			MissingFiles: missingFiles,
			Edges:        &edges,
		}

		md := goldmark.New(
			goldmark.WithParserOptions(
				parser.WithASTTransformers(
					util.Prioritized(transformer, 100),
				),
			),
		)

		buf.Reset()
		if err := md.Convert(meta.Rest, &buf); err != nil {
			log.Printf("Error converting %s: %v", relPath, err)
			continue
		}

		p := bluemonday.UGCPolicy()
		sanitizedHTML := p.SanitizeBytes(buf.Bytes())

		title := meta.Title
		if title == "" {
			title = filepath.Base(relPath)
		}

		page := Page{
			Title:   title,
			Author:  meta.Author,
			Date:    meta.Date,
			Content: template.HTML(sanitizedHTML),
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		if err := tmpl.Execute(outFile, page); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// 4. Generate stubs for missing files
	for missingRelPath, parents := range missingFiles {
		outPath := filepath.Join(outputDir, filepath.FromSlash(missingRelPath))
		// ensure the missing relative path has .html
		outPath = strings.TrimSuffix(outPath, ".md") + ".html"

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		// build simple HTML stub
		var htmlContent strings.Builder
		htmlContent.WriteString("<h2>This page doesn't exist yet</h2>\n")
		htmlContent.WriteString("<p>It was linked from:</p>\n<ul>\n")
		for _, parent := range parents {
			parentHtml := strings.TrimSuffix(parent, ".md") + ".html"
			// determine relative path from missing file to parent file for linking
			relParent, err := relPathFromTo(missingRelPath, parentHtml)
			if err == nil {
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", relParent, parent))
			} else {
				htmlContent.WriteString(fmt.Sprintf("<li>%s</li>\n", parent))
			}
		}
		htmlContent.WriteString("</ul>\n")

		page := Page{
			Title:   "Missing Page",
			Content: template.HTML(htmlContent.String()),
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		if err := tmpl.Execute(outFile, page); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// 5. Generate JSON artifacts
	graph := Graph{
		Nodes: make(map[string]GraphNode),
		Edges: edges,
	}
	if graph.Edges == nil {
		graph.Edges = [][]string{}
	}
	backlinks := make(map[string][]string)
	metaDataMap := make(map[string]MetaData)

	// Populate graph with existing pages
	for relPath, meta := range fileMap {
		nodeId := strings.TrimSuffix(relPath, ".md")

		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}

		graph.Nodes[nodeId] = GraphNode{
			Type:   "page",
			Render: shouldRender,
		}

		title := meta.Title
		if title == "" {
			title = filepath.Base(relPath)
			title = strings.TrimSuffix(title, ".md")
		}
		metaDataMap[nodeId] = MetaData{
			Title:  title,
			Author: meta.Author,
			Date:   meta.Date,
		}
	}

	// Populate graph with missing pages
	for target, parents := range missingFiles {
		targetId := strings.TrimSuffix(target, ".md")
		// Clean up parents to node IDs
		var parentIds []string
		for _, p := range parents {
			parentIds = append(parentIds, strings.TrimSuffix(p, ".md"))
		}

		graph.Nodes[targetId] = GraphNode{
			Type:         "stub",
			Render:       true,
			Missing:      true,
			ReferencedBy: parentIds,
		}
	}

	// Sort edges for determinism
	sort.Slice(edges, func(i, j int) bool {
		if edges[i][0] == edges[j][0] {
			return edges[i][1] < edges[j][1]
		}
		return edges[i][0] < edges[j][0]
	})

	// Populate backlinks

	for _, edge := range edges {
		src := edge[0]
		dst := edge[1]

		found := false
		for _, b := range backlinks[dst] {
			if b == src {
				found = true
				break
			}
		}
		if !found {
			backlinks[dst] = append(backlinks[dst], src)
		}
	}

	// Sort backlink arrays for determinism
	for k := range backlinks {
		sort.Strings(backlinks[k])
	}

	// Write graph.json

	writeJson(filepath.Join(outputDir, "graph.json"), graph)
	// Write backlinks.json
	writeJson(filepath.Join(outputDir, "backlinks.json"), backlinks)
	// Write meta.json
	writeJson(filepath.Join(outputDir, "meta.json"), metaDataMap)

	return nil
}

func writeJson(path string, data interface{}) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err == nil {
		os.WriteFile(path, b, 0644)
	}
}

type linkTransformer struct {
	CurrentFile  string // The current file being processed (e.g., docs/index.md)
	FileMap      map[string]*FileMeta
	MissingFiles map[string][]string // map[targetFile]parents
	Edges        *[][]string         // track all link edges
}

func (t *linkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
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

			// Check file map
			meta, exists := t.FileMap[targetRelPath]
			sourceId := strings.TrimSuffix(t.CurrentFile, ".md")
			targetId := strings.TrimSuffix(targetRelPath, ".md")
			*t.Edges = append(*t.Edges, []string{sourceId, targetId})

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

// relPathFromTo computes the relative URL path from base (e.g. dir1/missing.md) to target (e.g. index.html)
func relPathFromTo(base, target string) (string, error) {
	baseDir := filepath.Dir(base)
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
