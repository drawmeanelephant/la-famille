package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tbuddy/la-famille/internal/logger"

	"github.com/spf13/cobra"

	"github.com/tbuddy/la-famille/internal/config"
	"github.com/tbuddy/la-famille/internal/generator"
	"github.com/tbuddy/la-famille/internal/ragexport"
	"github.com/tbuddy/la-famille/internal/watcher"
)

var (
	globalLogFile string
	contentDir    string
	outputDir     string
	templateFile  string
	siteURL       string
)

func setupRootCmd(cfg config.Config) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:          "la-famille",
		Short:        "La Famille is a static site generator",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if cmd.Name() != "tui" {
				_, _ = logger.Setup(globalLogFile, false)
			}
		},
	}

	var buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Update config from flags
			cfg.ContentDir = contentDir
			cfg.OutputDir = outputDir
			cfg.Template = templateFile
			if cmd.Flags().Changed("site-url") || cmd.Flags().Changed("siteurl") {
				cfg.SiteURL = siteURL
				if err := cfg.ValidateSiteURL(); err != nil {
					return fmt.Errorf("invalid configuration: %w", err)
				}
			}
			res, err := generator.Build(cfg)
			if err != nil {
				return err
			}
			cacheStatus := "miss"
			if res.CacheHit {
				cacheStatus = "hit"
			}
			slog.Info("Build complete", "pages", res.PageCount, "duration", res.Duration, "cache", cacheStatus)
			return nil
		},
	}

	buildCmd.Flags().StringVarP(&contentDir, "content", "c", cfg.ContentDir, "Directory containing markdown files")
	buildCmd.Flags().StringVarP(&outputDir, "output", "o", cfg.OutputDir, "Directory for generated static site")
	buildCmd.Flags().StringVarP(&templateFile, "template", "t", cfg.Template, "Path to HTML layout template")
	buildCmd.Flags().StringVarP(&siteURL, "site-url", "s", cfg.SiteURL, "Public base URL of the site")
	buildCmd.Flags().StringVar(&siteURL, "siteurl", cfg.SiteURL, "Public base URL of the site (alias for --site-url)")

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := config.WriteDefault("config.yaml"); err != nil {
				return fmt.Errorf("failed to write config.yaml: %w", err)
			}
			slog.Info("Created default config.yaml")

			tmplDir := "templates"
			tmplPath := filepath.Join(tmplDir, "layout.html")
			if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
				if err := os.MkdirAll(tmplDir, 0755); err != nil {
					return fmt.Errorf("failed to create templates directory: %w", err)
				}
				defaultTmplContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
</head>
<body>
    <header>
        <h1>{{.Title}}</h1>
    </header>
    <main>
        <article>
            {{.Content}}
        </article>
    </main>
</body>
</html>
`
				if err := os.WriteFile(tmplPath, []byte(defaultTmplContent), 0600); err != nil {
					return fmt.Errorf("failed to write default template: %w", err)
				}
				slog.Info("Created default templates/layout.html")
			}
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
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()

			// Serve OutputDir
			dir := cfg.OutputDir
			port := servePort
			if port == 0 {
				port = cfg.Port
				if port == 0 {
					port = config.DefaultConfig().Port
				}
			}

			if watchMode {
				slog.Info("Starting watch mode...")
				cfg.WatchMode = true
			}

			slog.Info("Building site...")
			if _, err := generator.Build(cfg); err != nil {
				slog.Error("Initial build failed", "error", err)
				return fmt.Errorf("initial build failed: %w", err)
			}

			if watchMode {
				go func() { _ = watcher.Watch(ctx, cfg, nil) }()
			}

			slog.Info(fmt.Sprintf("Serving %s on http://localhost:%d", dir, port))
			slog.Info("Press Ctrl+C to stop")

			mux := http.NewServeMux()
			mux.Handle("/", http.FileServer(http.Dir(dir)))

			if watchMode {
				mux.HandleFunc("/livereload", watcher.LiveReloadHandler)
			}

			server := &http.Server{
				Addr:              fmt.Sprintf("127.0.0.1:%d", port),
				Handler:           mux,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       10 * time.Second,
				WriteTimeout:      10 * time.Second,
			}

			errChan := make(chan error, 1)
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errChan <- err
				}
			}()

			select {
			case err := <-errChan:
				return err
			case <-ctx.Done():
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
	rootCmd.PersistentFlags().StringVar(&globalLogFile, "log-file", "", "Path to log file (default is stderr for CLI, la-famille.log for TUI)")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(setupCheckCmd(cfg))
	rootCmd.AddCommand(setupNewCmd(cfg))
	rootCmd.AddCommand(setupAskCmd(cfg))

	return rootCmd
}

func main() {

	// Load config first to set defaults for flags
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Warn("Failed to load config.yaml", "error", err)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	rootCmd := setupRootCmd(cfg)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}
