package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/content"
	"github.com/tbuddy/la-famille/internal/graph"
	"github.com/tbuddy/la-famille/internal/jsonutil"
	"github.com/tbuddy/la-famille/internal/page"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/stub"
	"github.com/tbuddy/la-famille/internal/transform"
)

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

	var servePort int
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start a local web server to serve the generated site",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Serve OutputDir
			dir := cfg.OutputDir
			port := servePort
			if port == 0 {
				port = cfg.Port
			}

			fmt.Printf("Serving %s on http://localhost:%d\n", dir, port)
			fmt.Printf("Press Ctrl+C to stop\n")

			return http.ListenAndServe(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(dir)))
		},
	}
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Port to run the server on (overrides config)")

	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(ragCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(serveCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config.Config) error {
	// 1. Parse templates

	// 2. Pass 1: Walk content dir and gather metadata
	fileMap, err := content.GatherMetadata(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("failed to gather metadata: %w", err)
	}

	// Track missing files that need stubs. map[missingPath][]parentFiles
	missingFiles := make(map[string][]string)
	backlinks := make(map[string][]string)
	g := graph.Graph{
		Nodes: make(map[string]graph.Node),
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
		g.Nodes[id] = graph.Node{
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
		transformer := &transform.LinkTransformer{
			CurrentFile:  relPath,
			FileMap:      fileMap,
			MissingFiles: missingFiles,
			Backlinks:    backlinks,
			Graph:        &g,
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

		page := page.Page{
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
	if err := stub.GenerateStubs(cfg, missingFiles, &g, p); err != nil {
		return err
	}

	// 5. Write JSON outputs
	for _, parents := range backlinks {
		sort.Strings(parents)
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "graph.json"), g); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "backlinks.json"), backlinks); err != nil {
		return err
	}
	if err := jsonutil.WriteJSON(filepath.Join(cfg.OutputDir, "meta.json"), metaData); err != nil {
		return err
	}

	return nil
}
