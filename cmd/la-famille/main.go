package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
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
		// Note: Validation is now done inside config.Load()
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
			return generator.Build(cfg)
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", cfg.OutputDir, "Directory for generated static site")
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
			return ragexport.RunExport(cfg)
		},
	}

	var servePort int
	var watchMode bool
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

			if watchMode {
				fmt.Println("Starting watch mode...")
				go watcher.Watch(cfg)
			}

			fmt.Printf("Serving %s on http://localhost:%d\n", dir, port)
			fmt.Printf("Press Ctrl+C to stop\n")

			return http.ListenAndServe(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(dir)))
		},
	}
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "Port to run the server on (overrides config)")
	serveCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for file changes and auto-rebuild")

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
