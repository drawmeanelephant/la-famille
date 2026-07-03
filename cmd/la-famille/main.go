package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func setupRootCmd(cfg config.Config) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "la-famille",
		Short: "La Famille is a static site generator",
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(_ *cobra.Command, _ []string) error {
			// Update config from flags
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile
			_, err := generator.Build(cfg)
			return err
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", cfg.OutputDir, "Directory for generated static site")
	buildCmd.Flags().StringVarP(&templateFile, "template", "t", cfg.Template, "Path to HTML layout template")

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
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
		RunE: func(_ *cobra.Command, _ []string) error {
			return ragexport.RunExport(cfg)
		},
	}

	var servePort int
	var watchMode bool
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start a local web server to serve the generated site",
		RunE: func(_ *cobra.Command, _ []string) error {
			dir := cfg.OutputDir
			port := servePort
			if port == 0 {
				port = cfg.Port
				if port == 0 {
					port = config.DefaultConfig().Port
				}
			}

			if watchMode {
				fmt.Println("Starting watch mode...")
				cfg.WatchMode = true
			}

			fmt.Println("Building site...")
			if _, err := generator.Build(cfg); err != nil {
				log.Printf("Initial build failed: %v", err)
			}

			// Set up a context that listens for interruption/termination signals
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()

			if watchMode {
				go func() { _ = watcher.Watch(ctx, cfg, nil) }()
			}

			mux := http.NewServeMux()
			mux.Handle("/", http.FileServer(http.Dir(dir)))

			if watchMode {
				mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
			}

			server := &http.Server{
				Addr:              fmt.Sprintf("127.0.0.1:%d", port),
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
			}

			// Run server in a background goroutine
			serverErrChan := make(chan error, 1)
			go func() {
				fmt.Printf("Serving %s on http://localhost:%d\n", dir, port)
				fmt.Printf("Press Ctrl+C to stop\n")
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					serverErrChan <- err
				}
			}()

			// Wait for either an error or a shutdown signal
			select {
			case err := <-serverErrChan:
				return err
			case <-ctx.Done():
				fmt.Println("\nShutdown signal caught. Cleaning up network handles...")
				stop() // release resources early

				// Allow up to 5 seconds for running connections to drain gracefully
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			}
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

	return rootCmd
}

func main() {
	// Load config first to set defaults for flags
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config.yaml: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	rootCmd := setupRootCmd(cfg)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
