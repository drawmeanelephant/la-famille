package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adrg/frontmatter"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/ragexport"
)

type Page struct {
	Site            config.Config
	Title           string
	Author          string
	Date            string
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	Content         template.HTML
}

type FileMeta struct {
	RelPath         string
	Title           string
	Author          string
	Date            string
	Render          *bool
	VideoScript     string
	AnimationCues   string
	SoundtrackTheme string
	Layout          string
	Content         []byte
	Rest            []byte // The content after frontmatter
}

type Node struct {
	Type         string   `json:"type"`
	Render       bool     `json:"render"`
	Missing      bool     `json:"missing,omitempty"`
	ReferencedBy []string `json:"referenced_by,omitempty"`
}

type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges [][2]string     `json:"edges"`
}

var (
	contentDir   string
	outputDir    string
	templateFile string
)

func main() {
	// Load config first to set defaults for flags
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config.yaml: %v", err)
	}

	var rootCmd = &cobra.Command{
		Use:   "la-famille",
		Short: "La Famille is a static site generator",
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Update config from flags
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile
			return run(cfg)
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "contentDir", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "out", "o", cfg.OutputDir, "Directory for generated static site")
	buildCmd.Flags().StringVarP(&templateFile, "template", "t", cfg.Template, "Path to HTML layout template")

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.WriteDefault("config.yaml"); err != nil {
				return fmt.Errorf("failed to write config.yaml: %w", err)
			}
			fmt.Println("Created default config.yaml")
			return nil
		},
	}

	var ragCmd = &cobra.Command{
		Use:   "rag",
		Short: "Export project files into RAG-friendly markdown bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ragexport.RunExport()
		},
	}

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(ragCmd)
	rootCmd.AddCommand(prCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config.Config) error {
	// 1. Parse templates

	// 2. Pass 1: Walk content dir and gather metadata
	fileMap := make(map[string]*FileMeta)
	err := filepath.WalkDir(cfg.ContentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relPath, err := filepath.Rel(cfg.ContentDir, path)
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
			Title           string `yaml:"title"`
			Author          string `yaml:"author"`
			Date            string `yaml:"date"`
			Render          *bool  `yaml:"render"`
			VideoScript     string `yaml:"video_script"`
			AnimationCues   string `yaml:"animation_cues"`
			SoundtrackTheme string `yaml:"soundtrack_theme"`
			Layout          string `yaml:"layout"`
		}

		rest, err := frontmatter.Parse(bytes.NewReader(content), &matter)
		if err != nil {
			// If frontmatter parsing fails, treat the whole file as content
			rest = content
		}

		fileMap[relPath] = &FileMeta{
			RelPath:         relPath,
			Title:           matter.Title,
			Author:          matter.Author,
			Date:            matter.Date,
			Render:          matter.Render,
			VideoScript:     matter.VideoScript,
			AnimationCues:   matter.AnimationCues,
			SoundtrackTheme: matter.SoundtrackTheme,
			Layout:          matter.Layout,
			Content:         content,
			Rest:            rest,
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk content directory: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	graph := Graph{
		Nodes: make(map[string]Node),
		Edges: [][2]string{},
	}
	metaData := make(map[string]map[string]string)

	// 3. Pass 2: Process files in deterministic order
	var keys []string
	for k := range fileMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reusable buffer for markdown conversion
	var buf bytes.Buffer

	p := bluemonday.UGCPolicy()
	for _, relPath := range keys {
		meta := fileMap[relPath]
		shouldRender := true
		if meta.Render != nil && !*meta.Render {
			shouldRender = false
		}

		id := strings.TrimSuffix(relPath, ".md")
		graph.Nodes[id] = Node{
			Type:   "page",
			Render: shouldRender,
		}

		m := make(map[string]string)
		title := meta.Title
		if title == "" {
			title = filepath.Base(relPath)
		}
		m["title"] = title
		if meta.Author != "" {
			m["author"] = meta.Author
		}
		if meta.Date != "" {
			m["date"] = meta.Date
		}
		metaData[id] = m

		outPath := filepath.Join(cfg.OutputDir, filepath.FromSlash(relPath))
		if shouldRender {
			outPath = strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".html"
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
			Backlinks:    backlinks,
			Graph:        &graph,
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

		sanitizedHTML := p.SanitizeBytes(buf.Bytes())

		page := Page{
			Site:            cfg,
			Title:           title,
			Author:          meta.Author,
			Date:            meta.Date,
			VideoScript:     meta.VideoScript,
			AnimationCues:   meta.AnimationCues,
			SoundtrackTheme: meta.SoundtrackTheme,
			Layout:          meta.Layout,
			Content:         template.HTML(sanitizedHTML),
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		templatePath := cfg.Template
		if meta.Layout != "" {
						layoutPath := filepath.Join("templates", meta.Layout+".html")
			// If we are running tests, the templates directory is relative to the root, but the test might run from cmd/la-famille
			if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
				layoutPathFallback := filepath.Join("..", "..", "templates", meta.Layout+".html")
				if _, err2 := os.Stat(layoutPathFallback); err2 == nil {
					layoutPath = layoutPathFallback
				}
			}
			if _, err := os.Stat(layoutPath); err == nil {
				templatePath = layoutPath
			} else {
				log.Printf("Warning: layout template %s not found, falling back to %s", layoutPath, cfg.Template)
			}
		}

		pageTmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}

		if err := pageTmpl.Execute(outFile, page); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}
	// 4. Generate stubs for missing files in deterministic order
	var missingKeys []string
	for k := range missingFiles {
		missingKeys = append(missingKeys, k)
	}
	sort.Strings(missingKeys)

	for _, missingRelPath := range missingKeys {
		parents := missingFiles[missingRelPath]
		sort.Strings(parents)
		id := strings.TrimSuffix(missingRelPath, ".md")
		graph.Nodes[id] = Node{
			Type:         "stub",
			Render:       true,
			Missing:      true,
			ReferencedBy: parents,
		}

		outPath := filepath.Join(cfg.OutputDir, filepath.FromSlash(missingRelPath))
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
				htmlContent.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", html.EscapeString(relParent), html.EscapeString(parent)))
			} else {
				htmlContent.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(parent)))
			}
		}
		htmlContent.WriteString("</ul>\n")

		page := Page{
			Site:    cfg,
			Title:   "Missing Page",
			Content: template.HTML(htmlContent.String()),
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			return err
		}

		defaultTmpl, err := template.ParseFiles(cfg.Template)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to parse default template file for stubs: %w", err)
		}

		if err := defaultTmpl.Execute(outFile, page); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// 5. Write JSON outputs
	for _, parents := range backlinks {
		sort.Strings(parents)
	}
	if err := writeJSON(filepath.Join(cfg.OutputDir, "graph.json"), graph); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(cfg.OutputDir, "backlinks.json"), backlinks); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(cfg.OutputDir, "meta.json"), metaData); err != nil {
		return err
	}

	return nil
}

func writeJSON(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

type linkTransformer struct {
	CurrentFile  string // The current file being processed (e.g., docs/index.md)
	FileMap      map[string]*FileMeta
	MissingFiles map[string][]string // map[targetFile]parents
	Backlinks    map[string][]string
	Graph        *Graph
}

func (t *linkTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
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

// relPathFromTo computes the relative URL path from base (e.g. dir1/missing.md) to target (e.g. index.html)
func relPathFromTo(base, target string) (string, error) {
	baseDir := filepath.Dir(base)
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
